package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"snapdeploy-core/internal/application/dto"
	"snapdeploy-core/internal/application/service"
	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/middleware"

	"github.com/gin-gonic/gin"
)

// ProjectHandler handles project-related HTTP requests
type ProjectHandler struct {
	projectService *service.ProjectService
	userService    *service.UserService
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(projectService *service.ProjectService, userService *service.UserService) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
		userService:    userService,
	}
}

// CreateProject handles POST /users/:id/projects
// @Summary Create a new project
// @Description Creates a new project for a user
// @Tags Projects
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "User ID"
// @Param project body dto.CreateProjectRequest true "Project data"
// @Success 201 {object} dto.ProjectResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id}/projects [post]
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	userID := c.Param("id")

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

	// Verify user is creating project for themselves
	if dbUser.ID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You can only create projects for yourself",
		})
		return
	}

	var req dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	response, err := h.projectService.CreateProject(c.Request.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, project.ErrProjectAlreadyExists) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "project_exists",
				Message: "A project with this repository URL already exists",
			})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "creation_failed",
			Message: "Failed to create project",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetProject handles GET /projects/:id
// @Summary Get a project by ID
// @Description Returns a single project by its ID
// @Tags Projects
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "Project ID"
// @Success 200 {object} dto.ProjectResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /projects/{id} [get]
func (h *ProjectHandler) GetProject(c *gin.Context) {
	projectID := c.Param("id")

	response, err := h.projectService.GetProjectByID(c.Request.Context(), projectID)
	if err != nil {
		if errors.Is(err, project.ErrProjectNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Project not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch project",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetUserProjects handles GET /users/:id/projects
// @Summary Get user projects
// @Description Returns all projects for a user with pagination
// @Tags Projects
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "User ID"
// @Param page query int false "Page number" default(1) minimum(1)
// @Param limit query int false "Items per page" default(20) minimum(1) maximum(100)
// @Success 200 {object} dto.ProjectListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id}/projects [get]
func (h *ProjectHandler) GetUserProjects(c *gin.Context) {
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

	response, err := h.projectService.GetProjectsByUserID(
		c.Request.Context(),
		userID,
		int32(page),
		int32(limit),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch projects",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateProject handles PUT /projects/:id
// @Summary Update a project
// @Description Updates an existing project
// @Tags Projects
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "Project ID"
// @Param project body dto.UpdateProjectRequest true "Project data"
// @Success 200 {object} dto.ProjectResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /projects/{id} [put]
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	projectID := c.Param("id")

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

	var req dto.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	response, err := h.projectService.UpdateProject(c.Request.Context(), projectID, dbUser.ID, &req)
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
				Message: "You don't have permission to update this project",
			})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "update_failed",
			Message: "Failed to update project",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteProject handles DELETE /projects/:id
// @Summary Delete a project
// @Description Deletes a project
// @Tags Projects
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "Project ID"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /projects/{id} [delete]
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	projectID := c.Param("id")

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

	err = h.projectService.DeleteProject(c.Request.Context(), projectID, dbUser.ID)
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
				Message: "You don't have permission to delete this project",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "delete_failed",
			Message: "Failed to delete project",
			Details: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}
