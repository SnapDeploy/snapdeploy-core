package ecs

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"snapdeploy-core/internal/domain/deployment"
	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/infrastructure/alb"
	"snapdeploy-core/internal/infrastructure/route53"
)

// DeploymentOrchestrator orchestrates the full deployment process
type DeploymentOrchestrator struct {
	ecsClient       *ECSClient
	albClient       *alb.ALBClient
	route53Client   *route53.Route53Client
	deploymentRepo  deployment.DeploymentRepository
	envVarRepo      project.EnvironmentVariableRepository
	albDNS          string
	baseDomain      string
	subnetIDs       []string
	securityGroupID string
}

// NewDeploymentOrchestrator creates a new deployment orchestrator
func NewDeploymentOrchestrator(
	deploymentRepo deployment.DeploymentRepository,
	envVarRepo project.EnvironmentVariableRepository,
) (*DeploymentOrchestrator, error) {
	ecsClient, err := NewECSClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create ECS client: %w", err)
	}

	albClient, err := alb.NewALBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create ALB client: %w", err)
	}

	route53Client, err := route53.NewRoute53Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create Route53 client: %w", err)
	}

	// Get infrastructure configuration from environment
	albDNS := os.Getenv("ALB_DNS_NAME")
	baseDomain := os.Getenv("BASE_DOMAIN")
	if baseDomain == "" {
		baseDomain = "snap-deploy.com"
	}
	subnetIDs := strings.Split(os.Getenv("SUBNET_IDS"), ",")
	securityGroupID := os.Getenv("SECURITY_GROUP_ID")

	if albDNS == "" || len(subnetIDs) == 0 || securityGroupID == "" {
		return nil, fmt.Errorf("missing required environment variables (ALB_DNS_NAME, SUBNET_IDS, SECURITY_GROUP_ID)")
	}

	return &DeploymentOrchestrator{
		ecsClient:       ecsClient,
		albClient:       albClient,
		route53Client:   route53Client,
		deploymentRepo:  deploymentRepo,
		envVarRepo:      envVarRepo,
		albDNS:          albDNS,
		baseDomain:      baseDomain,
		subnetIDs:       subnetIDs,
		securityGroupID: securityGroupID,
	}, nil
}

// DeployToECS deploys a built image to ECS
func (o *DeploymentOrchestrator) DeployToECS(
	ctx context.Context,
	dep *deployment.Deployment,
	proj *project.Project,
	imageURI string,
) error {
	log.Printf("[ECS] Starting ECS deployment for project %s", proj.ID().String())

	// Update deployment status
	if err := dep.UpdateStatus(deployment.StatusDeploying); err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}
	if err := o.deploymentRepo.Save(ctx, dep); err != nil {
		return fmt.Errorf("failed to save deployment: %w", err)
	}

	dep.AppendLog("🚀 Starting ECS deployment...")
	o.deploymentRepo.Save(ctx, dep)

	// Generate service name based on project ID
	serviceName := generateServiceName(proj.ID().String())

	dep.AppendLog(fmt.Sprintf("📦 Deploying service: %s", serviceName))
	dep.AppendLog(fmt.Sprintf("🖼️  Image: %s", imageURI))
	o.deploymentRepo.Save(ctx, dep)

	// Load and decrypt project environment variables FIRST
	dep.AppendLog("🔐 Loading environment variables...")
	o.deploymentRepo.Save(ctx, dep)

	// Default system env vars
	projectEnvVars := map[string]string{
		"PROJECT_ID": proj.ID().String(),
		"LANGUAGE":   proj.Language().String(),
		"PORT":       "8080", // Default port, can be overridden by user
	}

	// Get decrypted user env vars from repository
	userEnvCount := 0
	if envVarRepoImpl, ok := o.envVarRepo.(interface {
		DecryptAll(ctx context.Context, projectID project.ProjectID) (map[string]string, error)
	}); ok {
		userEnvVars, err := envVarRepoImpl.DecryptAll(ctx, proj.ID())
		if err != nil {
			dep.AppendLog(fmt.Sprintf("⚠️  Warning: Could not load env vars: %v", err))
		} else if len(userEnvVars) > 0 {
			// Merge user env vars (they override defaults including PORT)
			for k, v := range userEnvVars {
				projectEnvVars[k] = v
			}
			userEnvCount = len(userEnvVars)
		}
	}

	if userEnvCount > 0 {
		dep.AppendLog(fmt.Sprintf("✅ Loaded %d custom environment variables", userEnvCount))
	} else {
		dep.AppendLog("ℹ️  No custom environment variables (using defaults)")
	}
	o.deploymentRepo.Save(ctx, dep)

	// Determine container port (from PORT env var if set, otherwise default 8080)
	containerPort := int32(8080)
	if portStr, ok := projectEnvVars["PORT"]; ok {
		if port, err := parsePort(portStr); err == nil {
			containerPort = port
			dep.AppendLog(fmt.Sprintf("🔌 Using custom PORT: %d", containerPort))
			o.deploymentRepo.Save(ctx, dep)
		}
	}

	// Create ALB target group and listener rule with the correct port
	dep.AppendLog("🔧 Creating ALB target group and routing rule...")
	o.deploymentRepo.Save(ctx, dep)

	targetGroupArn, err := o.albClient.CreateTargetGroupAndRule(
		ctx,
		serviceName,
		proj.CustomDomain().String(),
		o.baseDomain,
		containerPort,
	)
	if err != nil {
		dep.AppendLog(fmt.Sprintf("❌ Failed to create ALB routing: %v", err))
		dep.UpdateStatus(deployment.StatusFailed)
		o.deploymentRepo.Save(ctx, dep)
		return fmt.Errorf("failed to create ALB routing: %w", err)
	}

	dep.AppendLog("✅ ALB routing configured")
	o.deploymentRepo.Save(ctx, dep)

	// Prepare deployment request
	deployReq := DeploymentRequest{
		ServiceName:     serviceName,
		ImageURI:        imageURI,
		ProjectID:       proj.ID().String(),
		CustomDomain:    proj.CustomDomain().String(),
		CPU:             "256", // 0.25 vCPU
		Memory:          "512", // 512 MB
		DesiredCount:    1,
		ContainerPort:   containerPort,
		TargetGroupArn:  targetGroupArn,
		SubnetIDs:       o.subnetIDs,
		SecurityGroupID: o.securityGroupID,
		EnvVars:         projectEnvVars,
	}

	// Deploy to ECS
	if err := o.ecsClient.DeployService(ctx, deployReq); err != nil {
		dep.AppendLog(fmt.Sprintf("❌ ECS deployment failed: %v", err))
		dep.UpdateStatus(deployment.StatusFailed)
		o.deploymentRepo.Save(ctx, dep)
		// Clean up ALB resources
		o.albClient.DeleteTargetGroupAndRule(ctx, serviceName)
		return fmt.Errorf("failed to deploy to ECS: %w", err)
	}

	dep.AppendLog("✅ ECS service created/updated successfully")
	o.deploymentRepo.Save(ctx, dep)

	// Wait for service to stabilize
	dep.AppendLog("⏳ Waiting for service to become stable...")
	o.deploymentRepo.Save(ctx, dep)

	if err := o.ecsClient.WaitForServiceStable(ctx, serviceName, 5*time.Minute); err != nil {
		dep.AppendLog(fmt.Sprintf("⚠️  Warning: Service may not be fully stable: %v", err))
		// Don't fail the deployment, just log the warning
	} else {
		dep.AppendLog("✅ Service is running and stable")
	}
	o.deploymentRepo.Save(ctx, dep)

	// Create/Update DNS record
	dep.AppendLog(fmt.Sprintf("🌐 Configuring DNS for %s.%s...", proj.CustomDomain().String(), o.baseDomain))
	o.deploymentRepo.Save(ctx, dep)

	if err := o.route53Client.CreateOrUpdateRecord(ctx, route53.DNSRecordRequest{
		Subdomain: proj.CustomDomain().String(),
		Target:    o.albDNS,
		Type:      "ALIAS",
	}); err != nil {
		dep.AppendLog(fmt.Sprintf("⚠️  Warning: DNS configuration failed: %v", err))
		// Don't fail deployment if DNS fails
	} else {
		deploymentURL := fmt.Sprintf("https://%s.%s", proj.CustomDomain().String(), o.baseDomain)
		dep.AppendLog(fmt.Sprintf("✅ DNS configured successfully"))
		dep.AppendLog(fmt.Sprintf("🌍 Your app is live at: %s", deploymentURL))
	}
	o.deploymentRepo.Save(ctx, dep)

	// Mark deployment as successful
	if err := dep.UpdateStatus(deployment.StatusDeployed); err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}
	if err := o.deploymentRepo.Save(ctx, dep); err != nil {
		return fmt.Errorf("failed to save deployment: %w", err)
	}

	dep.AppendLog("🎉 Deployment completed successfully!")
	o.deploymentRepo.Save(ctx, dep)

	log.Printf("[ECS] Deployment completed successfully for project %s", proj.ID().String())
	return nil
}

// StopDeployment stops a running deployment
func (o *DeploymentOrchestrator) StopDeployment(ctx context.Context, proj *project.Project) error {
	serviceName := generateServiceName(proj.ID().String())
	return o.ecsClient.StopService(ctx, serviceName)
}

// DeleteDeployment removes a deployment completely
func (o *DeploymentOrchestrator) DeleteDeployment(ctx context.Context, proj *project.Project) error {
	serviceName := generateServiceName(proj.ID().String())

	// Delete DNS record
	if err := o.route53Client.DeleteRecord(ctx, proj.CustomDomain().String(), "A"); err != nil {
		log.Printf("[ECS] Warning: failed to delete DNS record: %v", err)
		// Continue with service deletion even if DNS deletion fails
	}

	// Delete ECS service
	if err := o.ecsClient.DeleteService(ctx, serviceName); err != nil {
		return fmt.Errorf("failed to delete ECS service: %w", err)
	}

	// Delete ALB target group and listener rule
	if err := o.albClient.DeleteTargetGroupAndRule(ctx, serviceName); err != nil {
		log.Printf("[ECS] Warning: failed to delete ALB routing: %v", err)
		// Continue even if ALB cleanup fails
	}

	return nil
}

// generateServiceName generates a consistent service name from project ID
func generateServiceName(projectID string) string {
	// Format: snapdeploy-{first-8-chars-of-project-id}
	// Keep it short to avoid hitting AWS naming limits
	shortID := projectID
	if len(projectID) > 8 {
		shortID = projectID[:8]
	}
	return fmt.Sprintf("snapdeploy-%s", shortID)
}

// parsePort parses a port string to int32
func parsePort(portStr string) (int32, error) {
	var port int
	_, err := fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return 0, err
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535")
	}
	return int32(port), nil
}
