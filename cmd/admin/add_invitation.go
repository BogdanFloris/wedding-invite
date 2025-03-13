package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"wedding-invite/pkg/db"
	"wedding-invite/pkg/security"

	"github.com/joho/godotenv"
	"github.com/skip2/go-qrcode"
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
	customCode := flag.String(
		"code",
		"",
		"Custom invitation code (optional, will be generated if not provided)",
	)

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

	// Flags for output options
	generateQR := flag.Bool("qr", true, "Generate QR code for invitation")
	outputDir := flag.String("output", "qrcodes", "Directory to save QR codes and output files")
	baseURL := flag.String("url", "https://wedding.bogdanfloris.com", "Base URL for invitations")

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
	invitationURL := fmt.Sprintf("%s/%s", *baseURL, invitationCode)

	// Generate QR code if requested
	qrFilePath := ""
	if *generateQR {
		// Create output directory if it doesn't exist
		if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
			if err := os.MkdirAll(*outputDir, 0755); err != nil {
				log.Printf("Warning: Failed to create output directory: %v", err)
			}
		}

		// Generate file name based on family name
		safeFamilyName := strings.ReplaceAll(*familyName, " ", "_")
		qrFilePath = filepath.Join(
			*outputDir,
			fmt.Sprintf("%s_%s.png", safeFamilyName, invitationCode),
		)

		// Generate and save QR code
		if err := generateQRCode(invitationURL, qrFilePath); err != nil {
			log.Printf("Warning: Failed to generate QR code: %v", err)
		} else {
			log.Printf("QR code saved to: %s", qrFilePath)
		}
	}

	// Generate WhatsApp message template
	whatsappMsg := generateWhatsAppMessage(*familyName, invitationCode, invitationURL)

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
	if qrFilePath != "" {
		fmt.Printf("QR Code: %s\n", qrFilePath)
	}
	fmt.Println("-----------------------------------")
	fmt.Println("WhatsApp Message Template:")
	fmt.Println(whatsappMsg)
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

// Generate a QR code for the given invitation URL
func generateQRCode(url, filePath string) error {
	// Generate QR code with medium-high quality
	err := qrcode.WriteFile(url, qrcode.Medium, 256, filePath)
	return err
}

// Generate a WhatsApp message template
func generateWhatsAppMessage(familyName, code, url string) string {
	return fmt.Sprintf(`Hello %s!

You're invited to *Ramona & Bogdan's Wedding* 💍

Please access your personalized invitation by either:
📱 Scanning the QR code (attached image)
OR
🔗 Tapping this link: %s

We can't wait to celebrate with you!

RSVP by September 1, 2025.

---
*Attending with others?* You can include up to 4 guests total when you RSVP.`,
		familyName, url)
}
