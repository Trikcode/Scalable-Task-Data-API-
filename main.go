package main

import (
	"log"
	"scalable-task-api/internal/api"
	"scalable-task-api/internal/config"
	"scalable-task-api/internal/database"
)



func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run database migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize and start API server
	server := api.NewServer(cfg, db)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}