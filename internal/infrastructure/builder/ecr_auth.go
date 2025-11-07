package builder

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types/registry"
)

// ECRAuthProvider handles AWS ECR authentication
type ECRAuthProvider struct {
	region    string
	accountID string
}

// NewECRAuthProvider creates a new ECR auth provider
func NewECRAuthProvider() *ECRAuthProvider {
	return &ECRAuthProvider{
		region:    os.Getenv("AWS_REGION"),
		accountID: os.Getenv("AWS_ACCOUNT_ID"),
	}
}

// IsECRRegistry checks if the given registry URL is an ECR registry
func (e *ECRAuthProvider) IsECRRegistry(registryURL string) bool {
	return strings.Contains(registryURL, ".ecr.") && strings.Contains(registryURL, ".amazonaws.com")
}

// ExtractRepositoryName extracts the repository name from an ECR image tag
func (e *ECRAuthProvider) ExtractRepositoryName(imageTag string) string {
	// Format: account.dkr.ecr.region.amazonaws.com/repo-name:tag
	parts := strings.Split(imageTag, "/")
	if len(parts) < 2 {
		return ""
	}
	repoAndTag := parts[1]
	repoParts := strings.Split(repoAndTag, ":")
	return repoParts[0]
}

// EnsureECRRepository ensures the ECR repository exists, creating it if necessary
func (e *ECRAuthProvider) EnsureECRRepository(ctx context.Context, repositoryName string) error {
	if e.region == "" {
		return fmt.Errorf("AWS_REGION environment variable not set")
	}

	// Check if repository exists
	cmd := exec.CommandContext(ctx, "aws", "ecr", "describe-repositories",
		"--repository-names", repositoryName,
		"--region", e.region)
	err := cmd.Run()
	if err == nil {
		// Repository exists
		return nil
	}

	// Repository doesn't exist, create it
	log.Printf("[ECR] Repository %s does not exist, creating it...", repositoryName)
	cmd = exec.CommandContext(ctx, "aws", "ecr", "create-repository",
		"--repository-name", repositoryName,
		"--region", e.region,
		"--image-scanning-configuration", "scanOnPush=true",
		"--encryption-configuration", "encryptionType=AES256")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if error is because repository already exists (race condition)
		if strings.Contains(string(output), "RepositoryAlreadyExistsException") {
			log.Printf("[ECR] Repository %s was created by another process", repositoryName)
			return nil
		}
		return fmt.Errorf("failed to create ECR repository: %w\nOutput: %s", err, string(output))
	}

	log.Printf("[ECR] Successfully created repository %s", repositoryName)
	return nil
}

// GetECRAuthToken gets an authentication token from AWS ECR
func (e *ECRAuthProvider) GetECRAuthToken(ctx context.Context) (string, error) {
	if e.region == "" {
		return "", fmt.Errorf("AWS_REGION environment variable not set")
	}

	// Use AWS CLI to get ECR login password
	cmd := exec.CommandContext(ctx, "aws", "ecr", "get-login-password", "--region", e.region)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get ECR password: %w", err)
	}

	password := strings.TrimSpace(string(output))

	// Create auth config
	authConfig := registry.AuthConfig{
		Username: "AWS",
		Password: password,
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ECR auth config: %w", err)
	}

	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}

// GetRegistryAuthWithECR returns authentication config, checking for ECR first
func GetRegistryAuthWithECR(ctx context.Context) (string, error) {
	registryURL := os.Getenv("DOCKER_REGISTRY")

	if registryURL == "" {
		return "", fmt.Errorf("DOCKER_REGISTRY environment variable not set")
	}

	ecrProvider := NewECRAuthProvider()

	// If it's an ECR registry, use ECR authentication
	if ecrProvider.IsECRRegistry(registryURL) {
		return ecrProvider.GetECRAuthToken(ctx)
	}

	// Otherwise, use standard username/password authentication
	return GetRegistryAuth()
}

