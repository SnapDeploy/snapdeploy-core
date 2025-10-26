package handlers

import (
	"net/http"

	"snapdeploy-core/internal/middleware"
	"snapdeploy-core/internal/models"
	"snapdeploy-core/internal/services"

	"github.com/gin-gonic/gin"
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
// @Security ClerkAuth
// @Success 200 {object} models.UserResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/me [get]
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	// Get user from context (set by auth middleware)
	clerkUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not found in context",
		})
		return
	}

	user, ok := clerkUser.(*middleware.ClerkUser)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user type in context",
		})
		return
	}

	// Get or create user in our database
	dbUser, err := h.userService.GetOrCreateUserByClerkID(c.Request.Context(), user)
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
