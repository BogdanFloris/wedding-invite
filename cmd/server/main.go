package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"wedding-invite/pkg/db"
	"wedding-invite/pkg/handlers"
	"wedding-invite/pkg/middleware"
	"wedding-invite/pkg/security"

	"github.com/joho/godotenv"
)

func init() {
	// Load environment variables from .env file if it exists
	// This is primarily for local development; production should use environment variables
	envFile := filepath.Join(".", ".env")
	if _, err := os.Stat(envFile); err == nil {
		if err := godotenv.Load(envFile); err != nil {
			log.Printf("Warning: Failed to load .env file: %v", err)
		} else {
			log.Println("Loaded environment variables from .env file")
		}
	}
}

func main() {
	// Initialize database
	if err := db.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize security package
	if err := security.Initialize(); err != nil {
		log.Fatalf("Failed to initialize security: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create a mux for routing
	mux := http.NewServeMux()

	// Apply CSRF protection to all routes
	handler := middleware.CSRF(mux)

	// Public routes
	mux.Handle("/", handlers.Home())
	mux.Handle("/login", handlers.HandleLogin())
	mux.Handle("/logout", handlers.HandleLogout())

	// Protected routes
	mux.Handle("/wedding", handlers.Wedding())
	mux.Handle("/rsvp", handlers.HandleRSVP())
	mux.Handle("/rsvp/status", handlers.HandleRSVPStatus())
	mux.Handle("/rsvp/guest/", handlers.HandleDeleteGuest())

	// HTMX endpoints for the new RSVP flow
	mux.Handle("/rsvp/quick-add", handlers.HandleQuickAdd())
	mux.Handle("/rsvp/add-first", handlers.HandleAddFirst())
	mux.Handle("/rsvp/add-guest-form", handlers.HandleAddGuestForm())
	mux.Handle("/rsvp/cancel-new-guest", handlers.HandleCancelNewGuest())
	mux.Handle("/rsvp/submit", handlers.HandleSubmitRSVP())

	// Handle invitation code URL pattern (e.g., /abc123)
	// This is a special case that's handled in the Home handler

	// Serve static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
