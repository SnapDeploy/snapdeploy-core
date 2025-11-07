package codebuild

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/aws/aws-sdk-go-v2/service/codebuild/types"
)

// CodeBuildClient wraps AWS CodeBuild operations
type CodeBuildClient struct {
	client    *codebuild.Client
	projectName string
}

// NewCodeBuildClient creates a new CodeBuild client
func NewCodeBuildClient(projectName string) (*CodeBuildClient, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &CodeBuildClient{
		client:      codebuild.NewFromConfig(cfg),
		projectName: projectName,
	}, nil
}

// BuildRequest contains information needed to start a build
type BuildRequest struct {
	RepositoryURL string
	Branch        string
	CommitHash    string
	ImageTag      string
	Dockerfile    string // Dockerfile content
	Language      string
	InstallCmd    string
	BuildCmd      string
	RunCmd        string
}

// StartBuild starts a CodeBuild build and returns the build ID
func (c *CodeBuildClient) StartBuild(ctx context.Context, req BuildRequest) (string, error) {
	// Get environment variables
	region := os.Getenv("AWS_REGION")
	accountID := os.Getenv("AWS_ACCOUNT_ID")
	dockerRegistry := os.Getenv("DOCKER_REGISTRY")

	// Build environment variables
	envVars := []types.EnvironmentVariable{
		{
			Name:  aws.String("REPOSITORY_URL"),
			Value: aws.String(req.RepositoryURL),
		},
		{
			Name:  aws.String("BRANCH"),
			Value: aws.String(req.Branch),
		},
		{
			Name:  aws.String("COMMIT_HASH"),
			Value: aws.String(req.CommitHash),
		},
		{
			Name:  aws.String("IMAGE_TAG"),
			Value: aws.String(req.ImageTag),
		},
		{
			Name:  aws.String("DOCKERFILE_CONTENT"),
			Value: aws.String(req.Dockerfile),
		},
		{
			Name:  aws.String("LANGUAGE"),
			Value: aws.String(req.Language),
		},
		{
			Name:  aws.String("INSTALL_COMMAND"),
			Value: aws.String(req.InstallCmd),
		},
		{
			Name:  aws.String("BUILD_COMMAND"),
			Value: aws.String(req.BuildCmd),
		},
		{
			Name:  aws.String("RUN_COMMAND"),
			Value: aws.String(req.RunCmd),
		},
		{
			Name:  aws.String("AWS_REGION"),
			Value: aws.String(region),
		},
		{
			Name:  aws.String("AWS_ACCOUNT_ID"),
			Value: aws.String(accountID),
		},
		{
			Name:  aws.String("DOCKER_REGISTRY"),
			Value: aws.String(dockerRegistry),
		},
	}

	// Generate inline buildspec
	buildspec := generateBuildspec()

	// Start the build
	input := &codebuild.StartBuildInput{
		ProjectName:              aws.String(c.projectName),
		EnvironmentVariablesOverride: envVars,
		BuildspecOverride:        aws.String(buildspec),
	}

	result, err := c.client.StartBuild(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to start CodeBuild: %w", err)
	}

	return *result.Build.Id, nil
}

// generateBuildspec generates an inline buildspec for CodeBuild
func generateBuildspec() string {
	return `version: 0.2
phases:
  pre_build:
    commands:
      - echo "Cloning repository..."
      - git clone --depth 1 --branch $BRANCH $REPOSITORY_URL /tmp/repo
      - cd /tmp/repo
      - |
        if [ "$COMMIT_HASH" != "HEAD" ] && [ "$COMMIT_HASH" != "" ]; then
          echo "Checking out commit $COMMIT_HASH"
          git fetch origin $COMMIT_HASH
          git checkout $COMMIT_HASH
        fi
      - echo "Writing Dockerfile..."
      - echo "$DOCKERFILE_CONTENT" > Dockerfile.snapdeploy
      - echo "Logging in to ECR..."
      - aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $DOCKER_REGISTRY
  build:
    commands:
      - echo "Building Docker image: $IMAGE_TAG"
      - docker build -f Dockerfile.snapdeploy -t $IMAGE_TAG .
  post_build:
    commands:
      - echo "Pushing image to ECR..."
      - docker push $IMAGE_TAG
      - echo "Build completed successfully!"
`
}

// GetBuildStatus gets the current status of a build
func (c *CodeBuildClient) GetBuildStatus(ctx context.Context, buildID string) (types.StatusType, error) {
	input := &codebuild.BatchGetBuildsInput{
		Ids: []string{buildID},
	}

	result, err := c.client.BatchGetBuilds(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get build status: %w", err)
	}

	if len(result.Builds) == 0 {
		return "", fmt.Errorf("build not found: %s", buildID)
	}

	return result.Builds[0].BuildStatus, nil
}

// WaitForBuild waits for a build to complete and returns the final status
func (c *CodeBuildClient) WaitForBuild(ctx context.Context, buildID string, timeout time.Duration) (types.StatusType, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return "", fmt.Errorf("timeout waiting for build %s", buildID)
			}

			status, err := c.GetBuildStatus(ctx, buildID)
			if err != nil {
				return "", err
			}

			// Check if build is complete
			switch status {
			case types.StatusTypeSucceeded:
				return status, nil
			case types.StatusTypeFailed:
				return status, nil
			case types.StatusTypeFault:
				return status, nil
			case types.StatusTypeTimedOut:
				return status, nil
			case types.StatusTypeStopped:
				return status, nil
			}
			// Otherwise continue waiting
		}
	}
}

