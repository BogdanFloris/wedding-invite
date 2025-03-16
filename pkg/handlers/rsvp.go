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

// renderRSVPForm is a helper function to render the RSVP form with the latest data
func renderRSVPForm(
	w http.ResponseWriter,
	r *http.Request,
	invitationID string,
	successMsg string,
) {
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

	// Render RSVP form
	templates.RSVPForm(familyName, invitationID, guests, canAddMore, maxGuests, models.MealOptions, successMsg, r).
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

		renderRSVPForm(w, r, session.InvitationID, successMsg)
	}))
}

// HandleAddGuest adds a new guest and redirects back to the RSVP page
func HandleAddGuest() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session from context
		session := middleware.GetSessionFromContext(r)
		if session == nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		invitationID := session.InvitationID

		// Check if we can add more guests first
		canAdd, err := models.CheckCanAddGuest(invitationID)
		if err != nil || !canAdd {
			log.Printf("Cannot add more guests: %v", err)
			http.Error(w, "Maximum number of guests reached", http.StatusBadRequest)
			return
		}

		var guestName string

		// For GET requests (direct link clicks), use default name
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

		// If no name was provided, leave it blank and let user fill it in
		// Empty guest names will be displayed with a placeholder in the UI

		// Create the guest
		_, err = models.CreateGuest(invitationID, guestName)
		if err != nil {
			log.Printf("Error creating guest: %v", err)
			http.Error(w, "Failed to add guest", http.StatusInternalServerError)
			return
		}

		// Redirect to the RSVP page with a timestamp to prevent caching
		timestamp := time.Now().Unix()
		redirectURL := fmt.Sprintf("/rsvp?t=%d", timestamp)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}))
}

// HandleDeleteGuest removes a guest and redirects back to the RSVP page
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

		// For standard GET requests (direct clicks on links)
		// For DELETE requests (from HTMX)
		// Both should be handled the same way

		// Delete the guest
		err = models.DeleteGuest(guestID, invitationID)
		if err != nil {
			log.Printf("Error deleting guest: %v", err)
			http.Error(w, "Failed to delete guest", http.StatusInternalServerError)
			return
		}

		// Redirect to the RSVP page with a specific GET parameter to force reload
		// Adding a timestamp to bust any cache
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

		// Get family name for success message
		familyName, err := models.GetInvitationName(invitationID)
		if err != nil {
			log.Printf("Error fetching family name: %v", err)
			familyName = "your party" // Default if not found
		}

		// Return success message
		templates.SuccessMessage(familyName, r).Render(r.Context(), w)
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
		templates.RSVPStatus(familyName, guests, r).Render(r.Context(), w)
	}))
}
