package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"snapdeploy-core/internal/application/service"
	"snapdeploy-core/internal/clerk"
	"snapdeploy-core/internal/config"
	"snapdeploy-core/internal/database"
	"snapdeploy-core/internal/github"
	"snapdeploy-core/internal/infrastructure/builder"
	"snapdeploy-core/internal/infrastructure/codebuild"
	"snapdeploy-core/internal/infrastructure/ecs"
	"snapdeploy-core/internal/infrastructure/encryption"
	infraClerk "snapdeploy-core/internal/infrastructure/clerk"
	infraGitHub "snapdeploy-core/internal/infrastructure/github"
	"snapdeploy-core/internal/infrastructure/persistence"
	"snapdeploy-core/internal/middleware"
	"snapdeploy-core/internal/presentation/handlers"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title SnapDeploy Core API
// @version 1.0
// @description A modern deployment platform with DDD architecture
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

	// Initialize infrastructure layer
	// External service clients
	clerkClient := clerk.NewClient(&cfg.Clerk)
	githubClient := github.NewClient()

	// Infrastructure implementations of domain services
	clerkService := infraClerk.NewClerkService(clerkClient)
	githubService := infraGitHub.NewGitHubService(githubClient)

	// Initialize encryption service
	encryptionService, err := encryption.NewEncryptionService()
	if err != nil {
		log.Fatalf("Failed to initialize encryption service: %v", err)
	}
	log.Printf("Encryption service initialized")

	// Repository implementations
	userRepository := persistence.NewUserRepository(db)
	repositoryRepository := persistence.NewRepositoryRepository(db)
	projectRepository := persistence.NewProjectRepository(db)
	deploymentRepository := persistence.NewDeploymentRepository(db)
	envVarRepository := persistence.NewEnvVarRepository(db, encryptionService)

	// Initialize application layer
	// Application services (use cases)
	userService := service.NewUserService(userRepository, repositoryRepository, clerkService)
	repositoryService := service.NewRepositoryService(repositoryRepository, githubService)
	projectService := service.NewProjectService(projectRepository)
	deploymentService := service.NewDeploymentService(deploymentRepository, projectRepository)
	envVarService := service.NewEnvVarService(envVarRepository, projectRepository, encryptionService)

	// Initialize presentation layer
	// HTTP handlers
	healthHandler := handlers.NewHealthHandler()
	
	// Initialize template generator for Dockerfile generation
	templateGenerator, err := builder.NewTemplateGenerator()
	if err != nil {
		log.Fatalf("Failed to initialize template generator: %v", err)
	}

	// Initialize CodeBuild service (required)
	codebuildProjectName := os.Getenv("CODEBUILD_PROJECT_NAME")
	if codebuildProjectName == "" {
		log.Fatalf("CODEBUILD_PROJECT_NAME environment variable is required")
	}

	codebuildService, err := codebuild.NewCodeBuildService(
		codebuildProjectName,
		deploymentRepository,
		projectRepository,
	)
	if err != nil {
		log.Fatalf("Failed to initialize CodeBuild service: %v", err)
	}
	log.Printf("CodeBuild service initialized with project: %s", codebuildProjectName)

	// Initialize ECS deployment orchestrator (optional - only if deploying to ECS)
	ecsOrchestrator, err := ecs.NewDeploymentOrchestrator(deploymentRepository, envVarRepository)
	if err != nil {
		log.Printf("Warning: ECS deployment orchestrator not initialized: %v", err)
		log.Printf("Deployments will only build images without deploying to ECS")
	} else {
		// Set up the deployment callback
		deploymentCallback := ecs.NewDeploymentCallbackAdapter(ecsOrchestrator)
		codebuildService.SetDeploymentCallback(deploymentCallback)
		log.Printf("ECS deployment orchestrator initialized successfully")
	}

	userHandler := handlers.NewUserHandler(userService)
	repositoryHandler := handlers.NewRepositoryHandler(repositoryService, clerkClient)
	projectHandler := handlers.NewProjectHandler(projectService, userService)
	envVarHandler := handlers.NewEnvVarHandler(envVarService, userService)
	deploymentHandler := handlers.NewDeploymentHandler(
		deploymentService, 
		userService, 
		codebuildService, 
		templateGenerator,
		projectRepository, 
		deploymentRepository,
	)

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

	// CORS middleware
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
		// Debug logging middleware
		v1.Use(func(c *gin.Context) {
			log.Printf("[ROUTE DEBUG] %s %s", c.Request.Method, c.Request.URL.Path)
			c.Next()
		})

		// Health check endpoint (no auth required)
		v1.GET("/health", healthHandler.Health)

		// Auth routes
		auth := v1.Group("/auth")
		auth.Use(authMiddleware.RequireAuth())
		{
			auth.GET("/me", userHandler.GetCurrentUser)
		}

		// User routes
		users := v1.Group("/users")
		users.Use(authMiddleware.RequireAuth())
		{
			users.GET("/:id/repos", repositoryHandler.GetUserRepositories)
			users.POST("/:id/repos/sync", repositoryHandler.SyncRepositories)
			users.GET("/:id/projects", projectHandler.GetUserProjects)
			users.POST("/:id/projects", projectHandler.CreateProject)
		}

		// Project routes
		projects := v1.Group("/projects")
		projects.Use(authMiddleware.RequireAuth())
		{
			projects.GET("/:id", projectHandler.GetProject)
			projects.PUT("/:id", projectHandler.UpdateProject)
			projects.DELETE("/:id", projectHandler.DeleteProject)
			projects.GET("/:id/deployments", deploymentHandler.GetProjectDeployments)
			projects.GET("/:id/deployments/latest", deploymentHandler.GetLatestProjectDeployment)
			// Environment variables
			projects.GET("/:id/env", envVarHandler.GetProjectEnvVars)
			projects.POST("/:id/env", envVarHandler.CreateOrUpdateEnvVar)
			projects.DELETE("/:id/env/:key", envVarHandler.DeleteEnvVar)
		}

		// Deployment routes
		deployments := v1.Group("/deployments")
		{
			// SSE endpoint - NO AUTH for now (outside middleware)
			deployments.GET("/:id/logs/stream", func(c *gin.Context) {
				log.Printf("[SSE] Endpoint hit directly - no auth middleware")
				deploymentHandler.StreamDeploymentLogs(c)
			})

			// Protected routes
			protectedDeployments := deployments.Group("")
			protectedDeployments.Use(authMiddleware.RequireAuth())
			{
				protectedDeployments.POST("", deploymentHandler.CreateDeployment)
				protectedDeployments.GET("/:id", deploymentHandler.GetDeployment)
				protectedDeployments.PATCH("/:id/status", deploymentHandler.UpdateDeploymentStatus)
				protectedDeployments.POST("/:id/logs", deploymentHandler.AppendDeploymentLog)
				protectedDeployments.DELETE("/:id", deploymentHandler.DeleteDeployment)
			}
		}

		// User deployment routes
		users.GET("/:id/deployments", deploymentHandler.GetUserDeployments)
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
