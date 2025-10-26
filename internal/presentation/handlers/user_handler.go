package handlers

import (
	"net/http"

	"snapdeploy-core/internal/application/service"
	"snapdeploy-core/internal/middleware"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *service.UserService) *UserHandler {
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
// @Success 200 {object} dto.UserResponse
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

	// Get or create user using application service
	dbUser, err := h.userService.GetOrCreateUserByClerkID(c.Request.Context(), user.GetUserID())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get user information",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dbUser)
}
