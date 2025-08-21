package api

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"scalable-task-api/internal/auth"
	"scalable-task-api/internal/config"
	"scalable-task-api/internal/handlers"
	"scalable-task-api/internal/middleware"
	"scalable-task-api/internal/monitoring"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server represents the HTTP server
type Server struct {
	config     *config.Config
	db         *sql.DB
	router     *gin.Engine
	jwtService *auth.JWTService
	metrics    *monitoring.Metrics
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, db *sql.DB) *Server {
	// Initialize services
	jwtService := auth.NewJWTService(&cfg.JWT)
	metrics := monitoring.NewMetrics()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, jwtService)
	taskHandler := handlers.NewTaskHandler(db, metrics)

	// Set up Gin
	if cfg.Server.Host == "0.0.0.0" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware())
	router.Use(monitoring.PrometheusMiddleware(metrics))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC(),
			"service":   "scalable-task-api",
		})
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.GET("/me", middleware.AuthMiddleware(jwtService), authHandler.Me)
		}

		// Protected routes
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware(jwtService))
		{
			// Task routes
			tasks := protected.Group("/tasks")
			{
				tasks.POST("", taskHandler.CreateTask)
				tasks.GET("", taskHandler.GetTasks)
				tasks.GET("/:id", taskHandler.GetTask)
				tasks.PUT("/:id", taskHandler.UpdateTask)
				tasks.DELETE("/:id", taskHandler.DeleteTask)
				tasks.GET("/metrics", taskHandler.GetTaskMetrics)
			}
		}
	}

	return &Server{
		config:     cfg,
		db:         db,
		router:     router,
		jwtService: jwtService,
		metrics:    metrics,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Start metrics server if enabled
	if s.config.Metrics.Enabled {
		go s.startMetricsServer()
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  s.config.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s:%d", s.config.Server.Host, s.config.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give outstanding requests 5 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		return err
	}

	log.Println("Server exited")
	return nil
}

// startMetricsServer starts the Prometheus metrics server
func (s *Server) startMetricsServer() {
	metricsRouter := gin.New()
	metricsRouter.Use(gin.Recovery())
	metricsRouter.GET(s.config.Metrics.Path, gin.WrapH(promhttp.Handler()))

	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Metrics.Port),
		Handler: metricsRouter,
	}

	log.Printf("Starting metrics server on port %d", s.config.Metrics.Port)
	if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("Metrics server error: %v", err)
	}
}