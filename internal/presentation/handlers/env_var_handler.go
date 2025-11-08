package handlers

import (
	"errors"
	"net/http"

	"snapdeploy-core/internal/application/dto"
	"snapdeploy-core/internal/application/service"
	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/middleware"

	"github.com/gin-gonic/gin"
)

// EnvVarHandler handles environment variable HTTP requests
type EnvVarHandler struct {
	envVarService *service.EnvVarService
	userService   *service.UserService
}

// NewEnvVarHandler creates a new environment variable handler
func NewEnvVarHandler(
	envVarService *service.EnvVarService,
	userService *service.UserService,
) *EnvVarHandler {
	return &EnvVarHandler{
		envVarService: envVarService,
		userService:   userService,
	}
}

// GetProjectEnvVars handles GET /projects/:id/env
// @Summary Get project environment variables
// @Description Returns all environment variables for a project (values are masked)
// @Tags Environment Variables
// @Security ClerkAuth
// @Param id path string true "Project ID"
// @Success 200 {object} dto.EnvVarListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /projects/{id}/env [get]
func (h *EnvVarHandler) GetProjectEnvVars(c *gin.Context) {
	projectID := c.Param("id")

	// Get authenticated user
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

	// Get internal user ID
	dbUser, err := h.userService.GetOrCreateUserByClerkID(c.Request.Context(), clerkUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to resolve user",
			Details: err.Error(),
		})
		return
	}

	response, err := h.envVarService.GetProjectEnvVars(c.Request.Context(), projectID, dbUser.ID)
	if err != nil {
		if errors.Is(err, project.ErrProjectNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Project not found",
			})
			return
		}
		if errors.Is(err, project.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have permission to access this project",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get environment variables",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CreateOrUpdateEnvVar handles POST /projects/:id/env
// @Summary Create or update an environment variable
// @Description Creates or updates an environment variable for a project
// @Tags Environment Variables
// @Security ClerkAuth
// @Param id path string true "Project ID"
// @Param env_var body dto.CreateEnvVarRequest true "Environment variable data"
// @Success 200 {object} dto.EnvVarResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /projects/{id}/env [post]
func (h *EnvVarHandler) CreateOrUpdateEnvVar(c *gin.Context) {
	projectID := c.Param("id")

	// Get authenticated user
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

	// Get internal user ID
	dbUser, err := h.userService.GetOrCreateUserByClerkID(c.Request.Context(), clerkUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to resolve user",
			Details: err.Error(),
		})
		return
	}

	var req dto.CreateEnvVarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	response, err := h.envVarService.CreateOrUpdateEnvVar(c.Request.Context(), projectID, dbUser.ID, &req)
	if err != nil {
		if errors.Is(err, project.ErrProjectNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Project not found",
			})
			return
		}
		if errors.Is(err, project.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have permission to modify this project",
			})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "creation_failed",
			Message: "Failed to create/update environment variable",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteEnvVar handles DELETE /projects/:id/env/:key
// @Summary Delete an environment variable
// @Description Deletes an environment variable from a project
// @Tags Environment Variables
// @Security ClerkAuth
// @Param id path string true "Project ID"
// @Param key path string true "Environment variable key"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /projects/{id}/env/{key} [delete]
func (h *EnvVarHandler) DeleteEnvVar(c *gin.Context) {
	projectID := c.Param("id")
	key := c.Param("key")

	// Get authenticated user
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

	// Get internal user ID
	dbUser, err := h.userService.GetOrCreateUserByClerkID(c.Request.Context(), clerkUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to resolve user",
			Details: err.Error(),
		})
		return
	}

	err = h.envVarService.DeleteEnvVar(c.Request.Context(), projectID, dbUser.ID, key)
	if err != nil {
		if errors.Is(err, project.ErrProjectNotFound) || errors.Is(err, project.ErrEnvVarNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Environment variable not found",
			})
			return
		}
		if errors.Is(err, project.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have permission to modify this project",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "delete_failed",
			Message: "Failed to delete environment variable",
			Details: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

