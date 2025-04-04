package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
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

// renderRSVPFormContent is a helper function to render just the form content for HTMX updates
func renderRSVPFormContent(
	w http.ResponseWriter,
	r *http.Request,
	email string,
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

	// Render just the form content for HTMX updates
	templates.RSVPFormContent(email, email, guests, canAddMore, maxGuests, models.MealOptions, r).
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

		// Check for success message
		successMsg := ""
		if r.URL.Query().Get("success") == "true" {
			successMsg = "Your RSVP has been successfully submitted!"
		}

		renderRSVPForm(w, r, session.InvitationEmail, successMsg)
	}))
}

// HandleAddGuest adds a new guest and renders the updated RSVP form
func HandleAddGuest() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session from context
		session := middleware.GetSessionFromContext(r)
		if session == nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		email := session.InvitationEmail

		// Check if we can add more guests first
		canAdd, err := models.CheckCanAddGuest(email)
		if err != nil || !canAdd {
			log.Printf("Cannot add more guests: %v", err)
			http.Error(w, "Maximum number of guests reached", http.StatusBadRequest)
			return
		}

		var guestName string

		// For POST requests (form submissions), get name from form
		if r.Method == http.MethodPost {
			// Parse the form
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Invalid form data", http.StatusBadRequest)
				return
			}

			// Get guest name from form
			guestName = strings.TrimSpace(r.Form.Get("guest_name"))
		}

		// Create the guest
		_, err = models.CreateGuest(email, guestName)
		if err != nil {
			log.Printf("Error creating guest: %v", err)
			http.Error(w, "Failed to add guest", http.StatusInternalServerError)
			return
		}

		// For HTMX requests, render just the form content
		if r.Header.Get("HX-Request") == "true" {
			renderRSVPFormContent(w, r, email)
			return
		}

		// For regular requests, redirect to prevent form resubmission
		timestamp := time.Now().Unix()
		redirectURL := fmt.Sprintf("/rsvp?t=%d", timestamp)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}))
}

// HandleDeleteGuest removes a guest and renders the updated RSVP form
func HandleDeleteGuest() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session from context
		session := middleware.GetSessionFromContext(r)
		if session == nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		email := session.InvitationEmail

		// Extract guest ID from URL
		// Expected format: /rsvp/guest/123
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 4 {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		guestIDStr := pathParts[len(pathParts)-1]
		guestID, err := strconv.ParseInt(guestIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid guest ID", http.StatusBadRequest)
			return
		}

		// Delete the guest
		err = models.DeleteGuest(guestID, email)
		if err != nil {
			log.Printf("Error deleting guest: %v", err)
			http.Error(w, "Failed to delete guest", http.StatusInternalServerError)
			return
		}

		// For HTMX requests, render just the form content
		if r.Header.Get("HX-Request") == "true" {
			renderRSVPFormContent(w, r, email)
			return
		}

		// For regular requests, redirect to prevent form resubmission
		timestamp := time.Now().Unix()
		redirectURL := fmt.Sprintf("/rsvp?t=%d", timestamp)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
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

		// Process existing guests
		guestIDs := r.Form["guest_ids[]"]
		
		// If there are no guests but the user has made a selection (not attending)
		// create a record to track their decision
		if len(guestIDs) == 0 {
			// Only create an entry if declining (attending=false)
			// For attending=true, we should have at least one guest
			if !partyAttending {
				err := models.RecordAttendanceStatus(email, partyAttending)
				if err != nil {
					log.Printf("Error recording attendance status: %v", err)
				}
			} else {
				log.Printf("Warning: User selected 'attending' but didn't add any guests")
			}
		} else {
			// Process existing guests
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

		// Return success message
		templates.SuccessMessage("your party", r).Render(r.Context(), w)
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

		// Render RSVP status page
		templates.RSVPStatus(email, guests, r).Render(r.Context(), w)
	}))
}
