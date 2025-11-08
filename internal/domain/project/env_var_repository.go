package project

import "context"

// EnvironmentVariableRepository defines the interface for environment variable persistence
type EnvironmentVariableRepository interface {
	// Save persists an environment variable (create or update)
	Save(ctx context.Context, envVar *EnvironmentVariable) error

	// FindByProjectID retrieves all environment variables for a project
	FindByProjectID(ctx context.Context, projectID ProjectID) ([]*EnvironmentVariable, error)

	// FindByKey retrieves a specific environment variable by project and key
	FindByKey(ctx context.Context, projectID ProjectID, key EnvVarKey) (*EnvironmentVariable, error)

	// Delete removes an environment variable
	Delete(ctx context.Context, projectID ProjectID, key EnvVarKey) error

	// DeleteAll removes all environment variables for a project
	DeleteAll(ctx context.Context, projectID ProjectID) error

	// Count returns the number of environment variables for a project
	Count(ctx context.Context, projectID ProjectID) (int64, error)
}

