package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Initialize() error {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "wedding.db"
	}

	log.Printf("Using database at: %s", dbPath)

	// Ensure the directory exists (for SQLite)
	dbDir := filepath.Dir(dbPath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		log.Printf("Creating database directory: %s", dbDir)
		if err := os.MkdirAll(dbDir, 0o755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	if err = DB.Ping(); err != nil {
		return err
	}

	log.Println("Database connected successfully")
	return setupSchema()
}

func setupSchema() error {
	// Create tables if they don't exist
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS invitations (
			email TEXT PRIMARY KEY,
			max_guests INTEGER NOT NULL DEFAULT 6,
			phone TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_access TIMESTAMP,
			approved BOOLEAN DEFAULT FALSE,
			registration_ip TEXT
		);
		
		CREATE TABLE IF NOT EXISTS guests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			invitation_email TEXT REFERENCES invitations(email),
			name TEXT NOT NULL,
			attending BOOLEAN DEFAULT NULL,
			meal_preference TEXT,
			dietary_restrictions TEXT,
			last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			invitation_email TEXT REFERENCES invitations(email),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			ip_address_hash TEXT
		);
	`)

	return err
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
