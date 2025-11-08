package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"snapdeploy-core/internal/database"
	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/domain/user"
)

// ProjectRepositoryImpl implements the domain project.ProjectRepository interface
type ProjectRepositoryImpl struct {
	db *database.DB
}

// NewProjectRepository creates a new project repository implementation
func NewProjectRepository(db *database.DB) project.ProjectRepository {
	return &ProjectRepositoryImpl{db: db}
}

// Save persists a project (create or update)
func (r *ProjectRepositoryImpl) Save(ctx context.Context, proj *project.Project) error {
	queries := database.New(r.db.GetConnection())

	// Check if project exists
	_, err := queries.GetProjectByID(ctx, proj.ID().UUID())
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check if project exists: %w", err)
	}

	// If no error, project exists - update it
	if err == nil {
		// Update existing project
		buildCmd := sql.NullString{
			String: proj.BuildCommand().String(),
			Valid:  !proj.BuildCommand().IsEmpty(),
		}
		_, err := queries.UpdateProject(ctx, &database.UpdateProjectParams{
			ID:             proj.ID().UUID(),
			RepositoryUrl:  proj.RepositoryURL().String(),
			InstallCommand: proj.InstallCommand().String(),
			BuildCommand:   buildCmd,
			RunCommand:     proj.RunCommand().String(),
			Language:       proj.Language().String(),
			CustomDomain:   proj.CustomDomain().String(),
		})
		if err != nil {
			return fmt.Errorf("failed to update project: %w", err)
		}
	} else {
		// Project doesn't exist (err == sql.ErrNoRows) - create it
		buildCmd := sql.NullString{
			String: proj.BuildCommand().String(),
			Valid:  !proj.BuildCommand().IsEmpty(),
		}
		_, err := queries.CreateProject(ctx, &database.CreateProjectParams{
			UserID:         proj.UserID().UUID(),
			RepositoryUrl:  proj.RepositoryURL().String(),
			InstallCommand: proj.InstallCommand().String(),
			BuildCommand:   buildCmd,
			RunCommand:     proj.RunCommand().String(),
			Language:       proj.Language().String(),
			CustomDomain:   proj.CustomDomain().String(),
		})
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}
	}

	return nil
}

// FindByID retrieves a project by its ID
func (r *ProjectRepositoryImpl) FindByID(ctx context.Context, id project.ProjectID) (*project.Project, error) {
	queries := database.New(r.db.GetConnection())

	dbProject, err := queries.GetProjectByID(ctx, id.UUID())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, project.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return r.toDomain(dbProject)
}

// FindByUserID retrieves all projects for a user with pagination
func (r *ProjectRepositoryImpl) FindByUserID(ctx context.Context, userID user.UserID, limit, offset int32) ([]*project.Project, error) {
	queries := database.New(r.db.GetConnection())

	dbProjects, err := queries.GetProjectsByUserID(ctx, &database.GetProjectsByUserIDParams{
		UserID: userID.UUID(),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	projects := make([]*project.Project, len(dbProjects))
	for i, dbProject := range dbProjects {
		domainProject, err := r.toDomain(dbProject)
		if err != nil {
			return nil, fmt.Errorf("failed to convert project: %w", err)
		}
		projects[i] = domainProject
	}

	return projects, nil
}

// FindByRepositoryURL retrieves a project by repository URL and user ID
func (r *ProjectRepositoryImpl) FindByRepositoryURL(ctx context.Context, userID user.UserID, repoURL project.RepositoryURL) (*project.Project, error) {
	queries := database.New(r.db.GetConnection())

	dbProject, err := queries.GetProjectByRepositoryURL(ctx, &database.GetProjectByRepositoryURLParams{
		UserID:        userID.UUID(),
		RepositoryUrl: repoURL.String(),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, project.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project by repository URL: %w", err)
	}

	return r.toDomain(dbProject)
}

// CountByUserID counts total projects for a user
func (r *ProjectRepositoryImpl) CountByUserID(ctx context.Context, userID user.UserID) (int64, error) {
	queries := database.New(r.db.GetConnection())

	count, err := queries.CountProjectsByUserID(ctx, userID.UUID())
	if err != nil {
		return 0, fmt.Errorf("failed to count projects: %w", err)
	}

	return count, nil
}

// Delete removes a project
func (r *ProjectRepositoryImpl) Delete(ctx context.Context, id project.ProjectID) error {
	queries := database.New(r.db.GetConnection())

	err := queries.DeleteProject(ctx, id.UUID())
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// ExistsByRepositoryURL checks if a project with the given repository URL exists for a user
func (r *ProjectRepositoryImpl) ExistsByRepositoryURL(ctx context.Context, userID user.UserID, repoURL project.RepositoryURL) (bool, error) {
	queries := database.New(r.db.GetConnection())

	exists, err := queries.ExistsProjectByRepositoryURL(ctx, &database.ExistsProjectByRepositoryURLParams{
		UserID:        userID.UUID(),
		RepositoryUrl: repoURL.String(),
	})
	if err != nil {
		return false, fmt.Errorf("failed to check project existence: %w", err)
	}

	return exists, nil
}

// toDomain converts database project to domain project
func (r *ProjectRepositoryImpl) toDomain(dbProject *database.Project) (*project.Project, error) {
	userID, err := user.ParseUserID(dbProject.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var createdAt, updatedAt = dbProject.CreatedAt.Time, dbProject.UpdatedAt.Time

	// Handle nullable build_command
	buildCommand := ""
	if dbProject.BuildCommand.Valid {
		buildCommand = dbProject.BuildCommand.String
	}

	return project.Reconstitute(
		dbProject.ID.String(),
		userID,
		dbProject.RepositoryUrl,
		dbProject.InstallCommand,
		buildCommand,
		dbProject.RunCommand,
		dbProject.Language,
		dbProject.CustomDomain,
		createdAt,
		updatedAt,
	)
}
