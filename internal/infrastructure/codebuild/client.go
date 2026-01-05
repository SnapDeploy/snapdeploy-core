package codebuild

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/aws/aws-sdk-go-v2/service/codebuild/types"
)

// CodeBuildClient wraps AWS CodeBuild operations
type CodeBuildClient struct {
	client      *codebuild.Client
	logsClient  *cloudwatchlogs.Client
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
		logsClient:  cloudwatchlogs.NewFromConfig(cfg),
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

// BuildResult contains build status and logs
type BuildResult struct {
	Status   types.StatusType
	Logs     []string
	LogGroup string
	LogStream string
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
		ProjectName:                  aws.String(c.projectName),
		EnvironmentVariablesOverride: envVars,
		BuildspecOverride:            aws.String(buildspec),
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
      - git clone --depth 1 --branch "$BRANCH" "$REPOSITORY_URL" /tmp/repo
      - cd /tmp/repo
      - |
        if [ "$COMMIT_HASH" != "HEAD" ] && [ -n "$COMMIT_HASH" ]; then
          echo "Checking out commit $COMMIT_HASH"
          git fetch origin "$COMMIT_HASH"
          git checkout "$COMMIT_HASH"
        fi
      - echo "Writing Dockerfile..."
      - printf "%s" "$DOCKERFILE_CONTENT" > Dockerfile.snapdeploy
      - echo "=== Generated Dockerfile ==="
      - cat Dockerfile.snapdeploy
      - echo "=== End Dockerfile ==="
      - echo "Logging in to ECR..."
      - aws ecr get-login-password --region "$AWS_REGION" | docker login --username AWS --password-stdin "$DOCKER_REGISTRY"
  build:
    commands:
      - echo "Building Docker image - $IMAGE_TAG"
      - docker build -f Dockerfile.snapdeploy -t "$IMAGE_TAG" . 2>&1
  post_build:
    commands:
      - |
        if [ $CODEBUILD_BUILD_SUCCEEDING -eq 1 ]; then
          echo "Pushing image to ECR..."
          docker push "$IMAGE_TAG"
          echo "Build completed successfully!"
        else
          echo "Build failed, skipping push"
        fi
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

// GetBuildDetails gets detailed build information including log location
func (c *CodeBuildClient) GetBuildDetails(ctx context.Context, buildID string) (*types.Build, error) {
	input := &codebuild.BatchGetBuildsInput{
		Ids: []string{buildID},
	}

	result, err := c.client.BatchGetBuilds(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get build details: %w", err)
	}

	if len(result.Builds) == 0 {
		return nil, fmt.Errorf("build not found: %s", buildID)
	}

	return &result.Builds[0], nil
}

// GetBuildLogs fetches the CloudWatch logs for a build
func (c *CodeBuildClient) GetBuildLogs(ctx context.Context, buildID string, tailLines int) ([]string, error) {
	// Get build details to find log group and stream
	build, err := c.GetBuildDetails(ctx, buildID)
	if err != nil {
		return nil, err
	}

	if build.Logs == nil || build.Logs.GroupName == nil || build.Logs.StreamName == nil {
		return nil, fmt.Errorf("build logs not available yet")
	}

	logGroup := *build.Logs.GroupName
	logStream := *build.Logs.StreamName

	// Fetch logs from CloudWatch
	input := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(logStream),
		StartFromHead: aws.Bool(false), // Get latest logs first
	}

	if tailLines > 0 {
		input.Limit = aws.Int32(int32(tailLines))
	}

	result, err := c.logsClient.GetLogEvents(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	var logs []string
	for _, event := range result.Events {
		if event.Message != nil {
			logs = append(logs, *event.Message)
		}
	}

	return logs, nil
}

// GetBuildLogsFiltered fetches logs and filters for errors/important messages
func (c *CodeBuildClient) GetBuildLogsFiltered(ctx context.Context, buildID string) ([]string, error) {
	logs, err := c.GetBuildLogs(ctx, buildID, 200) // Get last 200 lines
	if err != nil {
		return nil, err
	}

	// Filter for important log lines (errors, docker output, etc.)
	var filtered []string
	inErrorSection := false

	for _, line := range logs {
		lineLower := strings.ToLower(line)

		// Always include these patterns
		if strings.Contains(lineLower, "error") ||
			strings.Contains(lineLower, "failed") ||
			strings.Contains(lineLower, "exception") ||
			strings.Contains(lineLower, "cannot") ||
			strings.Contains(lineLower, "not found") ||
			strings.Contains(lineLower, "permission denied") ||
			strings.Contains(lineLower, "step ") ||
			strings.Contains(line, "---") ||
			strings.Contains(line, "===") ||
			strings.Contains(lineLower, "running") ||
			strings.Contains(lineLower, "copying") ||
			strings.Contains(lineLower, "npm err") ||
			strings.Contains(lineLower, "exit code") {
			filtered = append(filtered, line)
			inErrorSection = true
		} else if inErrorSection {
			// Include a few lines after an error
			filtered = append(filtered, line)
			if len(filtered) > 50 {
				inErrorSection = false
			}
		}
	}

	// If no filtered logs, return last 30 lines
	if len(filtered) == 0 && len(logs) > 0 {
		start := len(logs) - 30
		if start < 0 {
			start = 0
		}
		return logs[start:], nil
	}

	return filtered, nil
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
