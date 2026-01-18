package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"snapdeploy-core/internal/application/dto"
	"snapdeploy-core/internal/domain/deployment"
	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/domain/user"
)

// CleanupServiceInterface defines the interface for cleanup operations
type CleanupServiceInterface interface {
	CleanupDeployment(ctx context.Context, dep *deployment.Deployment) error
}

// DeploymentService handles deployment-related use cases
type DeploymentService struct {
	deploymentRepo deployment.DeploymentRepository
	projectRepo    project.ProjectRepository
	cleanupService CleanupServiceInterface
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(
	deploymentRepo deployment.DeploymentRepository,
	projectRepo project.ProjectRepository,
	cleanupService CleanupServiceInterface,
) *DeploymentService {
	return &DeploymentService{
		deploymentRepo: deploymentRepo,
		projectRepo:    projectRepo,
		cleanupService: cleanupService,
	}
}

// ActiveDeploymentError wraps the existing deployment info for 409 responses
type ActiveDeploymentError struct {
	ExistingDeployment *dto.DeploymentResponse
}

func (e *ActiveDeploymentError) Error() string {
	return deployment.ErrActiveDeploymentExists.Error()
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

	// Check for existing active deployment
	existingDep, err := s.deploymentRepo.FindActiveByProjectID(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to check for active deployment: %w", err)
	}

	if existingDep != nil {
		if !req.Force {
			// Return error with existing deployment info for UI to show confirmation
			return nil, &ActiveDeploymentError{
				ExistingDeployment: s.toDTO(existingDep),
			}
		}

		// Force=true: Clean up and delete existing deployment first
		log.Printf("[Deploy] Force replacing existing deployment %s for project %s",
			existingDep.ID().String(), pid.String())

		// Clean up AWS resources if deployment was DEPLOYED
		if existingDep.Status() == deployment.StatusDeployed && s.cleanupService != nil {
			if err := s.cleanupService.CleanupDeployment(ctx, existingDep); err != nil {
				log.Printf("[Deploy] Warning: failed to cleanup existing deployment: %v", err)
				// Continue with deletion even if cleanup fails
			}
		}

		// Delete the existing deployment record
		if err := s.deploymentRepo.Delete(ctx, existingDep.ID()); err != nil {
			return nil, fmt.Errorf("failed to delete existing deployment: %w", err)
		}
		log.Printf("[Deploy] Deleted existing deployment %s", existingDep.ID().String())
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

// DeleteDeployment marks a deployment for deletion and starts async cleanup
// Returns immediately after setting status to DELETING - cleanup happens in background
func (s *DeploymentService) DeleteDeployment(ctx context.Context, deploymentID, userID string) (*dto.DeploymentResponse, error) {
	// Parse IDs
	did, err := deployment.ParseDeploymentID(deploymentID)
	if err != nil {
		return nil, fmt.Errorf("invalid deployment ID: %w", err)
	}

	uid, err := user.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get deployment to check ownership
	dep, err := s.deploymentRepo.FindByID(ctx, did)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if !dep.BelongsToUser(uid) {
		return nil, deployment.ErrUnauthorized
	}

	// Set status to DELETING
	if err := dep.UpdateStatus(deployment.StatusDeleting); err != nil {
		return nil, fmt.Errorf("failed to update status to deleting: %w", err)
	}

	// Save the updated status
	if err := s.deploymentRepo.Save(ctx, dep); err != nil {
		return nil, fmt.Errorf("failed to save deployment status: %w", err)
	}

	log.Printf("[Delete] Marked deployment %s for deletion, starting async cleanup", deploymentID)

	// Start async cleanup in background goroutine
	go s.deleteDeploymentAsync(dep)

	return s.toDTO(dep), nil
}

// deleteDeploymentAsync performs the actual cleanup of AWS resources and deletes the deployment from the database
func (s *DeploymentService) deleteDeploymentAsync(dep *deployment.Deployment) {
	ctx := context.Background()
	deploymentID := dep.ID().String()

	log.Printf("[Delete] Starting async cleanup for deployment %s", deploymentID)

	// Clean up AWS resources if cleanup service is available
	if s.cleanupService != nil {
		if err := s.cleanupService.CleanupDeployment(ctx, dep); err != nil {
			log.Printf("[Delete] Warning: failed to cleanup deployment %s: %v", deploymentID, err)
			// Continue with deletion even if cleanup fails - resources may have been partially cleaned
		} else {
			log.Printf("[Delete] Successfully cleaned up AWS resources for deployment %s", deploymentID)
		}
	}

	// Delete deployment from database
	if err := s.deploymentRepo.Delete(ctx, dep.ID()); err != nil {
		log.Printf("[Delete] Error: failed to delete deployment %s from database: %v", deploymentID, err)
		return
	}

	log.Printf("[Delete] Successfully deleted deployment %s", deploymentID)
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

// ExtendDeploymentTTL extends the TTL of a deployment by the default duration
func (s *DeploymentService) ExtendDeploymentTTL(ctx context.Context, deploymentID, userID string) (*dto.ExtendDeploymentResponse, error) {
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

	// Check if deployment is expired
	if dep.Status() == deployment.StatusExpired {
		return nil, deployment.ErrDeploymentExpired
	}

	// Extend TTL
	if err := dep.ExtendTTL(); err != nil {
		return nil, err
	}

	// Save updated deployment
	if err := s.deploymentRepo.Save(ctx, dep); err != nil {
		return nil, fmt.Errorf("failed to save deployment: %w", err)
	}

	var expiresAt string
	if dep.ExpiresAt() != nil {
		expiresAt = dep.ExpiresAt().Format(time.RFC3339)
	}

	return &dto.ExtendDeploymentResponse{
		ID:            dep.ID().String(),
		ExpiresAt:     expiresAt,
		ExtendedCount: dep.ExtendedCount(),
		CanExtend:     dep.CanExtend(),
	}, nil
}

// toDTO converts a domain deployment to DTO
func (s *DeploymentService) toDTO(dep *deployment.Deployment) *dto.DeploymentResponse {
	var expiresAt *string
	if dep.ExpiresAt() != nil {
		formatted := dep.ExpiresAt().Format(time.RFC3339)
		expiresAt = &formatted
	}

	return &dto.DeploymentResponse{
		ID:            dep.ID().String(),
		ProjectID:    dep.ProjectID().String(),
		UserID:       dep.UserID().String(),
		CommitHash:   dep.CommitHash().String(),
		Branch:       dep.Branch().String(),
		Status:       dep.Status().String(),
		Logs:         dep.Logs().String(),
		DatabaseURL:  dep.DatabaseURL(),
		ExpiresAt:    expiresAt,
		ExtendedCount: dep.ExtendedCount(),
		CanExtend:    dep.CanExtend(),
		CreatedAt:    dep.CreatedAt().Format(time.RFC3339),
		UpdatedAt:    dep.UpdatedAt().Format(time.RFC3339),
	}
}

