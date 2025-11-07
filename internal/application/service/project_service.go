package service

import (
	"context"
	"fmt"
	"time"

	"snapdeploy-core/internal/application/dto"
	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/domain/user"
)

// ProjectService handles project-related use cases
type ProjectService struct {
	projectRepo project.ProjectRepository
}

// NewProjectService creates a new project service
func NewProjectService(projectRepo project.ProjectRepository) *ProjectService {
	return &ProjectService{
		projectRepo: projectRepo,
	}
}

// CreateProject creates a new project
func (s *ProjectService) CreateProject(ctx context.Context, userID string, req *dto.CreateProjectRequest) (*dto.ProjectResponse, error) {
	// Parse user ID
	uid, err := user.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if project with same repository URL already exists
	repoURL, err := project.NewRepositoryURL(req.RepositoryURL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	exists, err := s.projectRepo.ExistsByRepositoryURL(ctx, uid, repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to check project existence: %w", err)
	}

	if exists {
		return nil, project.ErrProjectAlreadyExists
	}

	// Create project entity
	proj, err := project.NewProject(
		uid,
		req.RepositoryURL,
		req.InstallCommand,
		req.BuildCommand,
		req.RunCommand,
		req.Language,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project entity: %w", err)
	}

	// Save project
	if err := s.projectRepo.Save(ctx, proj); err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	return s.toDTO(proj), nil
}

// GetProjectByID retrieves a project by its ID
func (s *ProjectService) GetProjectByID(ctx context.Context, projectID string) (*dto.ProjectResponse, error) {
	// Parse project ID
	pid, err := project.ParseProjectID(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	// Get project
	proj, err := s.projectRepo.FindByID(ctx, pid)
	if err != nil {
		return nil, err
	}

	return s.toDTO(proj), nil
}

// GetProjectsByUserID retrieves all projects for a user with pagination
func (s *ProjectService) GetProjectsByUserID(ctx context.Context, userID string, page, limit int32) (*dto.ProjectListResponse, error) {
	fmt.Printf("[PERF] GetProjectsByUserID called for user: %s\n", userID)
	startTime := time.Now()

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	uid, err := user.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	offset := (page - 1) * limit

	fmt.Printf("[PERF] Fetching projects from DB...\n")
	dbStart := time.Now()
	projects, err := s.projectRepo.FindByUserID(ctx, uid, limit, offset)
	fmt.Printf("[PERF] FindByUserID took: %v\n", time.Since(dbStart))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %w", err)
	}

	fmt.Printf("[PERF] Counting projects...\n")
	countStart := time.Now()
	total, err := s.projectRepo.CountByUserID(ctx, uid)
	fmt.Printf("[PERF] CountByUserID took: %v\n", time.Since(countStart))
	if err != nil {
		return nil, fmt.Errorf("failed to count projects: %w", err)
	}

	projectResponses := make([]*dto.ProjectResponse, len(projects))
	for i, proj := range projects {
		projectResponses[i] = s.toDTO(proj)
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	result := &dto.ProjectListResponse{
		Projects: projectResponses,
		Pagination: dto.PaginationResponse{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	fmt.Printf("[PERF] GetProjectsByUserID total time: %v\n", time.Since(startTime))
	return result, nil
}

// UpdateProject updates an existing project
func (s *ProjectService) UpdateProject(ctx context.Context, projectID, userID string, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
	// Parse IDs
	pid, err := project.ParseProjectID(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	uid, err := user.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get existing project
	proj, err := s.projectRepo.FindByID(ctx, pid)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if !proj.BelongsToUser(uid) {
		return nil, project.ErrUnauthorized
	}

	// Update project
	if err := proj.Update(req.RepositoryURL, req.InstallCommand, req.BuildCommand, req.RunCommand, req.Language); err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	// Save updated project
	if err := s.projectRepo.Save(ctx, proj); err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	return s.toDTO(proj), nil
}

// DeleteProject deletes a project
func (s *ProjectService) DeleteProject(ctx context.Context, projectID, userID string) error {
	// Parse IDs
	pid, err := project.ParseProjectID(projectID)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	uid, err := user.ParseUserID(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Get project to check ownership
	proj, err := s.projectRepo.FindByID(ctx, pid)
	if err != nil {
		return err
	}

	// Check ownership
	if !proj.BelongsToUser(uid) {
		return project.ErrUnauthorized
	}

	// Delete project
	if err := s.projectRepo.Delete(ctx, pid); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// toDTO converts a domain project to DTO
func (s *ProjectService) toDTO(proj *project.Project) *dto.ProjectResponse {
	return &dto.ProjectResponse{
		ID:             proj.ID().String(),
		UserID:         proj.UserID().String(),
		RepositoryURL:  proj.RepositoryURL().String(),
		InstallCommand: proj.InstallCommand().String(),
		BuildCommand:   proj.BuildCommand().String(),
		RunCommand:     proj.RunCommand().String(),
		Language:       proj.Language().String(),
		CreatedAt:      proj.CreatedAt().Format(time.RFC3339),
		UpdatedAt:      proj.UpdatedAt().Format(time.RFC3339),
	}
}
