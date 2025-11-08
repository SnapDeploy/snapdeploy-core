package service

import (
	"context"
	"fmt"
	"time"

	"snapdeploy-core/internal/application/dto"
	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/domain/user"
	"snapdeploy-core/internal/infrastructure/encryption"
)

// EnvVarService handles environment variable use cases
type EnvVarService struct {
	envVarRepo        project.EnvironmentVariableRepository
	projectRepo       project.ProjectRepository
	encryptionService *encryption.EncryptionService
}

// NewEnvVarService creates a new environment variable service
func NewEnvVarService(
	envVarRepo project.EnvironmentVariableRepository,
	projectRepo project.ProjectRepository,
	encryptionService *encryption.EncryptionService,
) *EnvVarService {
	return &EnvVarService{
		envVarRepo:        envVarRepo,
		projectRepo:       projectRepo,
		encryptionService: encryptionService,
	}
}

// CreateOrUpdateEnvVar creates or updates an environment variable
func (s *EnvVarService) CreateOrUpdateEnvVar(
	ctx context.Context,
	projectID, userID string,
	req *dto.CreateEnvVarRequest,
) (*dto.EnvVarResponse, error) {
	// Parse IDs
	pid, err := project.ParseProjectID(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	uid, err := user.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Verify project exists and belongs to user
	proj, err := s.projectRepo.FindByID(ctx, pid)
	if err != nil {
		return nil, err
	}

	if !proj.BelongsToUser(uid) {
		return nil, project.ErrUnauthorized
	}

	// Create environment variable entity
	envVar, err := project.NewEnvironmentVariable(pid, req.Key, req.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to create environment variable: %w", err)
	}

	// Save (will be encrypted in repository)
	if err := s.envVarRepo.Save(ctx, envVar); err != nil {
		return nil, fmt.Errorf("failed to save environment variable: %w", err)
	}

	return s.toDTO(envVar), nil
}

// GetProjectEnvVars retrieves all environment variables for a project
func (s *EnvVarService) GetProjectEnvVars(
	ctx context.Context,
	projectID, userID string,
) (*dto.EnvVarListResponse, error) {
	// Parse IDs
	pid, err := project.ParseProjectID(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	uid, err := user.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Verify project belongs to user
	proj, err := s.projectRepo.FindByID(ctx, pid)
	if err != nil {
		return nil, err
	}

	if !proj.BelongsToUser(uid) {
		return nil, project.ErrUnauthorized
	}

	// Get all environment variables
	envVars, err := s.envVarRepo.FindByProjectID(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment variables: %w", err)
	}

	// Convert to DTOs with masked values
	envVarResponses := make([]*dto.EnvVarResponse, len(envVars))
	for i, envVar := range envVars {
		envVarResponses[i] = s.toDTO(envVar)
	}

	count, _ := s.envVarRepo.Count(ctx, pid)

	return &dto.EnvVarListResponse{
		EnvironmentVariables: envVarResponses,
		Count:                count,
	}, nil
}

// DeleteEnvVar deletes an environment variable
func (s *EnvVarService) DeleteEnvVar(
	ctx context.Context,
	projectID, userID, key string,
) error {
	// Parse IDs
	pid, err := project.ParseProjectID(projectID)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	uid, err := user.ParseUserID(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	envKey, err := project.NewEnvVarKey(key)
	if err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	// Verify project belongs to user
	proj, err := s.projectRepo.FindByID(ctx, pid)
	if err != nil {
		return err
	}

	if !proj.BelongsToUser(uid) {
		return project.ErrUnauthorized
	}

	// Delete environment variable
	if err := s.envVarRepo.Delete(ctx, pid, envKey); err != nil {
		return fmt.Errorf("failed to delete environment variable: %w", err)
	}

	return nil
}

// toDTO converts domain env var to DTO with masked value
func (s *EnvVarService) toDTO(envVar *project.EnvironmentVariable) *dto.EnvVarResponse {
	// Decrypt value to mask it properly
	decrypted, err := s.encryptionService.Decrypt(envVar.Value().EncryptedValue())
	
	// Generate masked value: first_char*******last_char
	maskedValue := maskValue(decrypted, err)

	return &dto.EnvVarResponse{
		ID:        envVar.ID().String(),
		ProjectID: envVar.ProjectID().String(),
		Key:       envVar.Key().String(),
		Value:     maskedValue,
		CreatedAt: envVar.CreatedAt().Format(time.RFC3339),
		UpdatedAt: envVar.UpdatedAt().Format(time.RFC3339),
	}
}

// maskValue masks a value for display: first_char*******last_char
func maskValue(value string, decryptErr error) string {
	// If decryption failed or value is empty, show generic mask
	if decryptErr != nil || value == "" {
		return "********"
	}

	// For very short values (1-2 chars), mask completely
	if len(value) <= 2 {
		return "***"
	}

	// For longer values: first char + ******* + last char
	first := string(value[0])
	last := string(value[len(value)-1])
	
	return fmt.Sprintf("%s*******%s", first, last)
}

