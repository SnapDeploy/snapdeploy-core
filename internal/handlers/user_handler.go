package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"snapdeploy-core/internal/middleware"
	"snapdeploy-core/internal/models"
	"snapdeploy-core/internal/services"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetCurrentUser handles GET /auth/me
// @Summary Get current user information
// @Description Returns information about the currently authenticated user
// @Tags Authentication
// @Accept json
// @Produce json
// @Security CognitoAuth
// @Success 200 {object} models.UserResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/me [get]
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	// Get user from context (set by auth middleware)
	cognitoUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not found in context",
		})
		return
	}

	user, ok := cognitoUser.(*middleware.CognitoUser)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user type in context",
		})
		return
	}

	// Get or create user in our database
	dbUser, err := h.userService.GetOrCreateUserByCognitoID(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get user information",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dbUser.ToResponse())
}

// ListUsers handles GET /users
// @Summary List users
// @Description Retrieve a paginated list of users
// @Tags Users
// @Accept json
// @Produce json
// @Security CognitoAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of items per page" default(20)
// @Success 200 {object} UserListResponse
// @Failure 401 {object} ErrorResponse
// @Router /users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	// Parse query parameters
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.ParseInt(pageStr, 10, 32)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.ParseInt(limitStr, 10, 32)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	// Get users from service
	users, total, err := h.userService.ListUsers(c.Request.Context(), int32(page), int32(limit))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve users",
			Details: err.Error(),
		})
		return
	}

	// Convert to response format
	userResponses := make([]*models.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = user.ToResponse()
	}

	// Calculate pagination info
	totalPages := (total + int64(limit) - 1) / int64(limit)

	response := UserListResponse{
		Users: userResponses,
		Pagination: PaginationResponse{
			Page:       int32(page),
			Limit:      int32(limit),
			Total:      total,
			TotalPages: totalPages,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetUserByID handles GET /users/:id
// @Summary Get user by ID
// @Description Retrieve a specific user by its ID
// @Tags Users
// @Accept json
// @Produce json
// @Security CognitoAuth
// @Param id path string true "User ID"
// @Success 200 {object} models.UserResponse
// @Failure 404 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /users/{id} [get]
func (h *UserHandler) GetUserByID(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "User ID is required",
		})
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "User not found",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// UpdateUser handles PUT /users/:id
// @Summary Update user
// @Description Update an existing user
// @Tags Users
// @Accept json
// @Produce json
// @Security CognitoAuth
// @Param id path string true "User ID"
// @Param request body models.UpdateUserRequest true "Update user request"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "User ID is required",
		})
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), userID, &req)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "User not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update user",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// DeleteUser handles DELETE /users/:id
// @Summary Delete user
// @Description Delete a user by its ID
// @Tags Users
// @Accept json
// @Produce json
// @Security CognitoAuth
// @Param id path string true "User ID"
// @Success 204 "User deleted successfully"
// @Failure 404 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "User ID is required",
		})
		return
	}

	err := h.userService.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "User not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete user",
			Details: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// Response types
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type UserListResponse struct {
	Users      []*models.UserResponse `json:"users"`
	Pagination PaginationResponse     `json:"pagination"`
}

type PaginationResponse struct {
	Page       int32 `json:"page"`
	Limit      int32 `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}
