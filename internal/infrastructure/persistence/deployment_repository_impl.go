package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"snapdeploy-core/internal/database"
	"snapdeploy-core/internal/domain/deployment"
	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/domain/user"
)

// DeploymentRepositoryImpl implements the domain deployment.DeploymentRepository interface
type DeploymentRepositoryImpl struct {
	db *database.DB
}

// NewDeploymentRepository creates a new deployment repository implementation
func NewDeploymentRepository(db *database.DB) deployment.DeploymentRepository {
	return &DeploymentRepositoryImpl{db: db}
}

// Save persists a deployment (create or update)
func (r *DeploymentRepositoryImpl) Save(ctx context.Context, dep *deployment.Deployment) error {
	queries := database.New(r.db.GetConnection())

	// Check if deployment exists
	_, err := queries.GetDeploymentByID(ctx, dep.ID().UUID())
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check if deployment exists: %w", err)
	}

	// If no error, deployment exists - update it
	if err == nil {
		// Update existing deployment
		err := queries.UpdateDeployment(ctx, &database.UpdateDeploymentParams{
			ID:        dep.ID().UUID(),
			Status:    dep.Status().String(),
			Logs:      sql.NullString{String: dep.Logs().String(), Valid: true},
			UpdatedAt: sql.NullTime{Time: dep.UpdatedAt(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update deployment: %w", err)
		}
	} else {
		// Deployment doesn't exist (err == sql.ErrNoRows) - create it
		_, err := queries.CreateDeployment(ctx, &database.CreateDeploymentParams{
			ID:         dep.ID().UUID(),
			ProjectID:  dep.ProjectID().UUID(),
			UserID:     dep.UserID().UUID(),
			CommitHash: dep.CommitHash().String(),
			Branch:     dep.Branch().String(),
			Status:     dep.Status().String(),
			Logs:       sql.NullString{String: dep.Logs().String(), Valid: true},
			CreatedAt:  sql.NullTime{Time: dep.CreatedAt(), Valid: true},
			UpdatedAt:  sql.NullTime{Time: dep.UpdatedAt(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
	}

	return nil
}

// FindByID retrieves a deployment by its ID
func (r *DeploymentRepositoryImpl) FindByID(ctx context.Context, id deployment.DeploymentID) (*deployment.Deployment, error) {
	queries := database.New(r.db.GetConnection())

	dbDeployment, err := queries.GetDeploymentByID(ctx, id.UUID())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, deployment.ErrDeploymentNotFound
		}
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	return r.toDomain(dbDeployment)
}

// FindByProjectID retrieves all deployments for a project with pagination
func (r *DeploymentRepositoryImpl) FindByProjectID(ctx context.Context, projectID project.ProjectID, limit, offset int32) ([]*deployment.Deployment, error) {
	queries := database.New(r.db.GetConnection())

	dbDeployments, err := queries.GetDeploymentsByProjectID(ctx, &database.GetDeploymentsByProjectIDParams{
		ProjectID: projectID.UUID(),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}

	deployments := make([]*deployment.Deployment, len(dbDeployments))
	for i, dbDeployment := range dbDeployments {
		domainDeployment, err := r.toDomain(dbDeployment)
		if err != nil {
			return nil, fmt.Errorf("failed to convert deployment: %w", err)
		}
		deployments[i] = domainDeployment
	}

	return deployments, nil
}

// FindByUserID retrieves all deployments for a user with pagination
func (r *DeploymentRepositoryImpl) FindByUserID(ctx context.Context, userID user.UserID, limit, offset int32) ([]*deployment.Deployment, error) {
	queries := database.New(r.db.GetConnection())

	dbDeployments, err := queries.GetDeploymentsByUserID(ctx, &database.GetDeploymentsByUserIDParams{
		UserID: userID.UUID(),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}

	deployments := make([]*deployment.Deployment, len(dbDeployments))
	for i, dbDeployment := range dbDeployments {
		domainDeployment, err := r.toDomain(dbDeployment)
		if err != nil {
			return nil, fmt.Errorf("failed to convert deployment: %w", err)
		}
		deployments[i] = domainDeployment
	}

	return deployments, nil
}

// CountByProjectID counts total deployments for a project
func (r *DeploymentRepositoryImpl) CountByProjectID(ctx context.Context, projectID project.ProjectID) (int64, error) {
	queries := database.New(r.db.GetConnection())

	count, err := queries.CountDeploymentsByProjectID(ctx, projectID.UUID())
	if err != nil {
		return 0, fmt.Errorf("failed to count deployments: %w", err)
	}

	return count, nil
}

// CountByUserID counts total deployments for a user
func (r *DeploymentRepositoryImpl) CountByUserID(ctx context.Context, userID user.UserID) (int64, error) {
	queries := database.New(r.db.GetConnection())

	count, err := queries.CountDeploymentsByUserID(ctx, userID.UUID())
	if err != nil {
		return 0, fmt.Errorf("failed to count deployments: %w", err)
	}

	return count, nil
}

// Delete removes a deployment
func (r *DeploymentRepositoryImpl) Delete(ctx context.Context, id deployment.DeploymentID) error {
	queries := database.New(r.db.GetConnection())

	err := queries.DeleteDeployment(ctx, id.UUID())
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	return nil
}

// FindLatestByProjectID retrieves the most recent deployment for a project
func (r *DeploymentRepositoryImpl) FindLatestByProjectID(ctx context.Context, projectID project.ProjectID) (*deployment.Deployment, error) {
	queries := database.New(r.db.GetConnection())

	dbDeployment, err := queries.GetLatestDeploymentByProjectID(ctx, projectID.UUID())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, deployment.ErrDeploymentNotFound
		}
		return nil, fmt.Errorf("failed to get latest deployment: %w", err)
	}

	return r.toDomain(dbDeployment)
}

// toDomain converts database deployment to domain deployment
func (r *DeploymentRepositoryImpl) toDomain(dbDeployment *database.Deployment) (*deployment.Deployment, error) {
	projectID, err := project.ParseProjectID(dbDeployment.ProjectID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	userID, err := user.ParseUserID(dbDeployment.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var createdAt, updatedAt = dbDeployment.CreatedAt.Time, dbDeployment.UpdatedAt.Time
	var logs string
	if dbDeployment.Logs.Valid {
		logs = dbDeployment.Logs.String
	}

	return deployment.Reconstitute(
		dbDeployment.ID.String(),
		projectID,
		userID,
		dbDeployment.CommitHash,
		dbDeployment.Branch,
		dbDeployment.Status,
		logs,
		createdAt,
		updatedAt,
	)
}

