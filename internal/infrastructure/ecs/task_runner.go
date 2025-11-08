package ecs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// TaskRunner handles running one-off ECS tasks (like migrations)
type TaskRunner struct {
	client        *ecs.Client
	cluster       string
	subnets       []string
	securityGroup string
}

// NewTaskRunner creates a new task runner
func NewTaskRunner(client *ecs.Client, cluster string, subnets []string, securityGroup string) *TaskRunner {
	return &TaskRunner{
		client:        client,
		cluster:       cluster,
		subnets:       subnets,
		securityGroup: securityGroup,
	}
}

// RunTaskRequest represents a request to run a one-off task
type RunTaskRequest struct {
	TaskDefinition string
	Command        []string
	EnvVars        map[string]string
	TaskName       string
}

// RunTask runs a one-off ECS task and waits for it to complete
func (r *TaskRunner) RunTask(ctx context.Context, req RunTaskRequest) error {
	log.Printf("[TaskRunner] Running one-off task: %s", req.TaskName)
	log.Printf("[TaskRunner] Task definition: %s", req.TaskDefinition)
	log.Printf("[TaskRunner] Command: %v", req.Command)

	// Build environment variables
	envVars := []types.KeyValuePair{}
	for key, value := range req.EnvVars {
		envVars = append(envVars, types.KeyValuePair{
			Name:  aws.String(key),
			Value: aws.String(value),
		})
	}

	// Run the task
	input := &ecs.RunTaskInput{
		Cluster:        aws.String(r.cluster),
		TaskDefinition: aws.String(req.TaskDefinition),
		LaunchType:     types.LaunchTypeFargate,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				Subnets:        r.subnets,
				SecurityGroups: []string{r.securityGroup},
				AssignPublicIp: types.AssignPublicIpEnabled,
			},
		},
		Overrides: &types.TaskOverride{
			ContainerOverrides: []types.ContainerOverride{
				{
					Name:        aws.String(req.TaskName),
					Command:     req.Command,
					Environment: envVars,
				},
			},
		},
		Tags: []types.Tag{
			{
				Key:   aws.String("Type"),
				Value: aws.String("Migration"),
			},
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("SnapDeploy"),
			},
		},
	}

	result, err := r.client.RunTask(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to run task: %w", err)
	}

	if len(result.Tasks) == 0 {
		return fmt.Errorf("no tasks were started")
	}

	taskArn := *result.Tasks[0].TaskArn
	log.Printf("[TaskRunner] Task started: %s", taskArn)

	// Wait for task to complete
	return r.waitForTaskCompletion(ctx, taskArn)
}

// waitForTaskCompletion waits for a task to complete and checks its exit code
func (r *TaskRunner) waitForTaskCompletion(ctx context.Context, taskArn string) error {
	log.Printf("[TaskRunner] Waiting for task completion...")

	// Poll task status
	maxAttempts := 60 // 5 minutes (5 second intervals)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Describe the task
		describeInput := &ecs.DescribeTasksInput{
			Cluster: aws.String(r.cluster),
			Tasks:   []string{taskArn},
		}

		result, err := r.client.DescribeTasks(ctx, describeInput)
		if err != nil {
			return fmt.Errorf("failed to describe task: %w", err)
		}

		if len(result.Tasks) == 0 {
			return fmt.Errorf("task not found")
		}

		task := result.Tasks[0]
		lastStatus := aws.ToString(task.LastStatus)

		log.Printf("[TaskRunner] Task status: %s (attempt %d/%d)", lastStatus, attempt+1, maxAttempts)

		// Check if task has stopped
		if lastStatus == "STOPPED" {
			// Check exit code
			if len(task.Containers) > 0 {
				container := task.Containers[0]
				exitCode := aws.ToInt32(container.ExitCode)

				if exitCode == 0 {
					log.Printf("[TaskRunner] ✅ Task completed successfully")
					return nil
				} else {
					reason := aws.ToString(container.Reason)
					log.Printf("[TaskRunner] ❌ Task failed with exit code %d: %s", exitCode, reason)
					return fmt.Errorf("task failed with exit code %d: %s", exitCode, reason)
				}
			}

			// No container info, check stop reason
			stopReason := string(task.StopCode)
			return fmt.Errorf("task stopped with reason: %s", stopReason)
		}

		// Task still running, wait and retry
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("task did not complete within timeout")
}

// StopTask stops a running task
func (r *TaskRunner) StopTask(ctx context.Context, taskArn string) error {
	log.Printf("[TaskRunner] Stopping task: %s", taskArn)

	input := &ecs.StopTaskInput{
		Cluster: aws.String(r.cluster),
		Task:    aws.String(taskArn),
		Reason:  aws.String("Stopped by SnapDeploy"),
	}

	_, err := r.client.StopTask(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to stop task: %w", err)
	}

	log.Printf("[TaskRunner] Task stopped successfully")
	return nil
}
