package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	
	"github.com/joho/godotenv"
	
	"wedding-invite/pkg/db"
	"wedding-invite/pkg/security"
)

func init() {
	// Load environment variables from .env file if it exists
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
	// Command line arguments
	familyName := flag.String("name", "", "Family name for the invitation")
	maxGuests := flag.Int("guests", 2, "Maximum number of guests allowed")
	email := flag.String("email", "", "Contact email (optional)")
	phone := flag.String("phone", "", "Contact phone (optional)")
	customCode := flag.String("code", "", "Custom invitation code (optional, will be generated if not provided)")
	
	flag.Parse()
	
	// Validate input
	if *familyName == "" {
		fmt.Println("Error: Family name is required")
		flag.Usage()
		os.Exit(1)
	}
	
	// Initialize security package
	if err := security.Initialize(); err != nil {
		log.Fatalf("Failed to initialize security: %v", err)
	}
	
	// Initialize database
	if err := db.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	
	// Generate or use custom invitation code
	var invitationCode string
	var err error
	
	if *customCode != "" {
		invitationCode = *customCode
	} else {
		invitationCode, err = security.GenerateInvitationCode()
		if err != nil {
			log.Fatalf("Failed to generate invitation code: %v", err)
		}
	}
	
	// Create invitation in database
	_, err = db.DB.Exec(`
		INSERT INTO invitations (id, family_name, max_guests, email, phone)
		VALUES (?, ?, ?, ?, ?)
	`, invitationCode, *familyName, *maxGuests, 
	   sqlNullString(*email), sqlNullString(*phone))
	
	if err != nil {
		log.Fatalf("Failed to create invitation: %v", err)
	}
	
	// Create the invitation URL
	invitationURL := fmt.Sprintf("https://wedding.bogdanfloris.com/%s", invitationCode)
	
	// Output the result
	fmt.Println("✅ Invitation created successfully!")
	fmt.Println("-----------------------------------")
	fmt.Printf("Family: %s\n", *familyName)
	fmt.Printf("Max Guests: %d\n", *maxGuests)
	if *email != "" {
		fmt.Printf("Email: %s\n", *email)
	}
	if *phone != "" {
		fmt.Printf("Phone: %s\n", *phone)
	}
	fmt.Println("-----------------------------------")
	fmt.Printf("Invitation Code: %s\n", invitationCode)
	fmt.Printf("Invitation URL: %s\n", invitationURL)
	fmt.Println("-----------------------------------")
	fmt.Println("Share this URL with your guests!")
}

// Helper function to convert empty strings to SQL NULL
func sqlNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}