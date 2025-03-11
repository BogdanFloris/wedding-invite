package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"
)

var secretKey []byte

// Initialize sets up the security package
func Initialize() error {
	// Try to get secret key from environment variable
	existingKey := os.Getenv("SECRET_KEY")
	if existingKey != "" {
		// Decode base64 encoded key
		decoded, err := base64.StdEncoding.DecodeString(existingKey)
		if err != nil {
			// If it's not base64 encoded, use as is
			secretKey = []byte(existingKey)
		} else {
			secretKey = decoded
		}
		log.Println("Using SECRET_KEY from environment")
		return nil
	}

	// Generate a new secret key if one doesn't exist
	newKey := make([]byte, 32)
	if _, err := rand.Read(newKey); err != nil {
		return fmt.Errorf("failed to generate secret key: %w", err)
	}
	
	secretKey = newKey
	
	// For development/first-time setup, provide the key for adding to environment
	encodedKey := base64.StdEncoding.EncodeToString(secretKey)
	log.Printf("⚠️ WARNING: Using temporary SECRET_KEY. For production, set this in environment:")
	log.Printf("SECRET_KEY=\"%s\"", encodedKey)
	log.Printf("Add this to your .env file or fly.io secrets")
	
	return nil
}

// GenerateInvitationCode creates a new random invitation code
func GenerateInvitationCode() (string, error) {
	// Define the characters to use (alphanumeric only for readability)
	const charset = "abcdefghijkmnpqrstuvwxyz23456789" // removed confusing chars like 0/O, 1/l
	const codeLength = 8
	
	// Create a slice to store the code
	code := make([]byte, codeLength)
	
	// Generate random bytes
	randomBytes := make([]byte, codeLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	
	// Map random bytes to characters in charset
	for i, b := range randomBytes {
		code[i] = charset[b%byte(len(charset))]
	}
	
	return string(code), nil
}

// GenerateSessionID creates a new random session ID
func GenerateSessionID() (string, error) {
	// Generate 24 random bytes for session ID
	idBytes := make([]byte, 24)
	if _, err := rand.Read(idBytes); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	
	// Encode as hex
	id := hex.EncodeToString(idBytes)
	return id, nil
}

// HashIPAddress creates a secure hash of an IP address
func HashIPAddress(ip string) string {
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(ip))
	return hex.EncodeToString(h.Sum(nil))
}

// CreateSessionToken generates a signed session token
func CreateSessionToken(sessionID string) string {
	// Combine session ID and timestamp for added security
	timestamp := time.Now().Unix()
	payload := fmt.Sprintf("%s|%d", sessionID, timestamp)
	
	// Sign with HMAC
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(payload))
	signature := hex.EncodeToString(h.Sum(nil))
	
	// Combine payload and signature
	token := fmt.Sprintf("%s.%s", payload, signature)
	return base64.URLEncoding.EncodeToString([]byte(token))
}

// VerifySessionToken validates a session token
func VerifySessionToken(tokenString string) (string, bool) {
	// Decode token
	tokenBytes, err := base64.URLEncoding.DecodeString(tokenString)
	if err != nil {
		return "", false
	}
	
	token := string(tokenBytes)
	
	// Split token into payload and signature
	parts := split(token, ".")
	if len(parts) != 2 {
		return "", false
	}
	
	payload := parts[0]
	providedSignature := parts[1]
	
	// Compute expected signature
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(h.Sum(nil))
	
	// Verify signature
	if !hmac.Equal([]byte(providedSignature), []byte(expectedSignature)) {
		return "", false
	}
	
	// Extract session ID from payload
	payloadParts := split(payload, "|")
	if len(payloadParts) != 2 {
		return "", false
	}
	
	return payloadParts[0], true
}

// Helper function to split a string
func split(s, sep string) []string {
	result := make([]string, 0)
	if s == "" {
		return result
	}
	
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	
	result = append(result, s[start:])
	return result
}