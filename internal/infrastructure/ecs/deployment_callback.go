package ecs

import (
	"context"

	"snapdeploy-core/internal/domain/deployment"
	"snapdeploy-core/internal/domain/project"
)

// DeploymentCallbackAdapter adapts DeploymentOrchestrator to the callback interface
type DeploymentCallbackAdapter struct {
	orchestrator *DeploymentOrchestrator
}

// NewDeploymentCallbackAdapter creates a new callback adapter
func NewDeploymentCallbackAdapter(orchestrator *DeploymentOrchestrator) *DeploymentCallbackAdapter {
	return &DeploymentCallbackAdapter{
		orchestrator: orchestrator,
	}
}

// OnBuildSuccess is called after a successful build to trigger ECS deployment
func (a *DeploymentCallbackAdapter) OnBuildSuccess(
	ctx context.Context,
	dep *deployment.Deployment,
	proj *project.Project,
	imageURI string,
) error {
	return a.orchestrator.DeployToECS(ctx, dep, proj, imageURI)
}

