package handlers

import (
	"net/http"
	"wedding-invite/pkg/auth"
	"wedding-invite/pkg/middleware"
	"wedding-invite/pkg/models"
	"wedding-invite/templates"
)

// Home handles the home page and login form
func Home() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// Check if there's an error message
		errorMsg := ""
		if errType := r.URL.Query().Get("error"); errType != "" {
			switch errType {
			case "invalid_email":
				errorMsg = "Invalid email address. Please check and try again."
			case "auth_required":
				errorMsg = "Please enter your email to continue."
			case "system":
				errorMsg = "System error. Please try again later."
			}
		}

		// Try to get existing session
		session, _ := auth.GetSessionFromRequest(r)
		if session != nil {
			// Already logged in, redirect to wedding info
			http.Redirect(w, r, "/wedding", http.StatusFound)
			return
		}

		// Render login page
		templates.Login(errorMsg, r).Render(r.Context(), w)
	})
}

// Wedding handles the main wedding info page
func Wedding() http.Handler {
	// Wrap with auth middleware
	return middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session from context
		session := middleware.GetSessionFromContext(r)
		if session == nil {
			http.Redirect(w, r, "/?error=auth_required", http.StatusFound)
			return
		}

		// Check if the user has any RSVPs
		email := session.InvitationEmail
		guestCount, err := models.GetGuestCount(email)
		if err != nil {
			// If there's an error, assume no guests to be safe
			guestCount = 0
		}

		hasRSVP := guestCount > 0

		// Render wedding info page
		templates.Wedding(email, hasRSVP, r).Render(r.Context(), w)
	}))
}
