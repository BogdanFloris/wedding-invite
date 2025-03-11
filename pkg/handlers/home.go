package handlers

import (
	"net/http"
	"strings"
	
	"wedding-invite/pkg/auth"
	"wedding-invite/pkg/middleware"
	"wedding-invite/templates"
)

// Home handles the home page and login form
func Home() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			// If not the root path, try to handle it as an invitation code
			// Only process paths that look like invitation codes (no slashes in path)
			if !strings.Contains(r.URL.Path[1:], "/") {
				HandleInviteCode().ServeHTTP(w, r)
				return
			}
			
			http.NotFound(w, r)
			return
		}
		
		// Check if there's an error message
		errorMsg := ""
		if errType := r.URL.Query().Get("error"); errType != "" {
			switch errType {
			case "invalid_code":
				errorMsg = "Invalid invitation code. Please check and try again."
			case "rate_limit":
				errorMsg = "Too many attempts. Please try again later."
			case "auth_required":
				errorMsg = "Please enter your invitation code to continue."
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
		templates.Login(errorMsg).Render(r.Context(), w)
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
		
		// Render wedding info page
		templates.Wedding(session.InvitationID).Render(r.Context(), w)
	}))
}