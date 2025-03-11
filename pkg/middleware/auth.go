package middleware

import (
	"context"
	"log"
	"net/http"
	"os"
	
	"wedding-invite/pkg/auth"
)

// SessionKey is the key used to store the session in the request context
type contextKey string
const SessionKey contextKey = "session"

// Authentication middleware checks if user is authenticated
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get session from request
		session, err := auth.GetSessionFromRequest(r)
		if err != nil {
			log.Printf("Auth required but not authenticated: %v", err)
			http.Redirect(w, r, "/?error=auth_required", http.StatusFound)
			return
		}
		
		// Add session to request context
		ctx := context.WithValue(r.Context(), SessionKey, session)
		
		// Call the next handler with the session in context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetSessionFromContext retrieves the session from the request context
func GetSessionFromContext(r *http.Request) *auth.Session {
	session, _ := r.Context().Value(SessionKey).(*auth.Session)
	return session
}

// CSRF protection middleware
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check POST, PUT, DELETE methods
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		
		// Skip CSRF check in development environment
		if isDevelopment() {
			next.ServeHTTP(w, r)
			return
		}
		
		// Check Origin and Referer headers
		origin := r.Header.Get("Origin")
		referer := r.Header.Get("Referer")
		
		// Get host from request
		host := r.Host
		
		// Simple CSRF check: validate Origin/Referer contain our host
		validOrigin := origin == "" || origin == "https://"+host || origin == "http://"+host
		validReferer := referer == "" || 
		                 (len(referer) >= len(host) && 
		                  (referer[len(referer)-len(host):] == host || 
		                   referer[len(referer)-len(host)-1:] == host+"/"))
		
		if !validOrigin || !validReferer {
			log.Printf("CSRF check failed: Origin=%s, Referer=%s, Host=%s", origin, referer, host)
			http.Error(w, "CSRF check failed", http.StatusForbidden)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// isDevelopment checks if we're running in a development environment
func isDevelopment() bool {
	// Only check the environment variable for simplicity and reliability
	env := os.Getenv("ENVIRONMENT")
	return env == "development" || env == "dev"
}
