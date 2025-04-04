package auth

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"wedding-invite/pkg/db"
	"wedding-invite/pkg/security"
)

const (
	// SessionCookieName is the name of the cookie used for auth
	SessionCookieName = "wedding_session"

	// SessionDuration is how long a session lasts (30 days)
	SessionDuration = 30 * 24 * time.Hour
)

// Errors
var (
	ErrInvalidEmail   = errors.New("invalid email address")
	ErrSessionExpired = errors.New("session expired")
	ErrInternalError  = errors.New("an internal error occurred")
)

// Session represents an authenticated session
type Session struct {
	ID              string
	InvitationEmail string
	CreatedAt       time.Time
	ExpiresAt       time.Time
}

// Invitation represents invitation details
type Invitation struct {
	Email      string
	MaxGuests  int
	Phone      sql.NullString
	CreatedAt  time.Time
	LastAccess sql.NullTime
	Approved   bool
}

// ValidateEmail checks if an email is valid and creates a new invitation if it doesn't exist
func ValidateEmail(email string, r *http.Request) (*Invitation, error) {
	// Clean the email (remove spaces, convert to lowercase)
	email = strings.TrimSpace(strings.ToLower(email))

	// Basic email validation
	if len(email) < 5 || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return nil, ErrInvalidEmail
	}

	// Query the database for the invitation
	var invitation Invitation
	err := db.DB.QueryRow(`
		SELECT email, max_guests, phone, created_at, last_access, approved
		FROM invitations
		WHERE email = ?
	`, email).Scan(
		&invitation.Email,
		&invitation.MaxGuests,
		&invitation.Phone,
		&invitation.CreatedAt,
		&invitation.LastAccess,
		&invitation.Approved,
	)

	// If email not found, create a new invitation
	if err == sql.ErrNoRows {
		// Get IP address for registration tracking
		ipAddress := getIP(r)

		// Create new invitation
		_, err = db.DB.Exec(`
			INSERT INTO invitations (email, registration_ip, max_guests)
			VALUES (?, ?, 6)
		`, email, ipAddress)
		if err != nil {
			log.Printf("Error creating new invitation: %v", err)
			return nil, ErrInternalError
		}

		// Now retrieve the newly created invitation
		err = db.DB.QueryRow(`
			SELECT email, max_guests, phone, created_at, last_access, approved
			FROM invitations
			WHERE email = ?
		`, email).Scan(
			&invitation.Email,
			&invitation.MaxGuests,
			&invitation.Phone,
			&invitation.CreatedAt,
			&invitation.LastAccess,
			&invitation.Approved,
		)
		if err != nil {
			log.Printf("Error retrieving new invitation: %v", err)
			return nil, ErrInternalError
		}
	} else if err != nil {
		log.Printf("Database error validating email: %v", err)
		return nil, ErrInternalError
	}

	// Update last access time
	_, err = db.DB.Exec(`
		UPDATE invitations
		SET last_access = CURRENT_TIMESTAMP
		WHERE email = ?
	`, email)
	if err != nil {
		log.Printf("Error updating last access time: %v", err)
		// Non-fatal error, continue with authentication
	}

	return &invitation, nil
}

// CreateSession creates a new session for a valid invitation
func CreateSession(invitation *Invitation, r *http.Request) (*Session, error) {
	// Generate session ID
	sessionID, err := security.GenerateSessionID()
	if err != nil {
		return nil, err
	}

	// Calculate expiry time
	now := time.Now()
	expiresAt := now.Add(SessionDuration)

	// Create session in database
	ipHash := security.HashIPAddress(getIP(r))
	_, err = db.DB.Exec(`
		INSERT INTO sessions (id, invitation_email, created_at, expires_at, ip_address_hash)
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, invitation.Email, now, expiresAt, ipHash)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		return nil, ErrInternalError
	}

	// Return the session
	return &Session{
		ID:              sessionID,
		InvitationEmail: invitation.Email,
		CreatedAt:       now,
		ExpiresAt:       expiresAt,
	}, nil
}

// GetSession retrieves a session by ID
func GetSession(sessionID string) (*Session, error) {
	var session Session
	err := db.DB.QueryRow(`
		SELECT id, invitation_email, created_at, expires_at
		FROM sessions
		WHERE id = ?
	`, sessionID).Scan(
		&session.ID,
		&session.InvitationEmail,
		&session.CreatedAt,
		&session.ExpiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrSessionExpired
		}
		log.Printf("Error retrieving session: %v", err)
		return nil, ErrInternalError
	}

	// Check if the session has expired
	if time.Now().After(session.ExpiresAt) {
		// Delete expired session
		_, _ = db.DB.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
		return nil, ErrSessionExpired
	}

	return &session, nil
}

// SetSessionCookie sets a session cookie in the HTTP response
func SetSessionCookie(w http.ResponseWriter, session *Session) {
	// Create a secure session token
	token := security.CreateSessionToken(session.ID)

	// Check if we're in development mode
	isDev := os.Getenv("ENVIRONMENT") == "development" || os.Getenv("ENVIRONMENT") == "dev" ||
		os.Getenv("ENVIRONMENT") == ""

	// Set the cookie
	var sameSite http.SameSite
	if isDev {
		sameSite = http.SameSiteLaxMode
	} else {
		sameSite = http.SameSiteStrictMode
	}

	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   !isDev, // Only require HTTPS in production
		SameSite: sameSite,
	}

	http.SetCookie(w, cookie)
}

// ClearSessionCookie removes the session cookie
func ClearSessionCookie(w http.ResponseWriter) {
	// Check if we're in development mode
	isDev := os.Getenv("ENVIRONMENT") == "development" || os.Getenv("ENVIRONMENT") == "dev" ||
		os.Getenv("ENVIRONMENT") == ""

	var sameSite http.SameSite
	if isDev {
		sameSite = http.SameSiteLaxMode
	} else {
		sameSite = http.SameSiteStrictMode
	}

	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !isDev, // Only require HTTPS in production
		SameSite: sameSite,
	}

	http.SetCookie(w, cookie)
}

// GetSessionFromRequest extracts and validates the session from a request
func GetSessionFromRequest(r *http.Request) (*Session, error) {
	// Get the cookie
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil, ErrSessionExpired
	}

	// Verify token and extract session ID
	sessionID, valid := security.VerifySessionToken(cookie.Value)
	if !valid {
		return nil, ErrSessionExpired
	}

	// Get the session from the database
	return GetSession(sessionID)
}

// Helper function to get client IP
func getIP(r *http.Request) string {
	// Check for forwarded IP (for proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Get the first IP in the list
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}

	// Direct connection
	return strings.Split(r.RemoteAddr, ":")[0]
}
