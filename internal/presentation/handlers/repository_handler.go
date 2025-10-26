package handlers

import (
	"net/http"
	"strconv"

	"snapdeploy-core/internal/application/service"
	"snapdeploy-core/internal/clerk"
	"snapdeploy-core/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RepositoryHandler handles repository-related HTTP requests
type RepositoryHandler struct {
	repositoryService *service.RepositoryService
	clerkClient       *clerk.Client
}

// NewRepositoryHandler creates a new repository handler
func NewRepositoryHandler(repositoryService *service.RepositoryService, clerkClient *clerk.Client) *RepositoryHandler {
	return &RepositoryHandler{
		repositoryService: repositoryService,
		clerkClient:       clerkClient,
	}
}

// SyncRepositories handles POST /users/:id/repos/sync
// @Summary Sync user repositories from GitHub
// @Description Syncs the repositories for a user from GitHub
// @Tags Repositories
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "User ID"
// @Success 200 {object} dto.RepositorySyncResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id}/repos/sync [post]
func (h *RepositoryHandler) SyncRepositories(c *gin.Context) {
	userID := c.Param("id")

	// Get clerk user from context
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

	// Get GitHub access token from Clerk for this user
	githubToken, err := h.clerkClient.GetGitHubAccessToken(c.Request.Context(), clerkUser.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "github_not_connected",
			Message: "GitHub account not connected. Please connect your GitHub account in your user profile settings.",
			Details: err.Error(),
		})
		return
	}

	// Sync repositories using application service
	response, err := h.repositoryService.SyncRepositoriesFromGitHub(c.Request.Context(), userID, githubToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "sync_failed",
			Message: "Failed to sync repositories from GitHub",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetUserRepositories handles GET /users/:id/repos
// @Summary Get user repositories with search
// @Description Returns the repositories for a user with pagination and optional search
// @Tags Repositories
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "User ID"
// @Param page query int false "Page number" default(1) minimum(1)
// @Param limit query int false "Items per page" default(20) minimum(1) maximum(100)
// @Param search query string false "Search query (searches name, full_name, description)"
// @Success 200 {object} dto.RepositoryListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /users/{id}/repos [get]
func (h *RepositoryHandler) GetUserRepositories(c *gin.Context) {
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

	// Get search query
	searchQuery := c.DefaultQuery("search", "")

	// Fetch repositories using application service
	response, err := h.repositoryService.GetRepositoriesByUserID(
		c.Request.Context(),
		userID,
		searchQuery,
		int32(page),
		int32(limit),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch repositories",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
