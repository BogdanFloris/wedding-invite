package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"wedding-invite/pkg/middleware"
	"wedding-invite/pkg/models"
	"wedding-invite/templates"
)

// renderRSVPForm is a helper function to render the full RSVP form with the latest data
func renderRSVPForm(
	w http.ResponseWriter,
	r *http.Request,
	email string,
	successMsg string,
) {
	// Get guest data
	guests, err := models.GetGuestsByInvitation(email)
	if err != nil {
		log.Printf("Error fetching guests: %v", err)
		http.Error(w, "Failed to load guest data", http.StatusInternalServerError)
		return
	}

	// Check if more guests can be added
	canAddMore, err := models.CheckCanAddGuest(email)
	if err != nil {
		log.Printf("Error checking guest limit: %v", err)
		canAddMore = false // Default to false on error
	}

	// Get max guests
	maxGuests, err := models.GetMaxGuestCount(email)
	if err != nil {
		log.Printf("Error fetching max guests: %v", err)
		maxGuests = len(guests) // Default to current count on error
	}

	// Render RSVP form
	templates.RSVPForm(email, email, guests, canAddMore, maxGuests, models.MealOptions, successMsg, r).
		Render(r.Context(), w)
}

// HandleRSVP displays the RSVP form
func HandleRSVP() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session from context
		session := middleware.GetSessionFromContext(r)
		if session == nil {
			http.Redirect(w, r, "/?error=auth_required", http.StatusFound)
			return
		}

		// Check for "Primary Contact" auto-generated entries and remove them
		// so user starts with a clean form when editing
		err := models.RemovePrimaryContactGuest(session.InvitationEmail)
		if err != nil {
			log.Printf("Error removing primary contact entry: %v", err)
			// Continue anyway - non-critical error
		}

		// Check for success message
		successMsg := ""
		if r.URL.Query().Get("success") == "true" {
			successMsg = "Your RSVP has been successfully submitted!"
		}

		renderRSVPForm(w, r, session.InvitationEmail, successMsg)
	}))
}

// HandleSubmitRSVP processes the RSVP form submission
func HandleSubmitRSVP() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session from context
		session := middleware.GetSessionFromContext(r)
		if session == nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		email := session.InvitationEmail

		// Parse form data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// Get party attending status
		partyAttendingStr := r.Form.Get("party_attending")
		if partyAttendingStr == "" {
			http.Error(w, "Attendance status is required", http.StatusBadRequest)
			return
		}

		// Convert to bool
		partyAttending := partyAttendingStr == "yes"

		// Process all guests from form
		guestIDs := r.Form["guest_ids[]"]

		// First, find all existing guests in database regardless of whether guestIDs are present
		existingGuests, err := models.GetGuestsByInvitation(email)
		if err != nil {
			log.Printf("Error getting existing guests: %v", err)
			// Continue anyway
			existingGuests = []models.Guest{}
		}

		// For "not attending" mode with no guests in form
		if !partyAttending && len(guestIDs) == 0 {
			// First, remove any existing Primary Contact entries
			err := models.RemovePrimaryContactGuest(email)
			if err != nil {
				log.Printf("Error removing primary contact entries: %v", err)
			}

			// If we have real guests (not Primary Contact), update them to not attending
			var hasNonPrimaryGuests bool
			for _, guest := range existingGuests {
				if guest.Name != "Primary Contact" {
					hasNonPrimaryGuests = true
					// Update this guest to not attending
					err = models.UpdateGuestRSVP(
						guest.ID,
						false, // not attending
						"",    // clear meal preference
						"",    // clear dietary restrictions
					)
					if err != nil {
						log.Printf(
							"Error updating RSVP for guest %d to not attending: %v",
							guest.ID,
							err,
						)
					}
				}
			}

			// If no regular guests exist, create a Primary Contact entry
			if !hasNonPrimaryGuests {
				err := models.RecordAttendanceStatus(email, false)
				if err != nil {
					log.Printf("Error recording attendance status: %v", err)
				}
			}
		} else if partyAttending && len(guestIDs) == 0 {
			// User selected "attending" but didn't add any guests - error case
			log.Printf("Warning: User selected 'attending' but didn't add any guests")
		} else {
			// Normal case: process guest data from form

			// Build a map of existing guest IDs to check for removals
			existingGuestMap := make(map[int64]bool)
			for _, guestIDStr := range guestIDs {
				guestID, err := strconv.ParseInt(guestIDStr, 10, 64)
				if err != nil {
					continue
				}

				// Only track positive IDs (real DB guests)
				if guestID > 0 {
					existingGuestMap[guestID] = true
				}
			}

			// Remove Primary Contact entries as we'll be working with real guest data
			err := models.RemovePrimaryContactGuest(email)
			if err != nil {
				log.Printf("Error removing primary contact entries: %v", err)
			}

			// Delete any guests that were removed in the UI
			for _, guest := range existingGuests {
				if !existingGuestMap[guest.ID] {
					// This guest is in DB but not in the form, so delete it
					err := models.DeleteGuest(guest.ID, email)
					if err != nil {
						log.Printf("Error deleting removed guest %d: %v", guest.ID, err)
					}
				}
			}

			// Now process the guests in the form
			for _, guestIDStr := range guestIDs {
				guestID, err := strconv.ParseInt(guestIDStr, 10, 64)
				if err != nil {
					log.Printf("Invalid guest ID %s: %v", guestIDStr, err)
					continue
				}

				// Get form field values
				guestName := r.Form.Get(fmt.Sprintf("guest_name_%d", guestID))
				mealPreference := r.Form.Get(fmt.Sprintf("guest_meal_%d", guestID))
				dietaryRestrictions := r.Form.Get(fmt.Sprintf("guest_dietary_%d", guestID))

				// If the ID is negative, this is a temporary guest that needs to be created
				if guestID < 0 {
					// Create a new guest in the database
					newGuestID, err := models.CreateGuest(email, guestName)
					if err != nil {
						log.Printf("Error creating guest from temp ID %d: %v", guestID, err)
						continue
					}

					// Update the new guest's RSVP status
					err = models.UpdateGuestRSVP(
						newGuestID,
						partyAttending,
						mealPreference,
						dietaryRestrictions,
					)
					if err != nil {
						log.Printf("Error updating RSVP for new guest %d: %v", newGuestID, err)
					}
				} else {
					// This is an existing guest from the database

					// Update guest name if needed
					if guestName != "" {
						err := models.UpdateGuestName(guestID, email, guestName)
						if err != nil {
							log.Printf("Error updating guest name for guest %d: %v", guestID, err)
						}
					}

					// Update guest RSVP status
					err = models.UpdateGuestRSVP(
						guestID,
						partyAttending,
						mealPreference,
						dietaryRestrictions,
					)
					if err != nil {
						log.Printf("Error updating RSVP for guest %d: %v", guestID, err)
					}
				}
			}
		}

		// Return success message with the email address
		templates.SuccessMessage(email, r).Render(r.Context(), w)
	}))
}

// HandleRSVPStatus shows the current RSVP status
func HandleRSVPStatus() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session from context
		session := middleware.GetSessionFromContext(r)
		if session == nil {
			http.Redirect(w, r, "/?error=auth_required", http.StatusFound)
			return
		}

		email := session.InvitationEmail

		// Get guest data
		guests, err := models.GetGuestsByInvitation(email)
		if err != nil {
			log.Printf("Error fetching guests: %v", err)
			http.Error(w, "Failed to load guest data", http.StatusInternalServerError)
			return
		}

		// Check if we have only the "Primary Contact" auto entry
		hasPrimaryContactOnly := false
		if len(guests) == 1 && guests[0].Name == "Primary Contact" {
			hasPrimaryContactOnly = true
		}

		// Render RSVP status page with the flag
		templates.RSVPStatus(email, guests, hasPrimaryContactOnly, r).Render(r.Context(), w)
	}))
}

