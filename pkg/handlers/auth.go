package handlers

import (
	"log"
	"net/http"
	"wedding-invite/pkg/auth"
)

// HandleInviteCode processes an invitation code directly from the URL
func HandleInviteCode() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract code from URL path
		code := r.URL.Path[1:] // Remove leading slash

		// Validate the code
		invitation, err := auth.ValidateInvitationCode(code, r)
		if err != nil {
			log.Printf("Invalid invitation code from URL: %v", err)
			http.Redirect(w, r, "/?error=invalid_code", http.StatusFound)
			return
		}

		// Create a session
		session, err := auth.CreateSession(invitation, r)
		if err != nil {
			log.Printf("Error creating session: %v", err)
			http.Redirect(w, r, "/?error=system", http.StatusFound)
			return
		}

		// Set session cookie
		auth.SetSessionCookie(w, session)

		// Redirect to wedding info page
		http.Redirect(w, r, "/wedding", http.StatusFound)
	})
}

// HandleLogin handles the login form submission
func HandleLogin() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST
		if r.Method != http.MethodPost {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Parse form
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// Get the invitation code
		code := r.Form.Get("code")

		// Validate the code
		invitation, err := auth.ValidateInvitationCode(code, r)
		if err != nil {
			// Determine error type and redirect accordingly
			switch err {
			case auth.ErrInvalidCode:
				http.Redirect(w, r, "/?error=invalid_code", http.StatusFound)
			default:
				http.Redirect(w, r, "/?error=system", http.StatusFound)
			}
			return
		}

		// Create a session
		session, err := auth.CreateSession(invitation, r)
		if err != nil {
			log.Printf("Error creating session: %v", err)
			http.Redirect(w, r, "/?error=system", http.StatusFound)
			return
		}

		// Set session cookie
		auth.SetSessionCookie(w, session)

		// Redirect to wedding info page
		http.Redirect(w, r, "/wedding", http.StatusFound)
	})
}

// HandleLogout logs the user out
func HandleLogout() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clear the session cookie
		auth.ClearSessionCookie(w)

		// Redirect to home page
		http.Redirect(w, r, "/", http.StatusFound)
	})
}
