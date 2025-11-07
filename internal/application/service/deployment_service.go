package service

import (
	"context"
	"fmt"
	"time"

	"snapdeploy-core/internal/application/dto"
	"snapdeploy-core/internal/domain/deployment"
	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/domain/user"
)

// DeploymentService handles deployment-related use cases
type DeploymentService struct {
	deploymentRepo deployment.DeploymentRepository
	projectRepo    project.ProjectRepository
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(
	deploymentRepo deployment.DeploymentRepository,
	projectRepo project.ProjectRepository,
) *DeploymentService {
	return &DeploymentService{
		deploymentRepo: deploymentRepo,
		projectRepo:    projectRepo,
	}
}

// CreateDeployment creates a new deployment
func (s *DeploymentService) CreateDeployment(ctx context.Context, userID string, req *dto.CreateDeploymentRequest) (*dto.DeploymentResponse, error) {
	// Parse user ID
	uid, err := user.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Parse project ID
	pid, err := project.ParseProjectID(req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	// Verify project exists and belongs to user
	proj, err := s.projectRepo.FindByID(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	if !proj.BelongsToUser(uid) {
		return nil, deployment.ErrUnauthorized
	}

	// Create deployment entity
	dep, err := deployment.NewDeployment(
		pid,
		uid,
		req.CommitHash,
		req.Branch,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment entity: %w", err)
	}

	// Save deployment
	if err := s.deploymentRepo.Save(ctx, dep); err != nil {
		return nil, fmt.Errorf("failed to save deployment: %w", err)
	}

	return s.toDTO(dep), nil
}

// GetDeploymentByID retrieves a deployment by its ID
func (s *DeploymentService) GetDeploymentByID(ctx context.Context, deploymentID string) (*dto.DeploymentResponse, error) {
	// Parse deployment ID
	did, err := deployment.ParseDeploymentID(deploymentID)
	if err != nil {
		return nil, fmt.Errorf("invalid deployment ID: %w", err)
	}

	// Get deployment
	dep, err := s.deploymentRepo.FindByID(ctx, did)
	if err != nil {
		return nil, err
	}

	return s.toDTO(dep), nil
}

// GetDeploymentsByProjectID retrieves all deployments for a project with pagination
func (s *DeploymentService) GetDeploymentsByProjectID(ctx context.Context, projectID string, page, limit int32) (*dto.DeploymentListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	pid, err := project.ParseProjectID(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	offset := (page - 1) * limit

	deployments, err := s.deploymentRepo.FindByProjectID(ctx, pid, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deployments: %w", err)
	}

	total, err := s.deploymentRepo.CountByProjectID(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to count deployments: %w", err)
	}

	deploymentResponses := make([]*dto.DeploymentResponse, len(deployments))
	for i, dep := range deployments {
		deploymentResponses[i] = s.toDTO(dep)
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return &dto.DeploymentListResponse{
		Deployments: deploymentResponses,
		Pagination: dto.PaginationResponse{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// GetDeploymentsByUserID retrieves all deployments for a user with pagination
func (s *DeploymentService) GetDeploymentsByUserID(ctx context.Context, userID string, page, limit int32) (*dto.DeploymentListResponse, error) {
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

	deployments, err := s.deploymentRepo.FindByUserID(ctx, uid, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deployments: %w", err)
	}

	total, err := s.deploymentRepo.CountByUserID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to count deployments: %w", err)
	}

	deploymentResponses := make([]*dto.DeploymentResponse, len(deployments))
	for i, dep := range deployments {
		deploymentResponses[i] = s.toDTO(dep)
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return &dto.DeploymentListResponse{
		Deployments: deploymentResponses,
		Pagination: dto.PaginationResponse{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateDeploymentStatus updates the status of a deployment
func (s *DeploymentService) UpdateDeploymentStatus(ctx context.Context, deploymentID, userID string, req *dto.UpdateDeploymentStatusRequest) (*dto.DeploymentResponse, error) {
	// Parse IDs
	did, err := deployment.ParseDeploymentID(deploymentID)
	if err != nil {
		return nil, fmt.Errorf("invalid deployment ID: %w", err)
	}

	uid, err := user.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get deployment
	dep, err := s.deploymentRepo.FindByID(ctx, did)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if !dep.BelongsToUser(uid) {
		return nil, deployment.ErrUnauthorized
	}

	// Parse and update status
	status, err := deployment.NewDeploymentStatus(req.Status)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	if err := dep.UpdateStatus(status); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	// Save updated deployment
	if err := s.deploymentRepo.Save(ctx, dep); err != nil {
		return nil, fmt.Errorf("failed to save deployment: %w", err)
	}

	return s.toDTO(dep), nil
}

// AppendDeploymentLog appends a log line to a deployment
func (s *DeploymentService) AppendDeploymentLog(ctx context.Context, deploymentID, userID string, req *dto.AppendDeploymentLogRequest) (*dto.DeploymentResponse, error) {
	// Parse IDs
	did, err := deployment.ParseDeploymentID(deploymentID)
	if err != nil {
		return nil, fmt.Errorf("invalid deployment ID: %w", err)
	}

	uid, err := user.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get deployment
	dep, err := s.deploymentRepo.FindByID(ctx, did)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if !dep.BelongsToUser(uid) {
		return nil, deployment.ErrUnauthorized
	}

	// Append log
	dep.AppendLog(req.LogLine)

	// Save updated deployment
	if err := s.deploymentRepo.Save(ctx, dep); err != nil {
		return nil, fmt.Errorf("failed to save deployment: %w", err)
	}

	return s.toDTO(dep), nil
}

// DeleteDeployment deletes a deployment
func (s *DeploymentService) DeleteDeployment(ctx context.Context, deploymentID, userID string) error {
	// Parse IDs
	did, err := deployment.ParseDeploymentID(deploymentID)
	if err != nil {
		return fmt.Errorf("invalid deployment ID: %w", err)
	}

	uid, err := user.ParseUserID(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Get deployment to check ownership
	dep, err := s.deploymentRepo.FindByID(ctx, did)
	if err != nil {
		return err
	}

	// Check ownership
	if !dep.BelongsToUser(uid) {
		return deployment.ErrUnauthorized
	}

	// Delete deployment
	if err := s.deploymentRepo.Delete(ctx, did); err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	return nil
}

// GetLatestDeploymentByProjectID retrieves the most recent deployment for a project
func (s *DeploymentService) GetLatestDeploymentByProjectID(ctx context.Context, projectID string) (*dto.DeploymentResponse, error) {
	pid, err := project.ParseProjectID(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	dep, err := s.deploymentRepo.FindLatestByProjectID(ctx, pid)
	if err != nil {
		return nil, err
	}

	return s.toDTO(dep), nil
}

// toDTO converts a domain deployment to DTO
func (s *DeploymentService) toDTO(dep *deployment.Deployment) *dto.DeploymentResponse {
	return &dto.DeploymentResponse{
		ID:         dep.ID().String(),
		ProjectID:  dep.ProjectID().String(),
		UserID:     dep.UserID().String(),
		CommitHash: dep.CommitHash().String(),
		Branch:     dep.Branch().String(),
		Status:     dep.Status().String(),
		Logs:       dep.Logs().String(),
		CreatedAt:  dep.CreatedAt().Format(time.RFC3339),
		UpdatedAt:  dep.UpdatedAt().Format(time.RFC3339),
	}
}

