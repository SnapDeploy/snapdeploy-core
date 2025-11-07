package builder

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"snapdeploy-core/internal/domain/deployment"
	"snapdeploy-core/internal/domain/project"
)

// SSEBroadcaster interface for broadcasting logs (avoid circular dependency)
type SSEBroadcaster interface {
	BroadcastLog(deploymentID string, logLine string)
}

// BuilderService orchestrates the deployment build process
type BuilderService struct {
	dockerClient      *DockerClient
	templateGenerator *TemplateGenerator
	logManager        *LogManager
	deploymentRepo    deployment.DeploymentRepository
	workDir           string
	sseManager        SSEBroadcaster
}

// NewBuilderService creates a new builder service
func NewBuilderService(
	deploymentRepo deployment.DeploymentRepository,
	workDir string,
	logDir string,
) (*BuilderService, error) {
	dockerClient, err := NewDockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	templateGenerator, err := NewTemplateGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to create template generator: %w", err)
	}

	logManager := NewLogManager(logDir)
	if err := logManager.EnsureLogDirectory(); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	return &BuilderService{
		dockerClient:      dockerClient,
		templateGenerator: templateGenerator,
		logManager:        logManager,
		deploymentRepo:    deploymentRepo,
		workDir:           workDir,
		sseManager:        nil, // Will be set by handler
	}, nil
}

// SetSSEManager sets the SSE manager for real-time log streaming
func (bs *BuilderService) SetSSEManager(manager interface{}) {
	if m, ok := manager.(SSEBroadcaster); ok {
		bs.sseManager = m
	}
}

// BuildRequest contains all information needed to build a deployment
type BuildRequest struct {
	Deployment     *deployment.Deployment
	Project        *project.Project
	RepositoryPath string // Local path to the cloned repository
	ImageTag       string // e.g., "registry.example.com/project:commit-hash"
}

// BuildDeployment executes the full build pipeline for a deployment
func (bs *BuilderService) BuildDeployment(ctx context.Context, req BuildRequest) error {
	dep := req.Deployment
	proj := req.Project

	// Create log file
	logPath := bs.logManager.GetLogFilePath(dep.ID(), dep.CreatedAt())

	// Helper function to log and update deployment
	logAndUpdate := func(message string) error {
		// Write to file
		if err := bs.logManager.WriteLog(logPath, message); err != nil {
			return err
		}

		// Append to deployment logs
		dep.AppendLog(message)

		// Broadcast to SSE clients (real-time)
		if bs.sseManager != nil {
			bs.sseManager.BroadcastLog(dep.ID().String(), message)
		}

		// Save to database
		if err := bs.deploymentRepo.Save(ctx, dep); err != nil {
			return fmt.Errorf("failed to update deployment: %w", err)
		}

		return nil
	}

	// Update status to BUILDING
	if err := dep.UpdateStatus(deployment.StatusBuilding); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	if err := bs.deploymentRepo.Save(ctx, dep); err != nil {
		return fmt.Errorf("failed to save deployment: %w", err)
	}

	if err := logAndUpdate("Starting build process..."); err != nil {
		return err
	}

	// Generate Dockerfile
	if err := logAndUpdate(fmt.Sprintf("Generating Dockerfile for %s project...", proj.Language())); err != nil {
		return err
	}

	dockerfile, err := bs.templateGenerator.GenerateDockerfile(proj.Language(), TemplateData{
		InstallCommand: proj.InstallCommand().String(),
		BuildCommand:   proj.BuildCommand().String(),
		RunCommand:     proj.RunCommand().String(),
		Port:           "8080",
	})
	if err != nil {
		logAndUpdate(fmt.Sprintf("Failed to generate Dockerfile: %v", err))
		dep.UpdateStatus(deployment.StatusFailed)
		bs.deploymentRepo.Save(ctx, dep)
		return fmt.Errorf("failed to generate dockerfile: %w", err)
	}

	// Write Dockerfile to repository
	dockerfilePath, err := WriteDockerfile(req.RepositoryPath, dockerfile)
	if err != nil {
		logAndUpdate(fmt.Sprintf("Failed to write Dockerfile: %v", err))
		dep.UpdateStatus(deployment.StatusFailed)
		bs.deploymentRepo.Save(ctx, dep)
		return fmt.Errorf("failed to write dockerfile: %w", err)
	}

	if err := logAndUpdate("Dockerfile generated successfully"); err != nil {
		return err
	}

	// Build Docker image
	if err := logAndUpdate(fmt.Sprintf("Building Docker image: %s", req.ImageTag)); err != nil {
		return err
	}

	buildResponse, err := bs.dockerClient.BuildImage(ctx, BuildImageOptions{
		ContextPath:    req.RepositoryPath,
		DockerfilePath: dockerfilePath,
		Tags:           []string{req.ImageTag},
	})
	if err != nil {
		logAndUpdate(fmt.Sprintf("Failed to build image: %v", err))
		dep.UpdateStatus(deployment.StatusFailed)
		bs.deploymentRepo.Save(ctx, dep)
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer buildResponse.Close()

	// Stream build output to logs
	if err := bs.streamDockerOutput(buildResponse, logAndUpdate); err != nil {
		logAndUpdate(fmt.Sprintf("Build failed: %v", err))
		dep.UpdateStatus(deployment.StatusFailed)
		bs.deploymentRepo.Save(ctx, dep)
		return fmt.Errorf("build failed: %w", err)
	}

	if err := logAndUpdate("Docker image built successfully"); err != nil {
		return err
	}

	// Update status to DEPLOYING
	if err := dep.UpdateStatus(deployment.StatusDeploying); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	if err := bs.deploymentRepo.Save(ctx, dep); err != nil {
		return fmt.Errorf("failed to save deployment: %w", err)
	}

	// Push Docker image to registry
	if err := logAndUpdate(fmt.Sprintf("Pushing image to registry: %s", req.ImageTag)); err != nil {
		return err
	}

	// Ensure ECR repository exists if using ECR
	ecrProvider := NewECRAuthProvider()
	if ecrProvider.IsECRRegistry(req.ImageTag) {
		repoName := ecrProvider.ExtractRepositoryName(req.ImageTag)
		if repoName != "" {
			if err := ecrProvider.EnsureECRRepository(ctx, repoName); err != nil {
				logAndUpdate(fmt.Sprintf("Warning: failed to ensure ECR repository exists: %v", err))
				// Continue anyway - might already exist or permissions issue
			}
		}
	}

	pushResponse, err := bs.dockerClient.PushImage(ctx, req.ImageTag)
	if err != nil {
		logAndUpdate(fmt.Sprintf("Failed to push image: %v", err))
		dep.UpdateStatus(deployment.StatusFailed)
		bs.deploymentRepo.Save(ctx, dep)
		return fmt.Errorf("failed to push image: %w", err)
	}
	defer pushResponse.Close()

	// Stream push output to logs
	if err := bs.streamDockerOutput(pushResponse, logAndUpdate); err != nil {
		logAndUpdate(fmt.Sprintf("Push failed: %v", err))
		dep.UpdateStatus(deployment.StatusFailed)
		bs.deploymentRepo.Save(ctx, dep)
		return fmt.Errorf("push failed: %w", err)
	}

	if err := logAndUpdate("Image pushed successfully"); err != nil {
		return err
	}

	// Update status to DEPLOYED
	if err := dep.UpdateStatus(deployment.StatusDeployed); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	if err := bs.deploymentRepo.Save(ctx, dep); err != nil {
		return fmt.Errorf("failed to save deployment: %w", err)
	}

	if err := logAndUpdate("Deployment completed successfully!"); err != nil {
		return err
	}

	return nil
}

// streamDockerOutput streams Docker output and logs each line
func (bs *BuilderService) streamDockerOutput(reader io.ReadCloser, logFunc func(string) error) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse JSON output from Docker
		var logEntry struct {
			Stream string `json:"stream"`
			Error  string `json:"error"`
		}

		if err := json.Unmarshal([]byte(line), &logEntry); err == nil {
			if logEntry.Error != "" {
				logFunc(fmt.Sprintf("ERROR: %s", logEntry.Error))
				return fmt.Errorf("docker error: %s", logEntry.Error)
			}
			if logEntry.Stream != "" {
				logFunc(logEntry.Stream)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading docker output: %w", err)
	}

	return nil
}

// CleanupBuildArtifacts removes temporary build files
func (bs *BuilderService) CleanupBuildArtifacts(repositoryPath string) error {
	dockerfilePath := filepath.Join(repositoryPath, "Dockerfile.snapdeploy")
	if err := os.Remove(dockerfilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove Dockerfile: %w", err)
	}
	return nil
}

// GetDeploymentLogs retrieves the full logs for a deployment
func (bs *BuilderService) GetDeploymentLogs(dep *deployment.Deployment) (string, error) {
	logPath := bs.logManager.GetLogFilePath(dep.ID(), dep.CreatedAt())
	return bs.logManager.ReadLog(logPath)
}

// Close closes the builder service and releases resources
func (bs *BuilderService) Close() error {
	return bs.dockerClient.Close()
}
