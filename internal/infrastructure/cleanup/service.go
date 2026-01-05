package cleanup

import (
	"context"
	"fmt"
	"log"
	"os"

	"snapdeploy-core/internal/domain/deployment"
	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/infrastructure/alb"
	"snapdeploy-core/internal/infrastructure/ecr"
	"snapdeploy-core/internal/infrastructure/ecs"
	"snapdeploy-core/internal/infrastructure/route53"
)

// CleanupService orchestrates the cleanup of all AWS resources for expired deployments
type CleanupService struct {
	ecsClient     *ecs.ECSClient
	albClient     *alb.ALBClient
	route53Client *route53.Route53Client
	ecrClient     *ecr.ECRClient
	projectRepo   project.ProjectRepository
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(
	ecsClient *ecs.ECSClient,
	albClient *alb.ALBClient,
	route53Client *route53.Route53Client,
	ecrClient *ecr.ECRClient,
	projectRepo project.ProjectRepository,
) *CleanupService {
	return &CleanupService{
		ecsClient:     ecsClient,
		albClient:     albClient,
		route53Client: route53Client,
		ecrClient:     ecrClient,
		projectRepo:   projectRepo,
	}
}

// CleanupDeployment removes all AWS resources associated with an expired deployment
// This includes: ECS service, ALB target group/listener rule, Route53 DNS record, ECR images
func (s *CleanupService) CleanupDeployment(ctx context.Context, dep *deployment.Deployment) error {
	log.Printf("[Cleanup] Starting cleanup for deployment %s (project: %s)",
		dep.ID().String(), dep.ProjectID().String())

	// Get project details for domain and service naming
	proj, err := s.projectRepo.FindByID(ctx, dep.ProjectID())
	if err != nil {
		log.Printf("[Cleanup] Warning: could not find project %s: %v", dep.ProjectID().String(), err)
		// Continue with cleanup using fallback values
	}

	serviceName := ecs.GenerateServiceName(dep.ProjectID().String())
	var customDomain string
	if proj != nil {
		customDomain = proj.CustomDomain().String()
	}

	var cleanupErrors []error

	// 1. Delete ECS Service (this stops running tasks)
	if s.ecsClient != nil {
		log.Printf("[Cleanup] Deleting ECS service: %s", serviceName)
		if err := s.ecsClient.DeleteService(ctx, serviceName); err != nil {
			log.Printf("[Cleanup] Warning: failed to delete ECS service: %v", err)
			cleanupErrors = append(cleanupErrors, fmt.Errorf("ECS: %w", err))
		} else {
			log.Printf("[Cleanup] Successfully deleted ECS service: %s", serviceName)
		}
	}

	// 2. Delete ALB Target Group and Listener Rule
	if s.albClient != nil {
		log.Printf("[Cleanup] Deleting ALB resources for: %s", serviceName)
		if err := s.albClient.DeleteTargetGroupAndRule(ctx, serviceName); err != nil {
			log.Printf("[Cleanup] Warning: failed to delete ALB resources: %v", err)
			cleanupErrors = append(cleanupErrors, fmt.Errorf("ALB: %w", err))
		} else {
			log.Printf("[Cleanup] Successfully deleted ALB resources: %s", serviceName)
		}
	}

	// 3. Delete Route53 DNS Record
	if s.route53Client != nil && customDomain != "" {
		log.Printf("[Cleanup] Deleting DNS record for: %s", customDomain)
		if err := s.route53Client.DeleteRecord(ctx, customDomain, "A"); err != nil {
			log.Printf("[Cleanup] Warning: failed to delete DNS record: %v", err)
			cleanupErrors = append(cleanupErrors, fmt.Errorf("Route53: %w", err))
		} else {
			log.Printf("[Cleanup] Successfully deleted DNS record: %s", customDomain)
		}
	}

	// 4. Delete ECR Images for this deployment
	if s.ecrClient != nil {
		log.Printf("[Cleanup] Deleting ECR images for project: %s", dep.ProjectID().String())
		imageTag := ecr.GetImageTag(dep.ProjectID().String(), dep.CommitHash().String())
		if err := s.ecrClient.DeleteImage(ctx, imageTag); err != nil {
			log.Printf("[Cleanup] Warning: failed to delete ECR image: %v", err)
			cleanupErrors = append(cleanupErrors, fmt.Errorf("ECR: %w", err))
		} else {
			log.Printf("[Cleanup] Successfully deleted ECR image: %s", imageTag)
		}
	}

	// 5. Optionally delete CloudWatch Log Group
	// Note: We're keeping logs for debugging purposes
	// If you want to delete them, add CloudWatch Logs client here

	// Log summary
	if len(cleanupErrors) > 0 {
		log.Printf("[Cleanup] Completed with %d errors for deployment %s", len(cleanupErrors), dep.ID().String())
		// Return first error for status update, but we've logged all of them
		return cleanupErrors[0]
	}

	log.Printf("[Cleanup] Successfully cleaned up all resources for deployment %s", dep.ID().String())
	return nil
}

// CleanupProjectResources removes all AWS resources for a project (used when deleting a project)
func (s *CleanupService) CleanupProjectResources(ctx context.Context, projectID string) error {
	log.Printf("[Cleanup] Starting full cleanup for project: %s", projectID)

	serviceName := ecs.GenerateServiceName(projectID)

	// Delete ECS Service
	if s.ecsClient != nil {
		if err := s.ecsClient.DeleteService(ctx, serviceName); err != nil {
			log.Printf("[Cleanup] Warning: failed to delete ECS service: %v", err)
		}
	}

	// Delete ALB resources
	if s.albClient != nil {
		if err := s.albClient.DeleteTargetGroupAndRule(ctx, serviceName); err != nil {
			log.Printf("[Cleanup] Warning: failed to delete ALB resources: %v", err)
		}
	}

	// Delete all ECR images for project
	if s.ecrClient != nil {
		if err := s.ecrClient.DeleteImagesByProjectID(ctx, projectID); err != nil {
			log.Printf("[Cleanup] Warning: failed to delete ECR images: %v", err)
		}
	}

	log.Printf("[Cleanup] Completed cleanup for project: %s", projectID)
	return nil
}

// NewCleanupServiceFromEnv creates a cleanup service with clients initialized from environment
func NewCleanupServiceFromEnv(projectRepo project.ProjectRepository) (*CleanupService, error) {
	var ecsClient *ecs.ECSClient
	var albClient *alb.ALBClient
	var route53Client *route53.Route53Client
	var ecrClient *ecr.ECRClient

	// Initialize ECS client
	if os.Getenv("ECS_CLUSTER_NAME") != "" {
		client, err := ecs.NewECSClient()
		if err != nil {
			log.Printf("[Cleanup] Warning: failed to initialize ECS client: %v", err)
		} else {
			ecsClient = client
		}
	}

	// Initialize ALB client
	if os.Getenv("ALB_LISTENER_ARN") != "" {
		client, err := alb.NewALBClient()
		if err != nil {
			log.Printf("[Cleanup] Warning: failed to initialize ALB client: %v", err)
		} else {
			albClient = client
		}
	}

	// Initialize Route53 client
	if os.Getenv("ROUTE53_HOSTED_ZONE_ID") != "" {
		client, err := route53.NewRoute53Client()
		if err != nil {
			log.Printf("[Cleanup] Warning: failed to initialize Route53 client: %v", err)
		} else {
			route53Client = client
		}
	}

	// Initialize ECR client
	client, err := ecr.NewECRClient()
	if err != nil {
		log.Printf("[Cleanup] Warning: failed to initialize ECR client: %v", err)
	} else {
		ecrClient = client
	}

	return &CleanupService{
		ecsClient:     ecsClient,
		albClient:     albClient,
		route53Client: route53Client,
		ecrClient:     ecrClient,
		projectRepo:   projectRepo,
	}, nil
}

