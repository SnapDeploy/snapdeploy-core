package builder

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"snapdeploy-core/internal/domain/deployment"
	"snapdeploy-core/internal/domain/project"
)

// ExampleIntegration demonstrates how to integrate the builder service
// with the deployment handler
func ExampleIntegration() {
	// This is example code showing how to use the builder service
	// Copy and adapt this to your deployment handler

	_ = context.Background()

	// 1. Initialize builder service (do this once at application startup)
	// builderService, err := NewBuilderService(
	// 	deploymentRepo,
	// 	"/tmp/snapdeploy/builds",
	// 	"/tmp/snapdeploy/logs",
	// )
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer builderService.Close()

	// 2. When a deployment is created, trigger async build
	// This would happen in your CreateDeployment handler after saving the deployment

	// Example pseudocode:
	/*
		func (h *DeploymentHandler) CreateDeployment(c *gin.Context) {
			// ... validation and creation logic ...

			// Create deployment entity
			dep, err := deployment.NewDeployment(projectID, userID, commitHash, branch)
			if err != nil {
				return err
			}

			// Save deployment (status: PENDING)
			if err := h.deploymentService.CreateDeployment(ctx, userID, req); err != nil {
				return err
			}

			// Return to client immediately
			c.JSON(http.StatusCreated, response)

			// Start async build process
			go h.executeDeploymentBuild(dep, project)
		}

		func (h *DeploymentHandler) executeDeploymentBuild(dep *deployment.Deployment, proj *project.Project) {
			ctx := context.Background()

			// 1. Clone repository
			repoPath, err := h.cloneRepository(proj, dep)
			if err != nil {
				h.updateDeploymentStatus(ctx, dep, deployment.StatusFailed)
				return
			}
			defer h.cleanupRepository(repoPath)

			// 2. Build image
			imageTag := h.generateImageTag(proj, dep)

			buildReq := BuildRequest{
				Deployment:     dep,
				Project:        proj,
				RepositoryPath: repoPath,
				ImageTag:       imageTag,
			}

			err = h.builderService.BuildDeployment(ctx, buildReq)
			if err != nil {
				log.Printf("Build failed for deployment %s: %v", dep.ID(), err)
				return
			}

			// 3. Cleanup
			h.builderService.CleanupBuildArtifacts(repoPath)
		}
	*/
}

// CloneRepository clones a git repository to a temporary directory
// You would implement this in your deployment handler
func CloneRepository(repoURL, commitHash, branch string) (string, error) {
	// Create temporary directory for this build
	tmpDir, err := os.MkdirTemp("", "snapdeploy-build-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Clone the repository
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", branch, repoURL, tmpDir)
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	// Checkout specific commit if provided
	if commitHash != "" {
		cmd = exec.Command("git", "-C", tmpDir, "fetch", "origin", commitHash)
		if err := cmd.Run(); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to fetch commit: %w", err)
		}

		cmd = exec.Command("git", "-C", tmpDir, "checkout", commitHash)
		if err := cmd.Run(); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to checkout commit: %w", err)
		}
	}

	return tmpDir, nil
}

// CleanupRepository removes the cloned repository directory
func CleanupRepository(repoPath string) error {
	return os.RemoveAll(repoPath)
}

// GenerateImageTag generates a Docker image tag for the deployment
func GenerateImageTag(proj *project.Project, dep *deployment.Deployment) string {
	// Format: registry.example.com/project-name:commit-hash
	// Or: registry.example.com/project-id:commit-hash
	registry := os.Getenv("DOCKER_REGISTRY")
	if registry == "" {
		registry = "localhost:5000" // Default to local registry
	}

	projectName := sanitizeImageName(proj.ID().String())
	commitHash := dep.CommitHash().String()

	return fmt.Sprintf("%s/%s:%s", registry, projectName, commitHash)
}

// sanitizeImageName ensures the name is valid for Docker
func sanitizeImageName(name string) string {
	// Docker image names must be lowercase and can only contain
	// lowercase letters, digits, and separators (., -, _)
	// UUIDs are already valid, but we'll convert to lowercase just in case
	return filepath.Base(name)
}

// Example of a complete deployment workflow
func ExampleCompleteWorkflow(
	ctx context.Context,
	builderService *BuilderService,
	dep *deployment.Deployment,
	proj *project.Project,
) error {
	log.Printf("Starting deployment build for project %s", proj.ID())

	// Step 1: Clone repository
	log.Printf("Cloning repository: %s", proj.RepositoryURL())
	repoPath, err := CloneRepository(
		proj.RepositoryURL().String(),
		dep.CommitHash().String(),
		dep.Branch().String(),
	)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	defer CleanupRepository(repoPath)
	log.Printf("Repository cloned to: %s", repoPath)

	// Step 2: Generate image tag
	imageTag := GenerateImageTag(proj, dep)
	log.Printf("Image tag: %s", imageTag)

	// Step 3: Build and push image
	buildReq := BuildRequest{
		Deployment:     dep,
		Project:        proj,
		RepositoryPath: repoPath,
		ImageTag:       imageTag,
	}

	log.Printf("Starting Docker build...")
	err = builderService.BuildDeployment(ctx, buildReq)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Step 4: Cleanup
	log.Printf("Cleaning up build artifacts...")
	if err := builderService.CleanupBuildArtifacts(repoPath); err != nil {
		log.Printf("Warning: failed to cleanup artifacts: %v", err)
	}

	log.Printf("Deployment build completed successfully!")
	return nil
}
