package repo

import (
	"context"

	"snapdeploy-core/internal/domain/user"
)

// RepositoryRepo defines the interface for repository persistence
// This is defined in the domain layer, but implemented in infrastructure
type RepositoryRepo interface {
	// Save persists a repository (create or update)
	Save(ctx context.Context, repo *Repository) error

	// FindByID retrieves a repository by its ID
	FindByID(ctx context.Context, id RepositoryID) (*Repository, error)

	// FindByUserID retrieves repositories for a specific user with pagination
	FindByUserID(ctx context.Context, userID user.UserID, limit, offset int32) ([]*Repository, error)

	// SearchByUserID searches repositories for a user with optional search query
	SearchByUserID(ctx context.Context, userID user.UserID, searchQuery string, limit, offset int32) ([]*Repository, error)

	// CountByUserID returns the total number of repositories for a user
	CountByUserID(ctx context.Context, userID user.UserID) (int64, error)

	// CountSearchByUserID returns the total number of repositories matching search for a user
	CountSearchByUserID(ctx context.Context, userID user.UserID, searchQuery string) (int64, error)

	// FindByURL retrieves a repository by its URL
	FindByURL(ctx context.Context, url URL) (*Repository, error)

	// Delete removes a repository from persistence
	Delete(ctx context.Context, id RepositoryID) error
}
