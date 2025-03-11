package db

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Initialize() error {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "wedding.db"
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
		CREATE TABLE IF NOT EXISTS guests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT,
			phone TEXT,
			attending BOOLEAN DEFAULT false,
			plus_ones INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	
	return err
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}