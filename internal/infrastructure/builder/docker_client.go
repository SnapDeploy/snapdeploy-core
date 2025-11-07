package builder

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

// DockerClient wraps Docker operations
type DockerClient struct {
	client *client.Client
}

// NewDockerClient creates a new Docker client wrapper
func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &DockerClient{client: cli}, nil
}

// BuildImageOptions contains options for building a Docker image
type BuildImageOptions struct {
	ContextPath    string
	DockerfilePath string
	Tags           []string
	BuildArgs      map[string]*string
}

// BuildImage builds a Docker image from a context
func (dc *DockerClient) BuildImage(ctx context.Context, opts BuildImageOptions) (io.ReadCloser, error) {
	// Create a tar archive of the build context
	buildContext, err := archive.TarWithOptions(opts.ContextPath, &archive.TarOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create build context: %w", err)
	}

	// Prepare build options
	buildOpts := types.ImageBuildOptions{
		Tags:       opts.Tags,
		Dockerfile: opts.DockerfilePath,
		BuildArgs:  opts.BuildArgs,
		Remove:     true,
		NoCache:    false,
	}

	// Build the image
	response, err := dc.client.ImageBuild(ctx, buildContext, buildOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to build image: %w", err)
	}

	return response.Body, nil
}

// TagImage tags a Docker image
func (dc *DockerClient) TagImage(ctx context.Context, source, target string) error {
	if err := dc.client.ImageTag(ctx, source, target); err != nil {
		return fmt.Errorf("failed to tag image: %w", err)
	}
	return nil
}

// PushImage pushes a Docker image to a registry
func (dc *DockerClient) PushImage(ctx context.Context, imageName string) (io.ReadCloser, error) {
	// Get registry authentication (supports both ECR and standard registries)
	registryAuth, err := GetRegistryAuthWithECR(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry auth: %w", err)
	}

	pushOpts := image.PushOptions{
		RegistryAuth: registryAuth,
	}

	response, err := dc.client.ImagePush(ctx, imageName, pushOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to push image: %w", err)
	}

	return response, nil
}

// GetRegistryAuth returns base64 encoded authentication config for the registry
func GetRegistryAuth() (string, error) {
	username := os.Getenv("DOCKER_REGISTRY_USERNAME")
	password := os.Getenv("DOCKER_REGISTRY_PASSWORD")

	// No credentials configured - use anonymous push (will fail for private registries)
	if username == "" || password == "" {
		return "", nil
	}

	authConfig := registry.AuthConfig{
		Username: username,
		Password: password,
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal auth config: %w", err)
	}

	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}

// RemoveImage removes a Docker image
func (dc *DockerClient) RemoveImage(ctx context.Context, imageID string) error {
	_, err := dc.client.ImageRemove(ctx, imageID, image.RemoveOptions{
		Force:         true,
		PruneChildren: true,
	})
	if err != nil {
		return fmt.Errorf("failed to remove image: %w", err)
	}
	return nil
}

// CreateContainer creates a new container from an image
func (dc *DockerClient) CreateContainer(ctx context.Context, imageName, containerName string, config *container.Config, hostConfig *container.HostConfig) (string, error) {
	resp, err := dc.client.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	return resp.ID, nil
}

// StartContainer starts a container
func (dc *DockerClient) StartContainer(ctx context.Context, containerID string) error {
	if err := dc.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

// StopContainer stops a container
func (dc *DockerClient) StopContainer(ctx context.Context, containerID string) error {
	timeout := 10
	if err := dc.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	return nil
}

// RemoveContainer removes a container
func (dc *DockerClient) RemoveContainer(ctx context.Context, containerID string) error {
	if err := dc.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	return nil
}

// WriteDockerfile writes a Dockerfile to the specified path
func WriteDockerfile(contextPath, dockerfileContent string) (string, error) {
	dockerfilePath := filepath.Join(contextPath, "Dockerfile.snapdeploy")

	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	return "Dockerfile.snapdeploy", nil
}

// Close closes the Docker client
func (dc *DockerClient) Close() error {
	return dc.client.Close()
}
