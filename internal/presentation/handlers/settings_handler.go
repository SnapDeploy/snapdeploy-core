package handlers

import (
	"net/http"

	"snapdeploy-core/internal/application/dto"
	"snapdeploy-core/internal/application/service"

	"github.com/gin-gonic/gin"
)

// SettingsHandler handles user settings HTTP requests
type SettingsHandler struct {
	settingsService *service.SettingsService
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(settingsService *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{
		settingsService: settingsService,
	}
}

// GetUserSettings handles GET /users/:id/settings
// @Summary Get user settings
// @Description Returns settings for a user
// @Tags Settings
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "User ID" format(uuid)
// @Success 200 {object} dto.UserSettingsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id}/settings [get]
func (h *SettingsHandler) GetUserSettings(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "User ID is required",
		})
		return
	}

	settings, err := h.settingsService.GetUserSettings(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get user settings",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateUserSettings handles PUT /users/:id/settings
// @Summary Update user settings
// @Description Updates settings for a user
// @Tags Settings
// @Accept json
// @Produce json
// @Security ClerkAuth
// @Param id path string true "User ID" format(uuid)
// @Param request body dto.UpdateUserSettingsRequest true "Settings to update"
// @Success 200 {object} dto.UserSettingsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id}/settings [put]
func (h *SettingsHandler) UpdateUserSettings(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "User ID is required",
		})
		return
	}

	var req dto.UpdateUserSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	settings, err := h.settingsService.UpdateUserSettings(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update user settings",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, settings)
}
