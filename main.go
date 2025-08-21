package main

import (
	"log"
	"scalable-task-api/internal/api"
	"scalable-task-api/internal/config"
	"scalable-task-api/internal/database"
)

// @title Scalable Task Data API
// @version 1.0
// @description A RESTful API for managing time-series task data with TimescaleDB
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

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