package service

import (
	"context"
	"database/sql"
	"fmt"

	"snapdeploy-core/internal/application/dto"
	"snapdeploy-core/internal/database"

	"github.com/google/uuid"
)

// SettingsService handles user settings use cases
type SettingsService struct {
	db *database.DB
}

// NewSettingsService creates a new settings service
func NewSettingsService(db *database.DB) *SettingsService {
	return &SettingsService{
		db: db,
	}
}

// GetUserSettings retrieves settings for a user, creating defaults if none exist
func (s *SettingsService) GetUserSettings(ctx context.Context, userID string) (*dto.UserSettingsResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	queries := database.New(s.db.GetConnection())

	// Try to get existing settings
	settings, err := queries.GetUserSettingsByUserID(ctx, uid)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create default settings
			settings, err = queries.CreateUserSettings(ctx, &database.CreateUserSettingsParams{
				ID:            uuid.New(),
				UserID:        uid,
				DefaultBranch: sql.NullString{String: "main", Valid: true},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create default settings: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get user settings: %w", err)
		}
	}

	return s.toDTO(settings), nil
}

// UpdateUserSettings updates settings for a user
func (s *SettingsService) UpdateUserSettings(ctx context.Context, userID string, req *dto.UpdateUserSettingsRequest) (*dto.UserSettingsResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	queries := database.New(s.db.GetConnection())

	// Build the default branch value
	defaultBranch := sql.NullString{String: "main", Valid: true}
	if req.DefaultBranch != nil && *req.DefaultBranch != "" {
		defaultBranch = sql.NullString{String: *req.DefaultBranch, Valid: true}
	}

	// Upsert settings (create if not exists, update if exists)
	settings, err := queries.UpsertUserSettings(ctx, &database.UpsertUserSettingsParams{
		ID:            uuid.New(),
		UserID:        uid,
		DefaultBranch: defaultBranch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update user settings: %w", err)
	}

	return s.toDTO(settings), nil
}

// toDTO converts database model to DTO
func (s *SettingsService) toDTO(settings *database.UserSetting) *dto.UserSettingsResponse {
	defaultBranch := "main"
	if settings.DefaultBranch.Valid {
		defaultBranch = settings.DefaultBranch.String
	}

	createdAt := settings.CreatedAt.Time
	updatedAt := settings.UpdatedAt.Time

	return &dto.UserSettingsResponse{
		ID:            settings.ID.String(),
		UserID:        settings.UserID.String(),
		DefaultBranch: defaultBranch,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}
