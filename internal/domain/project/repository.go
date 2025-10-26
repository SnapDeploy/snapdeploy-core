package project

import (
	"context"

	"snapdeploy-core/internal/domain/user"
)

// ProjectRepository defines the interface for project persistence
type ProjectRepository interface {
	// Save persists a project (create or update)
	Save(ctx context.Context, project *Project) error

	// FindByID retrieves a project by its ID
	FindByID(ctx context.Context, id ProjectID) (*Project, error)

	// FindByUserID retrieves all projects for a user with pagination
	FindByUserID(ctx context.Context, userID user.UserID, limit, offset int32) ([]*Project, error)

	// FindByRepositoryURL retrieves a project by repository URL and user ID
	FindByRepositoryURL(ctx context.Context, userID user.UserID, repoURL RepositoryURL) (*Project, error)

	// CountByUserID counts total projects for a user
	CountByUserID(ctx context.Context, userID user.UserID) (int64, error)

	// Delete removes a project
	Delete(ctx context.Context, id ProjectID) error

	// ExistsByRepositoryURL checks if a project with the given repository URL exists for a user
	ExistsByRepositoryURL(ctx context.Context, userID user.UserID, repoURL RepositoryURL) (bool, error)
}
