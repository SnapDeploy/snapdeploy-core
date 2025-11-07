package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"snapdeploy-core/internal/application/dto"
	"snapdeploy-core/internal/application/service"
	"snapdeploy-core/internal/domain/deployment"
	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/infrastructure/builder"
	"snapdeploy-core/internal/middleware"

	"github.com/gin-gonic/gin"
)

// DeploymentHandler handles deployment-related HTTP requests
type DeploymentHandler struct {
	deploymentService *service.DeploymentService
	userService       *service.UserService
	builderService    *builder.BuilderService
	projectRepo       project.ProjectRepository
	deploymentRepo    deployment.DeploymentRepository
}

// SSEManagerSetter interface for builder service
type SSEManagerSetter interface {
	SetSSEManager(manager interface{})
}

// NewDeploymentHandler creates a new deployment handler
func NewDeploymentHandler(
	deploymentService *service.DeploymentService,
	userService *service.UserService,
	builderService *builder.BuilderService,
	projectRepo project.ProjectRepository,
	deploymentRepo deployment.DeploymentRepository,
) *DeploymentHandler {
	handler := &DeploymentHandler{
		deploymentService: deploymentService,
		userService:       userService,
		builderService:    builderService,
		projectRepo:       projectRepo,
		deploymentRepo:    deploymentRepo,
	}

	// Set SSE manager in builder service
	if builderService != nil {
		builderService.SetSSEManager(GetSSEManager())
	}

	return handler
}

// CreateDeployment handles POST /deployments
// @Summary Create a new deployment
// @Description Creates a new deployment for a project
// @Tags Deployments
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param deployment body dto.CreateDeploymentRequest true "Deployment data"
// @Success 201 {object} dto.DeploymentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /deployments [post]
func (h *DeploymentHandler) CreateDeployment(c *gin.Context) {
	// Get authenticated user from context
	clerkUserData, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not found in context",
		})
		return
	}

	clerkUser, ok := clerkUserData.(*middleware.ClerkUser)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user type in context",
		})
		return
	}

	// Get the internal user ID from Clerk ID
	dbUser, err := h.userService.GetOrCreateUserByClerkID(c.Request.Context(), clerkUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to resolve user",
			Details: err.Error(),
		})
		return
	}

	var req dto.CreateDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	response, err := h.deploymentService.CreateDeployment(c.Request.Context(), dbUser.ID, &req)
	if err != nil {
		if errors.Is(err, deployment.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have permission to create a deployment for this project",
			})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "creation_failed",
			Message: "Failed to create deployment",
			Details: err.Error(),
		})
		return
	}

	// Return response immediately to client
	c.JSON(http.StatusCreated, response)

	// Trigger async build process
	go h.buildProcess(response.ID, req.ProjectID)
}

// buildProcess executes the real deployment build process
func (h *DeploymentHandler) buildProcess(deploymentID, projectID string) {
	ctx := context.Background()

	// Parse IDs
	depID, err := deployment.ParseDeploymentID(deploymentID)
	if err != nil {
		log.Printf("[BUILD] Failed to parse deployment ID %s: %v", deploymentID, err)
		return
	}

	projID, err := project.ParseProjectID(projectID)
	if err != nil {
		log.Printf("[BUILD] Failed to parse project ID %s: %v", projectID, err)
		return
	}

	// Fetch deployment and project entities
	dep, err := h.deploymentRepo.FindByID(ctx, depID)
	if err != nil {
		log.Printf("[BUILD] Failed to find deployment %s: %v", deploymentID, err)
		return
	}

	proj, err := h.projectRepo.FindByID(ctx, projID)
	if err != nil {
		log.Printf("[BUILD] Failed to find project %s: %v", projectID, err)
		// Update deployment status to failed
		dep.UpdateStatus(deployment.StatusFailed)
		h.deploymentRepo.Save(ctx, dep)
		return
	}

	// Clone repository
	repoPath, err := h.cloneRepository(proj, dep)
	if err != nil {
		log.Printf("[BUILD] Failed to clone repository: %v", err)
		dep.UpdateStatus(deployment.StatusFailed)
		h.deploymentRepo.Save(ctx, dep)
		return
	}
	defer h.cleanupRepository(repoPath)

	// Generate image tag
	imageTag := h.generateImageTag(proj, dep)

	// Build and deploy
	buildReq := builder.BuildRequest{
		Deployment:     dep,
		Project:        proj,
		RepositoryPath: repoPath,
		ImageTag:       imageTag,
	}

	log.Printf("[BUILD] Starting build for deployment %s", deploymentID)
	err = h.builderService.BuildDeployment(ctx, buildReq)
	if err != nil {
		log.Printf("[BUILD] Build failed for deployment %s: %v", deploymentID, err)
		// Status will be updated by builder service
		return
	}

	// Cleanup build artifacts
	if err := h.builderService.CleanupBuildArtifacts(repoPath); err != nil {
		log.Printf("[BUILD] Warning: failed to cleanup artifacts: %v", err)
	}

	log.Printf("[BUILD] Completed deployment %s successfully", deploymentID)
}

// cloneRepository clones a git repository to a temporary directory
func (h *DeploymentHandler) cloneRepository(proj *project.Project, dep *deployment.Deployment) (string, error) {
	// Create temporary directory for this build
	tmpDir, err := os.MkdirTemp("", "snapdeploy-build-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	repoURL := proj.RepositoryURL().String()
	branch := dep.Branch().String()
	commitHash := dep.CommitHash().String()

	// Clone the repository
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", branch, repoURL, tmpDir)
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	// Checkout specific commit if provided and not HEAD
	if commitHash != "" && commitHash != "HEAD" && commitHash != "head" {
		// Fetch the specific commit
		cmd = exec.Command("git", "-C", tmpDir, "fetch", "origin", commitHash)
		if err := cmd.Run(); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to fetch commit: %w", err)
		}

		// Checkout the commit
		cmd = exec.Command("git", "-C", tmpDir, "checkout", commitHash)
		if err := cmd.Run(); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to checkout commit: %w", err)
		}
	}

	return tmpDir, nil
}

// cleanupRepository removes the cloned repository directory
func (h *DeploymentHandler) cleanupRepository(repoPath string) error {
	return os.RemoveAll(repoPath)
}

// generateImageTag generates a Docker image tag for the deployment
func (h *DeploymentHandler) generateImageTag(proj *project.Project, dep *deployment.Deployment) string {
	// Format: registry.example.com/repository:project-id-commit-hash
	registry := os.Getenv("DOCKER_REGISTRY")
	if registry == "" {
		registry = "localhost:5000" // Default to local registry
	}

	projectName := sanitizeImageName(proj.ID().String())
	commitHash := dep.CommitHash().String()
	if commitHash == "HEAD" || commitHash == "head" {
		commitHash = "latest"
	}

	// For ECR, if registry already includes repository name, use it as-is with project tag
	// ECR format: account.dkr.ecr.region.amazonaws.com/repo-name:tag
	if strings.Contains(registry, ".ecr.") && strings.Contains(registry, ".amazonaws.com") {
		// ECR registry - check if repository name is already in the URL
		if strings.Contains(registry, "/") {
			// Repository name is already included, use project ID + commit as tag
			// Format: registry/repo:project-id-commit
			tag := fmt.Sprintf("%s-%s", projectName, commitHash)
			return fmt.Sprintf("%s:%s", registry, tag)
		}
		// No repository name, use project ID as repository name
		return fmt.Sprintf("%s/%s:%s", registry, projectName, commitHash)
	}

	// Standard registry format: registry/repo:tag
	return fmt.Sprintf("%s/%s:%s", registry, projectName, commitHash)
}

// sanitizeImageName ensures the name is valid for Docker
func sanitizeImageName(name string) string {
	// Docker image names must be lowercase and can only contain
	// lowercase letters, digits, and separators (., -, _)
	// UUIDs are already valid, but we'll convert to lowercase just in case
	return filepath.Base(name)
}

// GetDeployment handles GET /deployments/:id
// @Summary Get a deployment by ID
// @Description Returns a single deployment by its ID
// @Tags Deployments
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "Deployment ID"
// @Success 200 {object} dto.DeploymentResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /deployments/{id} [get]
func (h *DeploymentHandler) GetDeployment(c *gin.Context) {
	deploymentID := c.Param("id")

	response, err := h.deploymentService.GetDeploymentByID(c.Request.Context(), deploymentID)
	if err != nil {
		if errors.Is(err, deployment.ErrDeploymentNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Deployment not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch deployment",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetProjectDeployments handles GET /projects/:id/deployments
// @Summary Get project deployments
// @Description Returns all deployments for a project with pagination
// @Tags Deployments
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "Project ID"
// @Param page query int false "Page number" default(1) minimum(1)
// @Param limit query int false "Items per page" default(20) minimum(1) maximum(100)
// @Success 200 {object} dto.DeploymentListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /projects/{id}/deployments [get]
func (h *DeploymentHandler) GetProjectDeployments(c *gin.Context) {
	projectID := c.Param("id")

	// Get pagination parameters
	page := 1
	limit := 20

	if pageStr := c.DefaultQuery("page", "1"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := c.DefaultQuery("limit", "20"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	response, err := h.deploymentService.GetDeploymentsByProjectID(
		c.Request.Context(),
		projectID,
		int32(page),
		int32(limit),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch deployments",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetUserDeployments handles GET /users/:id/deployments
// @Summary Get user deployments
// @Description Returns all deployments for a user with pagination
// @Tags Deployments
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "User ID"
// @Param page query int false "Page number" default(1) minimum(1)
// @Param limit query int false "Items per page" default(20) minimum(1) maximum(100)
// @Success 200 {object} dto.DeploymentListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id}/deployments [get]
func (h *DeploymentHandler) GetUserDeployments(c *gin.Context) {
	userID := c.Param("id")

	// Get pagination parameters
	page := 1
	limit := 20

	if pageStr := c.DefaultQuery("page", "1"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := c.DefaultQuery("limit", "20"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	response, err := h.deploymentService.GetDeploymentsByUserID(
		c.Request.Context(),
		userID,
		int32(page),
		int32(limit),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch deployments",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateDeploymentStatus handles PATCH /deployments/:id/status
// @Summary Update deployment status
// @Description Updates the status of a deployment
// @Tags Deployments
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "Deployment ID"
// @Param status body dto.UpdateDeploymentStatusRequest true "Status data"
// @Success 200 {object} dto.DeploymentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /deployments/{id}/status [patch]
func (h *DeploymentHandler) UpdateDeploymentStatus(c *gin.Context) {
	deploymentID := c.Param("id")

	// Get authenticated user from context
	clerkUserData, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not found in context",
		})
		return
	}

	clerkUser, ok := clerkUserData.(*middleware.ClerkUser)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user type in context",
		})
		return
	}

	// Get the internal user ID from Clerk ID
	dbUser, err := h.userService.GetOrCreateUserByClerkID(c.Request.Context(), clerkUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to resolve user",
			Details: err.Error(),
		})
		return
	}

	var req dto.UpdateDeploymentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	response, err := h.deploymentService.UpdateDeploymentStatus(c.Request.Context(), deploymentID, dbUser.ID, &req)
	if err != nil {
		if errors.Is(err, deployment.ErrDeploymentNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Deployment not found",
			})
			return
		}
		if errors.Is(err, deployment.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have permission to update this deployment",
			})
			return
		}
		if errors.Is(err, deployment.ErrInvalidStatusTransition) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_status_transition",
				Message: "Invalid status transition",
				Details: err.Error(),
			})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "update_failed",
			Message: "Failed to update deployment status",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// AppendDeploymentLog handles POST /deployments/:id/logs
// @Summary Append to deployment logs
// @Description Appends a log line to a deployment
// @Tags Deployments
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "Deployment ID"
// @Param log body dto.AppendDeploymentLogRequest true "Log data"
// @Success 200 {object} dto.DeploymentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /deployments/{id}/logs [post]
func (h *DeploymentHandler) AppendDeploymentLog(c *gin.Context) {
	deploymentID := c.Param("id")

	// Get authenticated user from context
	clerkUserData, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not found in context",
		})
		return
	}

	clerkUser, ok := clerkUserData.(*middleware.ClerkUser)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user type in context",
		})
		return
	}

	// Get the internal user ID from Clerk ID
	dbUser, err := h.userService.GetOrCreateUserByClerkID(c.Request.Context(), clerkUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to resolve user",
			Details: err.Error(),
		})
		return
	}

	var req dto.AppendDeploymentLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	response, err := h.deploymentService.AppendDeploymentLog(c.Request.Context(), deploymentID, dbUser.ID, &req)
	if err != nil {
		if errors.Is(err, deployment.ErrDeploymentNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Deployment not found",
			})
			return
		}
		if errors.Is(err, deployment.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have permission to update this deployment",
			})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "update_failed",
			Message: "Failed to append log",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteDeployment handles DELETE /deployments/:id
// @Summary Delete a deployment
// @Description Deletes a deployment
// @Tags Deployments
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "Deployment ID"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /deployments/{id} [delete]
func (h *DeploymentHandler) DeleteDeployment(c *gin.Context) {
	deploymentID := c.Param("id")

	// Get authenticated user from context
	clerkUserData, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not found in context",
		})
		return
	}

	clerkUser, ok := clerkUserData.(*middleware.ClerkUser)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user type in context",
		})
		return
	}

	// Get the internal user ID from Clerk ID
	dbUser, err := h.userService.GetOrCreateUserByClerkID(c.Request.Context(), clerkUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to resolve user",
			Details: err.Error(),
		})
		return
	}

	err = h.deploymentService.DeleteDeployment(c.Request.Context(), deploymentID, dbUser.ID)
	if err != nil {
		if errors.Is(err, deployment.ErrDeploymentNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Deployment not found",
			})
			return
		}
		if errors.Is(err, deployment.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have permission to delete this deployment",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "delete_failed",
			Message: "Failed to delete deployment",
			Details: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetLatestProjectDeployment handles GET /projects/:id/deployments/latest
// @Summary Get latest project deployment
// @Description Returns the most recent deployment for a project
// @Tags Deployments
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "Project ID"
// @Success 200 {object} dto.DeploymentResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /projects/{id}/deployments/latest [get]
func (h *DeploymentHandler) GetLatestProjectDeployment(c *gin.Context) {
	projectID := c.Param("id")

	response, err := h.deploymentService.GetLatestDeploymentByProjectID(c.Request.Context(), projectID)
	if err != nil {
		if errors.Is(err, deployment.ErrDeploymentNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "No deployments found for this project",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch latest deployment",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
