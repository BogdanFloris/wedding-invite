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

		// Handle form submission
		if r.Method == http.MethodPost {
			handleRSVPFormSubmission(w, r, invitationID)
			return
		}

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

// HandleAddGuestFields returns the HTML for adding a new guest (HTMX endpoint)
func HandleAddGuestFields() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		templates.NewGuestFields().Render(r.Context(), w)
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

		// For HTMX, return empty response to remove the element
		w.WriteHeader(http.StatusOK)
	}))
}

// HandleRSVPFormSubmission processes the RSVP form data
func handleRSVPFormSubmission(w http.ResponseWriter, r *http.Request, invitationID string) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Process based on action type
	action := r.Form.Get("action")
	log.Printf("Processing RSVP form action: %s", action)

	switch action {
	case "add_guest":
		handleAddGuest(w, r, invitationID)
	case "update_rsvp":
		handleUpdateRSVP(w, r, invitationID)
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
	}
}

// handleAddGuest processes the addition of a new guest
func handleAddGuest(w http.ResponseWriter, r *http.Request, invitationID string) {
	// Get guest name from form
	guestName := strings.TrimSpace(r.Form.Get("guest_name"))
	if guestName == "" {
		// Try the new form field name
		guestName = strings.TrimSpace(r.Form.Get("new_guest_name"))
		if guestName == "" {
			http.Error(w, "Guest name is required", http.StatusBadRequest)
			return
		}
	}

	// Check if another guest can be added
	canAdd, err := models.CheckCanAddGuest(invitationID)
	if err != nil {
		log.Printf("Error checking guest limit: %v", err)
		http.Error(w, "Failed to check guest limit", http.StatusInternalServerError)
		return
	}

	if !canAdd {
		http.Error(w, "Maximum number of guests reached", http.StatusBadRequest)
		return
	}

	// Create the new guest
	guestID, err := models.CreateGuest(invitationID, guestName)
	if err != nil {
		log.Printf("Error creating guest: %v", err)
		http.Error(w, "Failed to add guest", http.StatusInternalServerError)
		return
	}

	// If meal preference and dietary restrictions were provided, update them
	mealPreference := r.Form.Get("new_guest_meal")
	dietaryRestrictions := r.Form.Get("new_guest_dietary")

	if mealPreference != "" || dietaryRestrictions != "" {
		attending := true // Default to attending for newly added guests
		err := models.UpdateGuestRSVP(guestID, attending, mealPreference, dietaryRestrictions)
		if err != nil {
			log.Printf("Error updating new guest preferences: %v", err)
			// Continue despite error
		}
	}

	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// Get the guest that was just created
		guest, err := models.GetGuest(guestID)
		if err != nil {
			http.Error(w, "Failed to retrieve guest data", http.StatusInternalServerError)
			return
		}

		// Return the guest form HTML for the newly created guest
		templates.GuestForm(*guest, models.MealOptions).Render(r.Context(), w)
		return
	}

	// Regular form submission - redirect back to RSVP page
	http.Redirect(w, r, "/rsvp", http.StatusFound)
}

// handleUpdateRSVP processes RSVP updates for all guests
func handleUpdateRSVP(w http.ResponseWriter, r *http.Request, invitationID string) {
	// Get party attendance status from form
	partyAttendingStr := r.Form.Get("party_attending")
	if partyAttendingStr == "" {
		http.Error(w, "Attendance status is required", http.StatusBadRequest)
		return
	}

	// Convert to bool
	partyAttending := partyAttendingStr == "yes"

	// Get all guests for this invitation
	guests, err := models.GetGuestsByInvitation(invitationID)
	if err != nil {
		log.Printf("Error fetching guests: %v", err)
		http.Error(w, "Failed to load guest data", http.StatusInternalServerError)
		return
	}

	// Process each guest's RSVP
	for _, guest := range guests {
		// Get form field values (using hidden guest_id fields)
		guestIDField := fmt.Sprintf("guest_id_%d", guest.ID)
		if r.Form.Get(guestIDField) == "" {
			continue // Skip if this guest is not in the form
		}

		mealPreference := r.Form.Get(fmt.Sprintf("meal_%d", guest.ID))
		dietaryRestrictions := r.Form.Get(fmt.Sprintf("dietary_%d", guest.ID))

		// Update guest RSVP - using the party attending status for all guests
		err := models.UpdateGuestRSVP(guest.ID, partyAttending, mealPreference, dietaryRestrictions)
		if err != nil {
			log.Printf("Error updating RSVP for guest %d: %v", guest.ID, err)
			// Continue processing other guests despite error
		}
	}

	// Check for any new guests being added in the same form
	for i := 0; ; i++ {
		nameField := fmt.Sprintf("new_guest_name_%d", i)
		name := r.Form.Get(nameField)
		if name == "" {
			break // No more new guests
		}

		mealField := fmt.Sprintf("new_guest_meal_%d", i)
		dietaryField := fmt.Sprintf("new_guest_dietary_%d", i)

		mealPreference := r.Form.Get(mealField)
		dietaryRestrictions := r.Form.Get(dietaryField)

		// Create and update the new guest
		guestID, err := models.CreateGuest(invitationID, name)
		if err != nil {
			log.Printf("Error creating new guest: %v", err)
			continue
		}

		err = models.UpdateGuestRSVP(guestID, partyAttending, mealPreference, dietaryRestrictions)
		if err != nil {
			log.Printf("Error updating new guest RSVP: %v", err)
		}
	}

	// Redirect with success message
	http.Redirect(w, r, "/rsvp?success=true", http.StatusFound)
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
