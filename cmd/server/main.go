package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"snapdeploy-core/internal/clerk"
	"snapdeploy-core/internal/config"
	"snapdeploy-core/internal/database"
	"snapdeploy-core/internal/handlers"
	"snapdeploy-core/internal/middleware"
	"snapdeploy-core/internal/repositories"
	"snapdeploy-core/internal/services"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title SnapDeploy Core API
// @version 1.0
// @description A modern user management system with Clerk authentication
// @termsOfService http://swagger.io/terms/

// @contact.name SnapDeploy Team
// @contact.email support@snapdeploy.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey ClerkAuth
// @in header
// @name Authorization
// @description Clerk JWT token

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.NewConnection(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)

	// Initialize Clerk client
	clerkClient := clerk.NewClient(&cfg.Clerk)

	// Initialize services
	userService := services.NewUserService(userRepo, clerkClient)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler()
	userHandler := handlers.NewUserHandler(userService)

	// Initialize auth middleware
	authMiddleware, err := middleware.NewAuthMiddleware(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize auth middleware: %v", err)
	}

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := gin.New()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	// allow from all origins
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		// Handle preflight OPTIONS requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Health check endpoint (no auth required)
		v1.GET("/health", healthHandler.Health)

		// Auth routes
		auth := v1.Group("/auth")
		auth.Use(authMiddleware.RequireAuth())
		{
			auth.GET("/me", userHandler.GetCurrentUser)
		}

		// User management routes
		users := v1.Group("/users")
		users.Use(authMiddleware.RequireAuth())
		{
			users.GET("", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUserByID)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}
	}

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.GetServerAddress(),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", cfg.GetServerAddress())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
