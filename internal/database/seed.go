package database

import (
	"database/sql"
	"log"

	"golang.org/x/crypto/bcrypt"
)

// SeedDatabase inserts initial data for testing
func SeedDatabase(db *sql.DB) error {
	// Create a test user
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Insert test user
	_, err = db.Exec(`
		INSERT INTO users (username, email, password_hash, full_name, role) 
		VALUES ($1, $2, $3, $4, $5) 
		ON CONFLICT (username) DO NOTHING
	`, "testuser", "test@example.com", string(passwordHash), "Test User", "admin")

	if err != nil {
		return err
	}

	// Insert test project
	_, err = db.Exec(`
		INSERT INTO projects (name, description, owner_id, status) 
		VALUES ($1, $2, (SELECT id FROM users WHERE username = 'testuser'), $3)
		ON CONFLICT DO NOTHING
	`, "Test Project", "A test project for the API", "active")

	if err != nil {
		return err
	}

	log.Println("Database seeded successfully")
	return nil
}