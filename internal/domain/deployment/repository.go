package deployment

import (
	"context"

	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/domain/user"
)

// DeploymentRepository defines the interface for deployment persistence
type DeploymentRepository interface {
	// Save persists a deployment (create or update)
	Save(ctx context.Context, deployment *Deployment) error

	// FindByID retrieves a deployment by its ID
	FindByID(ctx context.Context, id DeploymentID) (*Deployment, error)

	// FindByProjectID retrieves all deployments for a project with pagination
	FindByProjectID(ctx context.Context, projectID project.ProjectID, limit, offset int32) ([]*Deployment, error)

	// FindByUserID retrieves all deployments for a user with pagination
	FindByUserID(ctx context.Context, userID user.UserID, limit, offset int32) ([]*Deployment, error)

	// CountByProjectID counts total deployments for a project
	CountByProjectID(ctx context.Context, projectID project.ProjectID) (int64, error)

	// CountByUserID counts total deployments for a user
	CountByUserID(ctx context.Context, userID user.UserID) (int64, error)

	// Delete removes a deployment
	Delete(ctx context.Context, id DeploymentID) error

	// FindLatestByProjectID retrieves the most recent deployment for a project
	FindLatestByProjectID(ctx context.Context, projectID project.ProjectID) (*Deployment, error)
}

