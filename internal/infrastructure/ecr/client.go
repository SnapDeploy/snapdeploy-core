package ecr

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

// ECRClient wraps AWS ECR operations for image management
type ECRClient struct {
	client         *ecr.Client
	repositoryName string
	registryID     string
}

// NewECRClient creates a new ECR client
func NewECRClient() (*ECRClient, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Get repository name from environment
	repositoryName := os.Getenv("ECR_REPOSITORY_NAME")
	if repositoryName == "" {
		repositoryName = "snapdeploy"
	}

	// Registry ID is optional - if not set, uses default registry
	registryID := os.Getenv("ECR_REGISTRY_ID")

	return &ECRClient{
		client:         ecr.NewFromConfig(cfg),
		repositoryName: repositoryName,
		registryID:     registryID,
	}, nil
}

// DeleteImage deletes a specific image by tag from ECR
func (c *ECRClient) DeleteImage(ctx context.Context, imageTag string) error {
	input := &ecr.BatchDeleteImageInput{
		RepositoryName: aws.String(c.repositoryName),
		ImageIds: []types.ImageIdentifier{
			{
				ImageTag: aws.String(imageTag),
			},
		},
	}

	if c.registryID != "" {
		input.RegistryId = aws.String(c.registryID)
	}

	result, err := c.client.BatchDeleteImage(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete image %s: %w", imageTag, err)
	}

	// Check for failures
	if len(result.Failures) > 0 {
		failure := result.Failures[0]
		// ImageNotFound is not an error - image may already be deleted
		if failure.FailureCode == types.ImageFailureCodeImageNotFound {
			log.Printf("[ECR] Image %s not found (may already be deleted)", imageTag)
			return nil
		}
		return fmt.Errorf("failed to delete image %s: %s - %s",
			imageTag,
			string(failure.FailureCode),
			aws.ToString(failure.FailureReason))
	}

	log.Printf("[ECR] Successfully deleted image: %s", imageTag)
	return nil
}

// DeleteImagesByProjectID deletes all images for a project
// Images are tagged with project ID prefix, e.g., "proj-abc123-commit456"
func (c *ECRClient) DeleteImagesByProjectID(ctx context.Context, projectID string) error {
	// List all images with the project prefix
	images, err := c.listImagesByPrefix(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to list images for project %s: %w", projectID, err)
	}

	if len(images) == 0 {
		log.Printf("[ECR] No images found for project %s", projectID)
		return nil
	}

	// Delete images in batches (AWS limits to 100 per request)
	const batchSize = 100
	for i := 0; i < len(images); i += batchSize {
		end := i + batchSize
		if end > len(images) {
			end = len(images)
		}
		batch := images[i:end]

		input := &ecr.BatchDeleteImageInput{
			RepositoryName: aws.String(c.repositoryName),
			ImageIds:       batch,
		}

		if c.registryID != "" {
			input.RegistryId = aws.String(c.registryID)
		}

		result, err := c.client.BatchDeleteImage(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to delete image batch: %w", err)
		}

		// Log failures but continue
		for _, failure := range result.Failures {
			log.Printf("[ECR] Failed to delete image: %s - %s",
				string(failure.FailureCode),
				aws.ToString(failure.FailureReason))
		}

		log.Printf("[ECR] Deleted %d images for project %s", len(batch)-len(result.Failures), projectID)
	}

	return nil
}

// listImagesByPrefix lists all images with tags starting with the given prefix
func (c *ECRClient) listImagesByPrefix(ctx context.Context, prefix string) ([]types.ImageIdentifier, error) {
	var images []types.ImageIdentifier
	var nextToken *string

	for {
		input := &ecr.ListImagesInput{
			RepositoryName: aws.String(c.repositoryName),
			NextToken:      nextToken,
		}

		if c.registryID != "" {
			input.RegistryId = aws.String(c.registryID)
		}

		result, err := c.client.ListImages(ctx, input)
		if err != nil {
			return nil, err
		}

		for _, imageID := range result.ImageIds {
			if imageID.ImageTag != nil && strings.HasPrefix(*imageID.ImageTag, prefix) {
				images = append(images, imageID)
			}
		}

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return images, nil
}

// DeleteImageByDigest deletes an image by its digest
func (c *ECRClient) DeleteImageByDigest(ctx context.Context, imageDigest string) error {
	input := &ecr.BatchDeleteImageInput{
		RepositoryName: aws.String(c.repositoryName),
		ImageIds: []types.ImageIdentifier{
			{
				ImageDigest: aws.String(imageDigest),
			},
		},
	}

	if c.registryID != "" {
		input.RegistryId = aws.String(c.registryID)
	}

	result, err := c.client.BatchDeleteImage(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete image by digest %s: %w", imageDigest, err)
	}

	if len(result.Failures) > 0 {
		failure := result.Failures[0]
		if failure.FailureCode == types.ImageFailureCodeImageNotFound {
			log.Printf("[ECR] Image with digest %s not found", imageDigest)
			return nil
		}
		return fmt.Errorf("failed to delete image: %s - %s",
			string(failure.FailureCode),
			aws.ToString(failure.FailureReason))
	}

	log.Printf("[ECR] Successfully deleted image with digest: %s", imageDigest)
	return nil
}

// GetImageTag generates a consistent image tag for a deployment
func GetImageTag(projectID, commitHash string) string {
	// Use first 8 chars of project ID and first 7 chars of commit hash
	shortProjectID := projectID
	if len(projectID) > 8 {
		shortProjectID = projectID[:8]
	}
	shortCommit := commitHash
	if len(commitHash) > 7 {
		shortCommit = commitHash[:7]
	}
	return fmt.Sprintf("%s-%s", shortProjectID, shortCommit)
}

