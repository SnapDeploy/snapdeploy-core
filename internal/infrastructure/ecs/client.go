package ecs

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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

	// Service exists - update it
	return c.updateService(ctx, req.ServiceName, taskDefArn, req.DesiredCount)
}

// createTaskDefinition creates a new task definition revision
func (c *ECSClient) createTaskDefinition(ctx context.Context, req DeploymentRequest) (string, error) {
	region := os.Getenv("AWS_REGION")
	accountID := os.Getenv("AWS_ACCOUNT_ID")
	
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

	// Get execution role ARN
	executionRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s-ecs-task-execution-role", 
		accountID, req.ServiceName)
	taskRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s-ecs-task-role", 
		accountID, req.ServiceName)

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
		Service:        aws.String(serviceName),
		Cluster:        aws.String(c.clusterName),
		TaskDefinition: aws.String(taskDefArn),
		DesiredCount:   aws.Int32(desiredCount),
		ForceNewDeployment: aws.Bool(true),
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

