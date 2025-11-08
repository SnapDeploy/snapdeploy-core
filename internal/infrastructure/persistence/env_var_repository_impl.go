package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"snapdeploy-core/internal/database"
	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/infrastructure/encryption"
)

// EnvVarRepositoryImpl implements the project.EnvironmentVariableRepository interface
type EnvVarRepositoryImpl struct {
	db                *database.DB
	encryptionService *encryption.EncryptionService
}

// NewEnvVarRepository creates a new environment variable repository
func NewEnvVarRepository(db *database.DB, encryptionService *encryption.EncryptionService) project.EnvironmentVariableRepository {
	return &EnvVarRepositoryImpl{
		db:                db,
		encryptionService: encryptionService,
	}
}

// Save persists an environment variable (create or update)
func (r *EnvVarRepositoryImpl) Save(ctx context.Context, envVar *project.EnvironmentVariable) error {
	queries := database.New(r.db.GetConnection())

	// Encrypt the value before storing
	encryptedValue, err := r.encryptionService.Encrypt(envVar.Value().EncryptedValue())
	if err != nil {
		return fmt.Errorf("failed to encrypt value: %w", err)
	}

	// Check if environment variable exists
	_, err = queries.GetProjectEnvVar(ctx, &database.GetProjectEnvVarParams{
		ProjectID: envVar.ProjectID().UUID(),
		Key:       envVar.Key().String(),
	})

	if err == sql.ErrNoRows {
		// Create new
		_, err = queries.CreateProjectEnvVar(ctx, &database.CreateProjectEnvVarParams{
			ProjectID: envVar.ProjectID().UUID(),
			Key:       envVar.Key().String(),
			Value:     encryptedValue,
		})
		if err != nil {
			return fmt.Errorf("failed to create environment variable: %w", err)
		}
	} else if err == nil {
		// Update existing
		_, err = queries.UpdateProjectEnvVar(ctx, &database.UpdateProjectEnvVarParams{
			ProjectID: envVar.ProjectID().UUID(),
			Key:       envVar.Key().String(),
			Value:     encryptedValue,
		})
		if err != nil {
			return fmt.Errorf("failed to update environment variable: %w", err)
		}
	} else {
		return fmt.Errorf("failed to check environment variable existence: %w", err)
	}

	return nil
}

// FindByProjectID retrieves all environment variables for a project
func (r *EnvVarRepositoryImpl) FindByProjectID(ctx context.Context, projectID project.ProjectID) ([]*project.EnvironmentVariable, error) {
	queries := database.New(r.db.GetConnection())

	dbEnvVars, err := queries.GetProjectEnvVars(ctx, projectID.UUID())
	if err != nil {
		return nil, fmt.Errorf("failed to get environment variables: %w", err)
	}

	envVars := make([]*project.EnvironmentVariable, len(dbEnvVars))
	for i, dbEnvVar := range dbEnvVars {
		envVar, err := r.toDomain(dbEnvVar, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to convert environment variable: %w", err)
		}
		envVars[i] = envVar
	}

	return envVars, nil
}

// FindByKey retrieves a specific environment variable by project and key
func (r *EnvVarRepositoryImpl) FindByKey(ctx context.Context, projectID project.ProjectID, key project.EnvVarKey) (*project.EnvironmentVariable, error) {
	queries := database.New(r.db.GetConnection())

	dbEnvVar, err := queries.GetProjectEnvVar(ctx, &database.GetProjectEnvVarParams{
		ProjectID: projectID.UUID(),
		Key:       key.String(),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, project.ErrEnvVarNotFound
		}
		return nil, fmt.Errorf("failed to get environment variable: %w", err)
	}

	return r.toDomain(dbEnvVar, projectID)
}

// Delete removes an environment variable
func (r *EnvVarRepositoryImpl) Delete(ctx context.Context, projectID project.ProjectID, key project.EnvVarKey) error {
	queries := database.New(r.db.GetConnection())

	err := queries.DeleteProjectEnvVar(ctx, &database.DeleteProjectEnvVarParams{
		ProjectID: projectID.UUID(),
		Key:       key.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to delete environment variable: %w", err)
	}

	return nil
}

// DeleteAll removes all environment variables for a project
func (r *EnvVarRepositoryImpl) DeleteAll(ctx context.Context, projectID project.ProjectID) error {
	queries := database.New(r.db.GetConnection())

	err := queries.DeleteAllProjectEnvVars(ctx, projectID.UUID())
	if err != nil {
		return fmt.Errorf("failed to delete all environment variables: %w", err)
	}

	return nil
}

// Count returns the number of environment variables for a project
func (r *EnvVarRepositoryImpl) Count(ctx context.Context, projectID project.ProjectID) (int64, error) {
	queries := database.New(r.db.GetConnection())

	count, err := queries.CountProjectEnvVars(ctx, projectID.UUID())
	if err != nil {
		return 0, fmt.Errorf("failed to count environment variables: %w", err)
	}

	return count, nil
}

// toDomain converts database env var to domain env var (keeps value encrypted)
func (r *EnvVarRepositoryImpl) toDomain(dbEnvVar *database.ProjectEnvironmentVariable, projectID project.ProjectID) (*project.EnvironmentVariable, error) {
	return project.ReconstituteEnvVar(
		dbEnvVar.ID.String(),
		projectID,
		dbEnvVar.Key,
		dbEnvVar.Value, // Still encrypted
		dbEnvVar.CreatedAt,
		dbEnvVar.UpdatedAt,
	)
}

// DecryptValue decrypts an environment variable value (used when passing to containers)
func (r *EnvVarRepositoryImpl) DecryptValue(ctx context.Context, envVar *project.EnvironmentVariable) (string, error) {
	return r.encryptionService.Decrypt(envVar.Value().EncryptedValue())
}

// DecryptAll decrypts all environment variables for a project (used for deployments)
func (r *EnvVarRepositoryImpl) DecryptAll(ctx context.Context, projectID project.ProjectID) (map[string]string, error) {
	envVars, err := r.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, envVar := range envVars {
		plaintext, err := r.encryptionService.Decrypt(envVar.Value().EncryptedValue())
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt %s: %w", envVar.Key().String(), err)
		}
		result[envVar.Key().String()] = plaintext
	}

	return result, nil
}

