package handlers

import (
	"log"
	"net/http"

	"wedding-invite/pkg/middleware"
	"wedding-invite/pkg/models"
	"wedding-invite/templates"
)

// HandleAdminGuests displays all guests in the database
func HandleAdminGuests() http.Handler {
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get all guests
		guests, err := models.GetAllGuests()
		if err != nil {
			log.Printf("Error fetching all guests: %v", err)
			http.Error(w, "Failed to load guest data", http.StatusInternalServerError)
			return
		}

		// Render admin guests page
		templates.AdminGuests(guests, r).Render(r.Context(), w)
	}))
}
