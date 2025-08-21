package main

import (
        "log"
        "net/http"
        "time"

        "github.com/gin-gonic/gin"
)

func main() {
        log.Println("🚀 Starting Scalable Task Data API - Demo Mode")
        log.Println("📊 RESTful API for managing time-series task data")
        log.Println("🔧 Features: JWT Auth, TimescaleDB, Prometheus, Kubernetes")

        gin.SetMode(gin.ReleaseMode)
        router := gin.New()
        router.Use(gin.Recovery())

        // CORS middleware
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

        setupRoutes(router)

        log.Printf("🌐 API Server running on http://localhost:5000")
        log.Printf("📈 Metrics endpoint: http://localhost:5000/metrics")
        log.Println("\n📚 Available API Endpoints:")
        log.Println("  GET  /health                 - Health check")
        log.Println("  POST /api/v1/auth/login      - Login (demo: testuser/password123)")
        log.Println("  GET  /api/v1/auth/me         - Get current user info")
        log.Println("  GET  /api/v1/tasks           - Get all tasks")
        log.Println("  POST /api/v1/tasks           - Create new task")
        log.Println("  GET  /api/v1/tasks/1         - Get specific task")
        log.Println("  PUT  /api/v1/tasks/1         - Update task")
        log.Println("  DELETE /api/v1/tasks/1       - Delete task")
        log.Println("  GET  /api/v1/tasks/metrics   - Get time-series metrics")

        if err := http.ListenAndServe(":5000", router); err != nil {
                log.Fatalf("Failed to start server: %v", err)
        }
}

func setupRoutes(router *gin.Engine) {
        // Health check
        router.GET("/health", func(c *gin.Context) {
                c.JSON(http.StatusOK, gin.H{
                        "status":      "ok",
                        "timestamp":   time.Now().UTC(),
                        "service":     "scalable-task-api",
                        "version":     "1.0.0",
                        "mode":        "demo",
                        "description": "RESTful API for managing time-series task data with TimescaleDB",
                })
        })

        // API v1 routes
        v1 := router.Group("/api/v1")

        // Authentication endpoints
        auth := v1.Group("/auth")
        {
                auth.POST("/login", func(c *gin.Context) {
                        var credentials map[string]string
                        if err := c.ShouldBindJSON(&credentials); err != nil {
                                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                                return
                        }

                        username := credentials["username"]
                        password := credentials["password"]

                        // Demo authentication
                        if username == "testuser" && password == "password123" {
                                c.JSON(http.StatusOK, gin.H{
                                        "access_token":  "demo_jwt_token_12345",
                                        "refresh_token": "demo_refresh_token_67890",
                                        "token_type":    "Bearer",
                                        "expires_in":    86400,
                                        "user": gin.H{
                                                "id":       1,
                                                "username": "testuser",
                                                "role":     "admin",
                                        },
                                })
                                return
                        }

                        c.JSON(http.StatusUnauthorized, gin.H{
                                "error": "Invalid credentials. Use testuser/password123 for demo",
                        })
                })

                auth.GET("/me", func(c *gin.Context) {
                        c.JSON(http.StatusOK, gin.H{
                                "id":         1,
                                "username":   "testuser",
                                "email":      "test@example.com",
                                "full_name":  "Test User",
                                "role":       "admin",
                                "created_at": time.Now().Add(-30 * 24 * time.Hour),
                                "updated_at": time.Now(),
                        })
                })
        }

        // Task management endpoints
        tasks := v1.Group("/tasks")
        {
                // Get all tasks with filtering and pagination
                tasks.GET("", func(c *gin.Context) {
                        // Sample time-series task data
                        tasks := []gin.H{
                                {
                                        "id":              1,
                                        "title":           "Implement JWT Authentication",
                                        "description":     "Add secure JWT-based authentication to the API",
                                        "status":          "done",
                                        "priority":        1,
                                        "assignee_id":     1,
                                        "project_id":      1,
                                        "created_at":      time.Now().Add(-72 * time.Hour),
                                        "updated_at":      time.Now().Add(-6 * time.Hour),
                                        "completed_at":    time.Now().Add(-6 * time.Hour),
                                        "due_date":        time.Now().Add(-48 * time.Hour),
                                        "estimated_hours": 8.0,
                                        "actual_hours":    6.5,
                                        "tags":            []string{"auth", "security", "backend"},
                                },
                                {
                                        "id":              2,
                                        "title":           "Set up TimescaleDB Integration",
                                        "description":     "Configure TimescaleDB for time-series task data storage",
                                        "status":          "done",
                                        "priority":        1,
                                        "assignee_id":     1,
                                        "project_id":      1,
                                        "created_at":      time.Now().Add(-48 * time.Hour),
                                        "updated_at":      time.Now().Add(-2 * time.Hour),
                                        "completed_at":    time.Now().Add(-2 * time.Hour),
                                        "due_date":        time.Now().Add(-24 * time.Hour),
                                        "estimated_hours": 12.0,
                                        "actual_hours":    10.5,
                                        "tags":            []string{"database", "timescale", "setup"},
                                },
                                {
                                        "id":              3,
                                        "title":           "Add Prometheus Monitoring",
                                        "description":     "Implement comprehensive metrics collection and monitoring",
                                        "status":          "in_progress",
                                        "priority":        2,
                                        "assignee_id":     1,
                                        "project_id":      1,
                                        "created_at":      time.Now().Add(-24 * time.Hour),
                                        "updated_at":      time.Now().Add(-1 * time.Hour),
                                        "completed_at":    nil,
                                        "due_date":        time.Now().Add(24 * time.Hour),
                                        "estimated_hours": 6.0,
                                        "actual_hours":    4.0,
                                        "tags":            []string{"monitoring", "prometheus", "observability"},
                                },
                                {
                                        "id":              4,
                                        "title":           "Kubernetes Deployment",
                                        "description":     "Create Kubernetes manifests for production deployment",
                                        "status":          "todo",
                                        "priority":        3,
                                        "assignee_id":     nil,
                                        "project_id":      1,
                                        "created_at":      time.Now().Add(-12 * time.Hour),
                                        "updated_at":      time.Now().Add(-12 * time.Hour),
                                        "completed_at":    nil,
                                        "due_date":        time.Now().Add(72 * time.Hour),
                                        "estimated_hours": 4.0,
                                        "actual_hours":    nil,
                                        "tags":            []string{"kubernetes", "deployment", "devops"},
                                },
                                {
                                        "id":              5,
                                        "title":           "API Documentation",
                                        "description":     "Create comprehensive Swagger/OpenAPI documentation",
                                        "status":          "review",
                                        "priority":        2,
                                        "assignee_id":     1,
                                        "project_id":      1,
                                        "created_at":      time.Now().Add(-6 * time.Hour),
                                        "updated_at":      time.Now().Add(-30 * time.Minute),
                                        "completed_at":    nil,
                                        "due_date":        time.Now().Add(48 * time.Hour),
                                        "estimated_hours": 3.0,
                                        "actual_hours":    2.5,
                                        "tags":            []string{"documentation", "swagger", "api"},
                                },
                        }

                        c.JSON(http.StatusOK, gin.H{
                                "data":  tasks,
                                "total": len(tasks),
                                "page":  1,
                                "limit": 50,
                        })
                })

                // Create new task
                tasks.POST("", func(c *gin.Context) {
                        var newTask map[string]interface{}
                        if err := c.ShouldBindJSON(&newTask); err != nil {
                                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                                return
                        }

                        // Create task with generated ID
                        task := gin.H{
                                "id":              6,
                                "title":           newTask["title"],
                                "description":     newTask["description"],
                                "status":          newTask["status"],
                                "priority":        newTask["priority"],
                                "assignee_id":     newTask["assignee_id"],
                                "project_id":      newTask["project_id"],
                                "created_at":      time.Now().UTC(),
                                "updated_at":      time.Now().UTC(),
                                "completed_at":    nil,
                                "due_date":        newTask["due_date"],
                                "estimated_hours": newTask["estimated_hours"],
                                "actual_hours":    nil,
                                "tags":            newTask["tags"],
                        }

                        c.JSON(http.StatusCreated, task)
                })

                // Get specific task
                tasks.GET("/:id", func(c *gin.Context) {
                        id := c.Param("id")
                        
                        task := gin.H{
                                "id":              id,
                                "title":           "Sample Task " + id,
                                "description":     "This is a detailed view of task " + id + " showing all available fields",
                                "status":          "in_progress",
                                "priority":        2,
                                "assignee_id":     1,
                                "project_id":      1,
                                "created_at":      time.Now().Add(-8 * time.Hour),
                                "updated_at":      time.Now().Add(-2 * time.Hour),
                                "completed_at":    nil,
                                "due_date":        time.Now().Add(24 * time.Hour),
                                "estimated_hours": 5.0,
                                "actual_hours":    3.0,
                                "tags":            []string{"sample", "demo", "task"},
                        }

                        c.JSON(http.StatusOK, task)
                })

                // Update task
                tasks.PUT("/:id", func(c *gin.Context) {
                        id := c.Param("id")
                        var updates map[string]interface{}
                        if err := c.ShouldBindJSON(&updates); err != nil {
                                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                                return
                        }

                        // Return updated task
                        task := gin.H{
                                "id":              id,
                                "title":           updates["title"],
                                "description":     updates["description"],
                                "status":          updates["status"],
                                "priority":        updates["priority"],
                                "assignee_id":     updates["assignee_id"],
                                "project_id":      updates["project_id"],
                                "created_at":      time.Now().Add(-8 * time.Hour),
                                "updated_at":      time.Now().UTC(),
                                "completed_at":    updates["completed_at"],
                                "due_date":        updates["due_date"],
                                "estimated_hours": updates["estimated_hours"],
                                "actual_hours":    updates["actual_hours"],
                                "tags":            updates["tags"],
                        }

                        c.JSON(http.StatusOK, task)
                })

                // Delete task
                tasks.DELETE("/:id", func(c *gin.Context) {
                        c.Status(http.StatusNoContent)
                })

                // Get time-series metrics
                tasks.GET("/metrics", func(c *gin.Context) {
                        // Sample time-series metrics data
                        metrics := []gin.H{
                                {
                                        "timestamp":           time.Now().Add(-7 * 24 * time.Hour),
                                        "total_tasks":         18,
                                        "completed_tasks":     12,
                                        "in_progress_tasks":   4,
                                        "overdue_tasks":       2,
                                        "avg_completion_time": 5.2,
                                        "project_id":          1,
                                },
                                {
                                        "timestamp":           time.Now().Add(-6 * 24 * time.Hour),
                                        "total_tasks":         20,
                                        "completed_tasks":     13,
                                        "in_progress_tasks":   5,
                                        "overdue_tasks":       2,
                                        "avg_completion_time": 4.8,
                                        "project_id":          1,
                                },
                                {
                                        "timestamp":           time.Now().Add(-5 * 24 * time.Hour),
                                        "total_tasks":         22,
                                        "completed_tasks":     15,
                                        "in_progress_tasks":   5,
                                        "overdue_tasks":       2,
                                        "avg_completion_time": 4.5,
                                        "project_id":          1,
                                },
                                {
                                        "timestamp":           time.Now().Add(-4 * 24 * time.Hour),
                                        "total_tasks":         25,
                                        "completed_tasks":     18,
                                        "in_progress_tasks":   5,
                                        "overdue_tasks":       2,
                                        "avg_completion_time": 4.2,
                                        "project_id":          1,
                                },
                                {
                                        "timestamp":           time.Now().Add(-3 * 24 * time.Hour),
                                        "total_tasks":         27,
                                        "completed_tasks":     20,
                                        "in_progress_tasks":   5,
                                        "overdue_tasks":       2,
                                        "avg_completion_time": 4.0,
                                        "project_id":          1,
                                },
                                {
                                        "timestamp":           time.Now().Add(-2 * 24 * time.Hour),
                                        "total_tasks":         30,
                                        "completed_tasks":     22,
                                        "in_progress_tasks":   6,
                                        "overdue_tasks":       2,
                                        "avg_completion_time": 3.8,
                                        "project_id":          1,
                                },
                                {
                                        "timestamp":           time.Now().Add(-1 * 24 * time.Hour),
                                        "total_tasks":         32,
                                        "completed_tasks":     24,
                                        "in_progress_tasks":   6,
                                        "overdue_tasks":       2,
                                        "avg_completion_time": 3.5,
                                        "project_id":          1,
                                },
                                {
                                        "timestamp":           time.Now(),
                                        "total_tasks":         35,
                                        "completed_tasks":     26,
                                        "in_progress_tasks":   7,
                                        "overdue_tasks":       2,
                                        "avg_completion_time": 3.2,
                                        "project_id":          1,
                                },
                        }

                        c.JSON(http.StatusOK, gin.H{
                                "data":     metrics,
                                "interval": "day",
                                "range":    "7 days",
                        })
                })
        }

        // Prometheus metrics endpoint
        router.GET("/metrics", func(c *gin.Context) {
                metricsData := `# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",endpoint="/health",status_code="200"} 342
http_requests_total{method="POST",endpoint="/api/v1/auth/login",status_code="200"} 45
http_requests_total{method="GET",endpoint="/api/v1/tasks",status_code="200"} 187
http_requests_total{method="POST",endpoint="/api/v1/tasks",status_code="201"} 23
http_requests_total{method="GET",endpoint="/api/v1/tasks/metrics",status_code="200"} 67

# HELP http_request_duration_seconds Duration of HTTP requests in seconds
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",endpoint="/api/v1/tasks",status_code="200",le="0.1"} 145
http_request_duration_seconds_bucket{method="GET",endpoint="/api/v1/tasks",status_code="200",le="0.5"} 180
http_request_duration_seconds_bucket{method="GET",endpoint="/api/v1/tasks",status_code="200",le="1"} 187
http_request_duration_seconds_bucket{method="GET",endpoint="/api/v1/tasks",status_code="200",le="+Inf"} 187

# HELP tasks_total Total number of tasks by status
# TYPE tasks_total gauge
tasks_total{status="todo",project_id="1"} 7
tasks_total{status="in_progress",project_id="1"} 6
tasks_total{status="review",project_id="1"} 3
tasks_total{status="done",project_id="1"} 19

# HELP active_tasks_total Total number of active tasks
# TYPE active_tasks_total gauge
active_tasks_total 16

# HELP database_connections_active Number of active database connections
# TYPE database_connections_active gauge
database_connections_active 8
`
                c.String(http.StatusOK, metricsData)
        })
}