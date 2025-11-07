package codebuild

import (
	"context"
	"fmt"
	"time"

	"snapdeploy-core/internal/domain/deployment"
	"snapdeploy-core/internal/domain/project"
)

// SSEBroadcaster interface for broadcasting logs (avoid circular dependency)
type SSEBroadcaster interface {
	BroadcastLog(deploymentID string, logLine string)
}

// CodeBuildService orchestrates builds using AWS CodeBuild
type CodeBuildService struct {
	client      *CodeBuildClient
	deploymentRepo deployment.DeploymentRepository
	sseManager  SSEBroadcaster
}

// NewCodeBuildService creates a new CodeBuild service
func NewCodeBuildService(
	projectName string,
	deploymentRepo deployment.DeploymentRepository,
) (*CodeBuildService, error) {
	client, err := NewCodeBuildClient(projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to create CodeBuild client: %w", err)
	}

	return &CodeBuildService{
		client:        client,
		deploymentRepo: deploymentRepo,
	}, nil
}

// SetSSEManager sets the SSE manager for real-time log streaming
func (s *CodeBuildService) SetSSEManager(manager interface{}) {
	if m, ok := manager.(SSEBroadcaster); ok {
		s.sseManager = m
	}
}

// ServiceBuildRequest contains all information needed to build a deployment
type ServiceBuildRequest struct {
	Deployment     *deployment.Deployment
	Project        *project.Project
	RepositoryURL  string
	Branch         string
	CommitHash     string
	ImageTag       string
	Dockerfile     string
}

// StartBuild starts a CodeBuild build for a deployment
func (s *CodeBuildService) StartBuild(ctx context.Context, req ServiceBuildRequest) (string, error) {
	dep := req.Deployment
	proj := req.Project

	// Update status to BUILDING
	if err := dep.UpdateStatus(deployment.StatusBuilding); err != nil {
		return "", fmt.Errorf("failed to update status: %w", err)
	}
	if err := s.deploymentRepo.Save(ctx, dep); err != nil {
		return "", fmt.Errorf("failed to save deployment: %w", err)
	}

	// Log initial message
	s.logAndUpdate(ctx, dep, "Starting build process with AWS CodeBuild...")

	// Prepare CodeBuild request
	buildReq := BuildRequest{
		RepositoryURL: req.RepositoryURL,
		Branch:        req.Branch,
		CommitHash:    req.CommitHash,
		ImageTag:      req.ImageTag,
		Dockerfile:    req.Dockerfile,
		Language:      proj.Language().String(),
		InstallCmd:    proj.InstallCommand().String(),
		BuildCmd:      proj.BuildCommand().String(),
		RunCmd:        proj.RunCommand().String(),
	}

	// Start the build
	buildID, err := s.client.StartBuild(ctx, buildReq)
	if err != nil {
		s.logAndUpdate(ctx, dep, fmt.Sprintf("Failed to start CodeBuild: %v", err))
		dep.UpdateStatus(deployment.StatusFailed)
		s.deploymentRepo.Save(ctx, dep)
		return "", fmt.Errorf("failed to start CodeBuild: %w", err)
	}

	s.logAndUpdate(ctx, dep, fmt.Sprintf("CodeBuild build started: %s", buildID))
	s.logAndUpdate(ctx, dep, "Build is running in isolated environment...")

	// Start monitoring build status in background
	go s.monitorBuild(ctx, dep, buildID)

	return buildID, nil
}

// monitorBuild monitors the build status and updates deployment accordingly
func (s *CodeBuildService) monitorBuild(ctx context.Context, dep *deployment.Deployment, buildID string) {
	// Wait for build to complete (with 30 minute timeout)
	status, err := s.client.WaitForBuild(ctx, buildID, 30*time.Minute)
	if err != nil {
		s.logAndUpdate(ctx, dep, fmt.Sprintf("Error monitoring build: %v", err))
		dep.UpdateStatus(deployment.StatusFailed)
		s.deploymentRepo.Save(ctx, dep)
		return
	}

	// Update deployment status based on build result
	switch status {
	case "SUCCEEDED":
		s.logAndUpdate(ctx, dep, "Build completed successfully!")
		dep.UpdateStatus(deployment.StatusDeployed)
	case "FAILED", "FAULT", "TIMED_OUT", "STOPPED":
		s.logAndUpdate(ctx, dep, fmt.Sprintf("Build failed with status: %s", status))
		dep.UpdateStatus(deployment.StatusFailed)
	default:
		s.logAndUpdate(ctx, dep, fmt.Sprintf("Build ended with status: %s", status))
		dep.UpdateStatus(deployment.StatusFailed)
	}

	s.deploymentRepo.Save(ctx, dep)
}

// logAndUpdate logs a message and updates the deployment
func (s *CodeBuildService) logAndUpdate(ctx context.Context, dep *deployment.Deployment, message string) {
	// Append to deployment logs
	dep.AppendLog(message)

	// Broadcast to SSE clients (real-time)
	if s.sseManager != nil {
		s.sseManager.BroadcastLog(dep.ID().String(), message)
	}

	// Save to database
	s.deploymentRepo.Save(ctx, dep)
}

