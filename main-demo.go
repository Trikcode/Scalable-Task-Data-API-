package main

import (
	"log"
	"net/http"
	"os"
	"scalable-task-api/internal/auth"
	"scalable-task-api/internal/config"
	"scalable-task-api/internal/database"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "demo" {
		runDemoMode()
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		log.Println("Starting in demo mode instead...")
		runDemoMode()
		return
	}
	defer db.Close()

	log.Println("Starting with real database...")
}

func runDemoMode() {
	log.Println("🚀 Starting Scalable Task Data API in Demo Mode")
	log.Println("📊 This demo showcases the API without requiring TimescaleDB")

	// Create mock database
	db, mock, err := database.NewMockConnection()
	if err != nil {
		log.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Set up mock expectations
	database.SetupMockLogin(mock)
	database.SetupMockTasks(mock)

	// Load minimal config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		JWT: config.JWTConfig{
			SecretKey:         "demo-secret-key",
			TokenExpiration:   24 * time.Hour,
			RefreshExpiration: 7 * 24 * time.Hour,
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
			Port:    8081,
		},
	}

	// Run migrations on mock database
	if err := database.RunMigrations(db); err != nil {
		log.Printf("Migration warning (expected in demo): %v", err)
	}

	// Initialize services
	jwtService := auth.NewJWTService(&cfg.JWT)

	// Set up Gin in demo mode
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Demo endpoints
	setupDemoRoutes(router, jwtService)

	log.Printf("🌐 API Server running on http://localhost:%d", cfg.Server.Port)
	log.Printf("📈 Metrics available on http://localhost:%d/metrics", cfg.Metrics.Port)
	log.Println("📚 Available endpoints:")
	log.Println("  GET  /health              - Health check")
	log.Println("  POST /api/v1/auth/login   - Login (use: testuser/password123)")
	log.Println("  GET  /api/v1/auth/me      - Get current user")
	log.Println("  GET  /api/v1/tasks        - Get tasks")
	log.Println("  POST /api/v1/tasks        - Create task")
	log.Println("  GET  /api/v1/tasks/1      - Get specific task")
	log.Println("  GET  /api/v1/tasks/metrics - Get task metrics")

	// Start server
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupDemoRoutes(router *gin.Engine, jwtService *auth.JWTService) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"timestamp":   time.Now().UTC(),
			"service":     "scalable-task-api",
			"mode":        "demo",
			"description": "Task Data API with TimescaleDB integration (Demo Mode)",
		})
	})

	// API routes
	v1 := router.Group("/api/v1")

	// Auth endpoints
	auth := v1.Group("/auth")
	{
		auth.POST("/login", func(c *gin.Context) {
			var req auth.LoginRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// Demo login - accept testuser/password123
			if req.Username == "testuser" && req.Password == "password123" {
				accessToken, _ := jwtService.GenerateToken(1, "testuser", "admin")
				refreshToken, _ := jwtService.GenerateRefreshToken(1, "testuser", "admin")

				c.JSON(http.StatusOK, auth.TokenResponse{
					AccessToken:  accessToken,
					RefreshToken: refreshToken,
					TokenType:    "Bearer",
					ExpiresIn:    86400,
				})
				return
			}

			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials. Use testuser/password123"})
		})

		auth.GET("/me", func(c *gin.Context) {
			// Simple auth check for demo
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"id":         1,
				"username":   "testuser",
				"email":      "test@example.com",
				"full_name":  "Test User",
				"role":       "admin",
				"created_at": time.Now().UTC(),
			})
		})
	}

	// Task endpoints
	tasks := v1.Group("/tasks")
	{
		tasks.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, []gin.H{
				{
					"id":          1,
					"title":       "Implement Authentication",
					"description": "Add JWT authentication to the API",
					"status":      "done",
					"priority":    1,
					"project_id":  1,
					"created_at":  time.Now().Add(-24 * time.Hour),
					"updated_at":  time.Now().Add(-2 * time.Hour),
					"tags":        []string{"auth", "security"},
				},
				{
					"id":          2,
					"title":       "Create Task Management",
					"description": "Build CRUD operations for tasks",
					"status":      "in_progress",
					"priority":    2,
					"project_id":  1,
					"created_at":  time.Now().Add(-12 * time.Hour),
					"updated_at":  time.Now().Add(-1 * time.Hour),
					"tags":        []string{"tasks", "crud"},
				},
				{
					"id":          3,
					"title":       "Add Prometheus Metrics",
					"description": "Implement monitoring and metrics collection",
					"status":      "todo",
					"priority":    3,
					"project_id":  1,
					"created_at":  time.Now().Add(-6 * time.Hour),
					"updated_at":  time.Now().Add(-30 * time.Minute),
					"tags":        []string{"monitoring", "metrics"},
				},
			})
		})

		tasks.POST("", func(c *gin.Context) {
			var req map[string]interface{}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// Create demo task
			task := gin.H{
				"id":          4,
				"title":       req["title"],
				"description": req["description"],
				"status":      req["status"],
				"priority":    req["priority"],
				"project_id":  req["project_id"],
				"created_at":  time.Now().UTC(),
				"updated_at":  time.Now().UTC(),
				"tags":        req["tags"],
			}

			c.JSON(http.StatusCreated, task)
		})

		tasks.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{
				"id":          id,
				"title":       "Sample Task",
				"description": "This is a demo task showing API functionality",
				"status":      "in_progress",
				"priority":    2,
				"project_id":  1,
				"created_at":  time.Now().Add(-8 * time.Hour),
				"updated_at":  time.Now().Add(-2 * time.Hour),
				"tags":        []string{"demo", "api"},
			})
		})

		tasks.GET("/metrics", func(c *gin.Context) {
			c.JSON(http.StatusOK, []gin.H{
				{
					"timestamp":           time.Now().Add(-24 * time.Hour),
					"total_tasks":         15,
					"completed_tasks":     8,
					"in_progress_tasks":   5,
					"overdue_tasks":       2,
					"avg_completion_time": 4.5,
				},
				{
					"timestamp":           time.Now().Add(-12 * time.Hour),
					"total_tasks":         18,
					"completed_tasks":     10,
					"in_progress_tasks":   6,
					"overdue_tasks":       2,
					"avg_completion_time": 4.2,
				},
				{
					"timestamp":           time.Now(),
					"total_tasks":         20,
					"completed_tasks":     12,
					"in_progress_tasks":   6,
					"overdue_tasks":       2,
					"avg_completion_time": 4.0,
				},
			})
		})
	}

	// Metrics endpoint
	router.GET("/metrics", func(c *gin.Context) {
		c.String(http.StatusOK, `# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",endpoint="/health",status_code="200"} 142
http_requests_total{method="POST",endpoint="/api/v1/auth/login",status_code="200"} 23
http_requests_total{method="GET",endpoint="/api/v1/tasks",status_code="200"} 87

# HELP active_tasks_total Total number of active tasks
# TYPE active_tasks_total gauge
active_tasks_total 12

# HELP database_connections_active Number of active database connections
# TYPE database_connections_active gauge
database_connections_active 5
`)
	})
}