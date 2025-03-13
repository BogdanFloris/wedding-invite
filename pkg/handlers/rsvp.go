package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"wedding-invite/pkg/middleware"
	"wedding-invite/pkg/models"
	"wedding-invite/templates"
)

// HandleRSVP displays and processes the RSVP form
func HandleRSVP() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session from context
		session := middleware.GetSessionFromContext(r)
		if session == nil {
			http.Redirect(w, r, "/?error=auth_required", http.StatusFound)
			return
		}

		invitationID := session.InvitationID

		// Get guest data
		guests, err := models.GetGuestsByInvitation(invitationID)
		if err != nil {
			log.Printf("Error fetching guests: %v", err)
			http.Error(w, "Failed to load guest data", http.StatusInternalServerError)
			return
		}

		// Get family name
		familyName, err := models.GetInvitationName(invitationID)
		if err != nil {
			log.Printf("Error fetching family name: %v", err)
			http.Error(w, "Failed to load invitation data", http.StatusInternalServerError)
			return
		}

		// Check if more guests can be added
		canAddMore, err := models.CheckCanAddGuest(invitationID)
		if err != nil {
			log.Printf("Error checking guest limit: %v", err)
			canAddMore = false // Default to false on error
		}

		// Get max guests
		maxGuests, err := models.GetMaxGuestCount(invitationID)
		if err != nil {
			log.Printf("Error fetching max guests: %v", err)
			maxGuests = len(guests) // Default to current count on error
		}

		// Check for success message
		successMsg := ""
		if r.URL.Query().Get("success") == "true" {
			successMsg = "Your RSVP has been successfully submitted!"
		}

		// Render RSVP form
		templates.RSVPForm(familyName, invitationID, guests, canAddMore, maxGuests, models.MealOptions, successMsg).
			Render(r.Context(), w)
	}))
}

// HandleQuickAdd shows the quick add form for the first guest
func HandleQuickAdd() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Render the quick add form
		templates.QuickAddForm().Render(r.Context(), w)
	}))
}

// HandleAddFirst adds the first guest and shows the main RSVP form
func HandleAddFirst() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session from context
		session := middleware.GetSessionFromContext(r)
		if session == nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		invitationID := session.InvitationID

		// Parse the form
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// Get guest name
		guestName := strings.TrimSpace(r.Form.Get("guest_name"))
		if guestName == "" {
			http.Error(w, "Guest name is required", http.StatusBadRequest)
			return
		}

		// Create the first guest
		_, err := models.CreateGuest(invitationID, guestName)
		if err != nil {
			log.Printf("Error creating first guest: %v", err)
			http.Error(w, "Failed to add guest", http.StatusInternalServerError)
			return
		}

		// Get updated guest data
		guests, err := models.GetGuestsByInvitation(invitationID)
		if err != nil {
			log.Printf("Error fetching guests: %v", err)
			http.Error(w, "Failed to load guest data", http.StatusInternalServerError)
			return
		}

		// Get family name
		familyName, err := models.GetInvitationName(invitationID)
		if err != nil {
			log.Printf("Error fetching family name: %v", err)
			http.Error(w, "Failed to load invitation data", http.StatusInternalServerError)
			return
		}

		// Check if more guests can be added
		canAddMore, err := models.CheckCanAddGuest(invitationID)
		if err != nil {
			log.Printf("Error checking guest limit: %v", err)
			canAddMore = false
		}

		// Get max guests
		maxGuests, err := models.GetMaxGuestCount(invitationID)
		if err != nil {
			log.Printf("Error fetching max guests: %v", err)
			maxGuests = len(guests)
		}

		// Render the main RSVP form content
		templates.RsvpFormContent(familyName, invitationID, guests, canAddMore, maxGuests, models.MealOptions).
			Render(r.Context(), w)
	}))
}

// HandleAddGuestForm returns the form for adding a new guest
func HandleAddGuestForm() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the current guest count
		countStr := r.URL.Query().Get("count")
		count, err := strconv.Atoi(countStr)
		if err != nil {
			count = 0 // Default if not provided
		}

		// Render the new guest form
		templates.NewGuestForm(count, models.MealOptions).Render(r.Context(), w)
	}))
}

// HandleDeleteGuest handles HTMX delete requests for guests
func HandleDeleteGuest() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session from context
		session := middleware.GetSessionFromContext(r)
		if session == nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		invitationID := session.InvitationID

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
		err = models.DeleteGuest(guestID, invitationID)
		if err != nil {
			log.Printf("Error deleting guest: %v", err)
			http.Error(w, "Failed to delete guest", http.StatusInternalServerError)
			return
		}

		// Return the updated form content
		guests, err := models.GetGuestsByInvitation(invitationID)
		if err != nil {
			log.Printf("Error fetching guests after delete: %v", err)
			http.Error(w, "Failed to load guest data", http.StatusInternalServerError)
			return
		}

		// Get family name
		familyName, err := models.GetInvitationName(invitationID)
		if err != nil {
			log.Printf("Error fetching family name: %v", err)
			http.Error(w, "Failed to load invitation data", http.StatusInternalServerError)
			return
		}

		// Check if more guests can be added
		canAddMore, err := models.CheckCanAddGuest(invitationID)
		if err != nil {
			log.Printf("Error checking guest limit: %v", err)
			canAddMore = false
		}

		// Get max guests
		maxGuests, err := models.GetMaxGuestCount(invitationID)
		if err != nil {
			log.Printf("Error fetching max guests: %v", err)
			maxGuests = len(guests)
		}

		// Render updated RSVP form
		templates.RsvpFormContent(familyName, invitationID, guests, canAddMore, maxGuests, models.MealOptions).
			Render(r.Context(), w)
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

		invitationID := session.InvitationID

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
				err := models.UpdateGuestName(guestID, invitationID, guestName)
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

		// Process any new guests
		for i := 0; i < 10; i++ { // Limit to reasonable number to prevent abuse
			nameField := fmt.Sprintf("new_guest_name_%d", i)
			name := r.Form.Get(nameField)

			if name == "" {
				continue // No more new guests
			}

			mealField := fmt.Sprintf("new_guest_meal_%d", i)
			dietaryField := fmt.Sprintf("new_guest_dietary_%d", i)

			mealPreference := r.Form.Get(mealField)
			dietaryRestrictions := r.Form.Get(dietaryField)

			// Create the new guest
			guestID, err := models.CreateGuest(invitationID, name)
			if err != nil {
				log.Printf("Error creating new guest: %v", err)
				continue
			}

			// Update the new guest's RSVP status
			err = models.UpdateGuestRSVP(
				guestID,
				partyAttending,
				mealPreference,
				dietaryRestrictions,
			)
			if err != nil {
				log.Printf("Error updating new guest RSVP: %v", err)
			}
		}

		// Get family name for success message
		familyName, err := models.GetInvitationName(invitationID)
		if err != nil {
			log.Printf("Error fetching family name: %v", err)
			familyName = "your party" // Default if not found
		}

		// Return success message
		templates.SuccessMessage(familyName).Render(r.Context(), w)
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

		invitationID := session.InvitationID

		// Get guest data
		guests, err := models.GetGuestsByInvitation(invitationID)
		if err != nil {
			log.Printf("Error fetching guests: %v", err)
			http.Error(w, "Failed to load guest data", http.StatusInternalServerError)
			return
		}

		// Get family name
		familyName, err := models.GetInvitationName(invitationID)
		if err != nil {
			log.Printf("Error fetching family name: %v", err)
			http.Error(w, "Failed to load invitation data", http.StatusInternalServerError)
			return
		}

		// Render RSVP status page
		templates.RSVPStatus(familyName, guests).Render(r.Context(), w)
	}))
}

// HandleCancelNewGuest handles canceling the addition of a new guest
func HandleCancelNewGuest() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Nothing needs to be done since we're using hx-swap="delete"
		// Just return a 200 OK status
		w.WriteHeader(http.StatusOK)
	}))
}
