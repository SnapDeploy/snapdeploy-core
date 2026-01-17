package ecs

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// ECSClient wraps AWS ECS operations
type ECSClient struct {
	client      *ecs.Client
	clusterName string
}

// NewECSClient creates a new ECS client
func NewECSClient() (*ECSClient, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	clusterName := os.Getenv("ECS_CLUSTER_NAME")
	if clusterName == "" {
		clusterName = "snapdeploy-cluster"
	}

	return &ECSClient{
		client:      ecs.NewFromConfig(cfg),
		clusterName: clusterName,
	}, nil
}

// DeploymentRequest contains information needed to deploy a service
type DeploymentRequest struct {
	ServiceName     string
	ImageURI        string
	ProjectID       string
	CustomDomain    string
	CPU             string // e.g., "256"
	Memory          string // e.g., "512"
	DesiredCount    int32
	ContainerPort   int32
	TargetGroupArn  string // ALB target group
	SubnetIDs       []string
	SecurityGroupID string
	EnvVars         map[string]string
}

// DeployService creates or updates an ECS service
func (c *ECSClient) DeployService(ctx context.Context, req DeploymentRequest) error {
	// Check if service exists
	service, err := c.getService(ctx, req.ServiceName)
	if err != nil && !isServiceNotFoundError(err) {
		return fmt.Errorf("failed to check service existence: %w", err)
	}

	// Create or update task definition
	taskDefArn, err := c.createTaskDefinition(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create task definition: %w", err)
	}

	if service == nil {
		// Service doesn't exist - create it
		return c.createService(ctx, req, taskDefArn)
	}

	// Service exists - check if its load balancer configuration is still valid
	// ECS doesn't allow changing load balancer config after creation, so if the
	// target group doesn't match or was deleted, we need to recreate the service
	needsRecreation := c.serviceNeedsRecreation(service, req.TargetGroupArn)

	if needsRecreation {
		log.Printf("[ECS] Service %s has invalid or mismatched load balancer config, recreating...", req.ServiceName)

		// Delete the existing service
		if err := c.DeleteService(ctx, req.ServiceName); err != nil {
			log.Printf("[ECS] Warning: failed to delete old service: %v", err)
			// Continue anyway - the service might be in a bad state
		}

		// Wait a bit for the service to be fully deleted
		time.Sleep(10 * time.Second)

		// Create the service fresh with the new target group
		return c.createService(ctx, req, taskDefArn)
	}

	// Service exists and load balancer config is valid - just update it
	return c.updateService(ctx, req.ServiceName, taskDefArn, req.DesiredCount)
}

// serviceNeedsRecreation checks if an ECS service needs to be recreated
// This happens when the target group has changed or no longer exists
func (c *ECSClient) serviceNeedsRecreation(service *types.Service, newTargetGroupArn string) bool {
	if service == nil {
		return false
	}

	// Check if the service has load balancers configured
	if len(service.LoadBalancers) == 0 {
		// No load balancers configured, but we need one - recreate
		log.Printf("[ECS] Service has no load balancers configured, needs recreation")
		return true
	}

	// Check if the target group ARN matches
	for _, lb := range service.LoadBalancers {
		if lb.TargetGroupArn != nil {
			if *lb.TargetGroupArn == newTargetGroupArn {
				// Target group matches - no need to recreate
				return false
			}
			// Target group doesn't match - the old one may have been deleted
			log.Printf("[ECS] Service target group ARN mismatch: current=%s, new=%s", *lb.TargetGroupArn, newTargetGroupArn)
			return true
		}
	}

	// No target group ARN found in load balancers - recreate
	log.Printf("[ECS] Service has load balancers but no target group ARN found")
	return true
}

// createTaskDefinition creates a new task definition revision
func (c *ECSClient) createTaskDefinition(ctx context.Context, req DeploymentRequest) (string, error) {
	region := os.Getenv("AWS_REGION")

	// Create CloudWatch log group if it doesn't exist
	logGroupName := fmt.Sprintf("/ecs/%s", req.ServiceName)
	if err := c.ensureLogGroupExists(ctx, logGroupName, region); err != nil {
		log.Printf("[ECS] Warning: failed to create log group %s: %v", logGroupName, err)
		// Don't fail the deployment, just log the warning
	}

	// Build environment variables
	envVars := []types.KeyValuePair{}
	for key, value := range req.EnvVars {
		envVars = append(envVars, types.KeyValuePair{
			Name:  aws.String(key),
			Value: aws.String(value),
		})
	}

	// Create container definition
	containerDef := types.ContainerDefinition{
		Name:      aws.String(req.ServiceName),
		Image:     aws.String(req.ImageURI),
		Cpu:       0, // Let Fargate manage
		Memory:    nil,
		Essential: aws.Bool(true),
		PortMappings: []types.PortMapping{
			{
				ContainerPort: aws.Int32(req.ContainerPort),
				HostPort:      aws.Int32(req.ContainerPort),
				Protocol:      types.TransportProtocolTcp,
			},
		},
		Environment: envVars,
		LogConfiguration: &types.LogConfiguration{
			LogDriver: types.LogDriverAwslogs,
			Options: map[string]string{
				"awslogs-group":         fmt.Sprintf("/ecs/%s", req.ServiceName),
				"awslogs-region":        region,
				"awslogs-stream-prefix": "ecs",
			},
		},
	}

	// Get shared user deployment role ARNs from environment
	// These roles are shared across all user deployments for security and simplicity
	taskRoleArn := os.Getenv("USER_DEPLOYMENT_TASK_ROLE_ARN")
	executionRoleArn := os.Getenv("USER_DEPLOYMENT_EXECUTION_ROLE_ARN")

	if taskRoleArn == "" || executionRoleArn == "" {
		return "", fmt.Errorf("USER_DEPLOYMENT_TASK_ROLE_ARN and USER_DEPLOYMENT_EXECUTION_ROLE_ARN environment variables must be set")
	}

	// Register task definition
	input := &ecs.RegisterTaskDefinitionInput{
		Family:                  aws.String(req.ServiceName),
		TaskRoleArn:             aws.String(taskRoleArn),
		ExecutionRoleArn:        aws.String(executionRoleArn),
		NetworkMode:             types.NetworkModeAwsvpc,
		RequiresCompatibilities: []types.Compatibility{types.CompatibilityFargate},
		Cpu:                     aws.String(req.CPU),
		Memory:                  aws.String(req.Memory),
		ContainerDefinitions:    []types.ContainerDefinition{containerDef},
	}

	result, err := c.client.RegisterTaskDefinition(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to register task definition: %w", err)
	}

	return *result.TaskDefinition.TaskDefinitionArn, nil
}

// createService creates a new ECS service
func (c *ECSClient) createService(ctx context.Context, req DeploymentRequest, taskDefArn string) error {
	input := &ecs.CreateServiceInput{
		ServiceName:    aws.String(req.ServiceName),
		Cluster:        aws.String(c.clusterName),
		TaskDefinition: aws.String(taskDefArn),
		DesiredCount:   aws.Int32(req.DesiredCount),
		LaunchType:     types.LaunchTypeFargate,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				Subnets:        req.SubnetIDs,
				SecurityGroups: []string{req.SecurityGroupID},
				AssignPublicIp: types.AssignPublicIpEnabled,
			},
		},
		LoadBalancers: []types.LoadBalancer{
			{
				TargetGroupArn: aws.String(req.TargetGroupArn),
				ContainerName:  aws.String(req.ServiceName),
				ContainerPort:  aws.Int32(req.ContainerPort),
			},
		},
		HealthCheckGracePeriodSeconds: aws.Int32(60),
	}

	_, err := c.client.CreateService(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

// updateService updates an existing ECS service with a new task definition
func (c *ECSClient) updateService(ctx context.Context, serviceName, taskDefArn string, desiredCount int32) error {
	input := &ecs.UpdateServiceInput{
		Service:            aws.String(serviceName),
		Cluster:            aws.String(c.clusterName),
		TaskDefinition:     aws.String(taskDefArn),
		DesiredCount:       aws.Int32(desiredCount),
		ForceNewDeployment: true,
	}

	_, err := c.client.UpdateService(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	return nil
}

// getService retrieves service information
func (c *ECSClient) getService(ctx context.Context, serviceName string) (*types.Service, error) {
	input := &ecs.DescribeServicesInput{
		Cluster:  aws.String(c.clusterName),
		Services: []string{serviceName},
	}

	result, err := c.client.DescribeServices(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(result.Services) == 0 {
		return nil, fmt.Errorf("service not found")
	}

	service := &result.Services[0]

	// Check if service is inactive (deleted)
	if service.Status != nil && *service.Status == "INACTIVE" {
		return nil, fmt.Errorf("service not found")
	}

	return service, nil
}

// WaitForServiceStable waits for the service to reach a stable state
func (c *ECSClient) WaitForServiceStable(ctx context.Context, serviceName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		service, err := c.getService(ctx, serviceName)
		if err != nil {
			return err
		}

		// Check if deployment is stable
		if service.RunningCount == service.DesiredCount && len(service.Deployments) == 1 {
			return nil
		}

		// Wait before checking again
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timeout waiting for service to stabilize")
}

// StopService scales a service down to 0 tasks
func (c *ECSClient) StopService(ctx context.Context, serviceName string) error {
	return c.updateService(ctx, serviceName, "", 0)
}

// DeleteService deletes an ECS service
func (c *ECSClient) DeleteService(ctx context.Context, serviceName string) error {
	// First, scale down to 0
	if err := c.StopService(ctx, serviceName); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Wait a bit for tasks to stop
	time.Sleep(5 * time.Second)

	// Delete the service
	input := &ecs.DeleteServiceInput{
		Service: aws.String(serviceName),
		Cluster: aws.String(c.clusterName),
		Force:   aws.Bool(true),
	}

	_, err := c.client.DeleteService(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}

// isServiceNotFoundError checks if the error indicates a service doesn't exist
func isServiceNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "service not found"
}

// ensureLogGroupExists creates a CloudWatch log group if it doesn't already exist
func (c *ECSClient) ensureLogGroupExists(ctx context.Context, logGroupName, region string) error {
	// Create CloudWatch Logs client
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	logsClient := cloudwatchlogs.NewFromConfig(cfg)

	// Try to create the log group
	_, err = logsClient.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(logGroupName),
	})

	if err != nil {
		// Check if it's because the log group already exists
		if err.Error() != "" && (err.Error() == "ResourceAlreadyExistsException" ||
			err.Error() == "The specified log group already exists") {
			// Log group already exists, this is fine
			log.Printf("[ECS] Log group %s already exists", logGroupName)
			return nil
		}
		return fmt.Errorf("failed to create log group: %w", err)
	}

	log.Printf("[ECS] Created CloudWatch log group: %s", logGroupName)
	return nil
}

// GetServiceLogs fetches CloudWatch logs for a running ECS service
func (c *ECSClient) GetServiceLogs(ctx context.Context, serviceName string, tailLines int) ([]string, error) {
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	logsClient := cloudwatchlogs.NewFromConfig(cfg)

	// CloudWatch log group follows the pattern /ecs/{serviceName}
	logGroupName := fmt.Sprintf("/ecs/%s", serviceName)

	// First, find log streams in this log group (sorted by last event time)
	streamsInput := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String(logGroupName),
		OrderBy:      "LastEventTime",
		Descending:   aws.Bool(true),
		Limit:        aws.Int32(5), // Get the 5 most recent streams
	}

	streamsResult, err := logsClient.DescribeLogStreams(ctx, streamsInput)
	if err != nil {
		return nil, fmt.Errorf("failed to describe log streams: %w", err)
	}

	if len(streamsResult.LogStreams) == 0 {
		return nil, fmt.Errorf("no log streams found for service %s", serviceName)
	}

	// Collect logs from streams
	var allLogs []string

	// Use default limit if not specified
	if tailLines <= 0 {
		tailLines = 200
	}

	// Get logs from the most recent stream(s)
	for _, stream := range streamsResult.LogStreams {
		if stream.LogStreamName == nil {
			continue
		}

		eventsInput := &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  aws.String(logGroupName),
			LogStreamName: stream.LogStreamName,
			StartFromHead: aws.Bool(false), // Get latest logs first
			Limit:         aws.Int32(int32(tailLines)),
		}

		eventsResult, err := logsClient.GetLogEvents(ctx, eventsInput)
		if err != nil {
			log.Printf("[ECS] Warning: failed to get logs from stream %s: %v", *stream.LogStreamName, err)
			continue
		}

		for _, event := range eventsResult.Events {
			if event.Message != nil {
				allLogs = append(allLogs, *event.Message)
			}
		}

		// If we got enough logs from this stream, stop
		if len(allLogs) >= tailLines {
			break
		}
	}

	return allLogs, nil
}

// GenerateServiceName generates a consistent service name from project ID
// Exported so handlers can use it
func GenerateServiceName(projectID string) string {
	// Format: snapdeploy-{first-8-chars-of-project-id}
	// Keep it short to avoid hitting AWS naming limits
	shortID := projectID
	if len(projectID) > 8 {
		shortID = projectID[:8]
	}
	return fmt.Sprintf("snapdeploy-%s", shortID)
}
