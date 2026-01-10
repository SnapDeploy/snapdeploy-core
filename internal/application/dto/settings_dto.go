package dto

import "time"

// UserSettingsResponse represents user settings in API responses
type UserSettingsResponse struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	DefaultBranch string    `json:"default_branch"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UpdateUserSettingsRequest represents a request to update user settings
type UpdateUserSettingsRequest struct {
	DefaultBranch *string `json:"default_branch,omitempty" validate:"omitempty,min=1,max=255"`
}
