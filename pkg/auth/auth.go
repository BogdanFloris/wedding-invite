package auth

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
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
	
	// RateLimitCount is the maximum number of code attempts per IP per minute
	RateLimitCount = 5
	
	// RateLimitWindow is the time window for rate limiting (1 minute)
	RateLimitWindow = time.Minute
)

// Errors
var (
	ErrInvalidCode        = errors.New("invalid invitation code")
	ErrSessionExpired     = errors.New("session expired")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded, please try again later")
	ErrInternalError      = errors.New("an internal error occurred")
)

// Session represents an authenticated session
type Session struct {
	ID           string
	InvitationID string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// Invitation represents invitation details
type Invitation struct {
	ID         string
	FamilyName string
	MaxGuests  int
	Email      sql.NullString
	Phone      sql.NullString
	CreatedAt  time.Time
	LastAccess sql.NullTime
}

// ValidateInvitationCode checks if an invitation code is valid
func ValidateInvitationCode(code string, r *http.Request) (*Invitation, error) {
	// Clean the code (remove spaces, case insensitive)
	code = strings.TrimSpace(strings.ToLower(code))
	
	// Check if code meets minimum requirements
	if len(code) < 8 {
		return nil, ErrInvalidCode
	}
	
	// Check rate limit
	ipHash := security.HashIPAddress(getIP(r))
	if isRateLimited(ipHash) {
		return nil, ErrRateLimitExceeded
	}
	
	// Query the database for the invitation
	var invitation Invitation
	err := db.DB.QueryRow(`
		SELECT id, family_name, max_guests, email, phone, created_at, last_access
		FROM invitations
		WHERE id = ?
	`, code).Scan(
		&invitation.ID,
		&invitation.FamilyName,
		&invitation.MaxGuests,
		&invitation.Email,
		&invitation.Phone,
		&invitation.CreatedAt,
		&invitation.LastAccess,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			// Record failed attempt for rate limiting
			recordFailedAttempt(ipHash)
			return nil, ErrInvalidCode
		}
		log.Printf("Database error validating invitation code: %v", err)
		return nil, ErrInternalError
	}
	
	// Update last access time
	_, err = db.DB.Exec(`
		UPDATE invitations
		SET last_access = CURRENT_TIMESTAMP
		WHERE id = ?
	`, code)
	
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
		INSERT INTO sessions (id, invitation_id, created_at, expires_at, ip_address_hash)
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, invitation.ID, now, expiresAt, ipHash)
	
	if err != nil {
		log.Printf("Error creating session: %v", err)
		return nil, ErrInternalError
	}
	
	// Return the session
	return &Session{
		ID:           sessionID,
		InvitationID: invitation.ID,
		CreatedAt:    now,
		ExpiresAt:    expiresAt,
	}, nil
}

// GetSession retrieves a session by ID
func GetSession(sessionID string) (*Session, error) {
	var session Session
	err := db.DB.QueryRow(`
		SELECT id, invitation_id, created_at, expires_at
		FROM sessions
		WHERE id = ?
	`, sessionID).Scan(
		&session.ID,
		&session.InvitationID,
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
	
	// Set the cookie
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   true, // Requires HTTPS
		SameSite: http.SameSiteStrictMode,
	}
	
	http.SetCookie(w, cookie)
}

// ClearSessionCookie removes the session cookie
func ClearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
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

// Rate limiting helpers
func isRateLimited(ipHash string) bool {
	var count int
	err := db.DB.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT 1 FROM sessions 
			WHERE ip_address_hash = ? 
			AND created_at > ?
			LIMIT ?
		)
	`, ipHash, time.Now().Add(-RateLimitWindow), RateLimitCount+1).Scan(&count)
	
	if err != nil {
		log.Printf("Error checking rate limit: %v", err)
		return false // Don't block on errors
	}
	
	return count > RateLimitCount
}

func recordFailedAttempt(ipHash string) {
	// Record a failed attempt as a temporary session
	sessionID, err := security.GenerateSessionID()
	if err != nil {
		log.Printf("Error generating session ID for rate limiting: %v", err)
		return
	}
	
	now := time.Now()
	expiresAt := now.Add(RateLimitWindow)
	
	_, err = db.DB.Exec(`
		INSERT INTO sessions (id, invitation_id, created_at, expires_at, ip_address_hash)
		VALUES (?, 'failed_attempt', ?, ?, ?)
	`, sessionID, now, expiresAt, ipHash)
	
	if err != nil {
		log.Printf("Error recording failed attempt: %v", err)
	}
}