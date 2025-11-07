package deployment

import (
	"snapdeploy-core/internal/domain/events"
)

// Event types
const (
	EventTypeDeploymentCreated       = "deployment.created"
	EventTypeDeploymentStatusChanged = "deployment.status_changed"
	EventTypeDeploymentCompleted     = "deployment.completed"
	EventTypeDeploymentFailed        = "deployment.failed"
)

// DeploymentCreated is raised when a new deployment is created
type DeploymentCreated struct {
	events.BaseEvent
	DeploymentID string
	ProjectID    string
	UserID       string
	CommitHash   string
	Branch       string
}

func NewDeploymentCreated(deploymentID, projectID, userID, commitHash, branch string) *DeploymentCreated {
	return &DeploymentCreated{
		BaseEvent:    events.NewBaseEvent(EventTypeDeploymentCreated, deploymentID),
		DeploymentID: deploymentID,
		ProjectID:    projectID,
		UserID:       userID,
		CommitHash:   commitHash,
		Branch:       branch,
	}
}

// DeploymentStatusChanged is raised when a deployment's status changes
type DeploymentStatusChanged struct {
	events.BaseEvent
	DeploymentID string
	OldStatus    string
	NewStatus    string
}

func NewDeploymentStatusChanged(deploymentID, oldStatus, newStatus string) *DeploymentStatusChanged {
	return &DeploymentStatusChanged{
		BaseEvent:    events.NewBaseEvent(EventTypeDeploymentStatusChanged, deploymentID),
		DeploymentID: deploymentID,
		OldStatus:    oldStatus,
		NewStatus:    newStatus,
	}
}

// DeploymentCompleted is raised when a deployment completes successfully
type DeploymentCompleted struct {
	events.BaseEvent
	DeploymentID string
	ProjectID    string
}

func NewDeploymentCompleted(deploymentID, projectID string) *DeploymentCompleted {
	return &DeploymentCompleted{
		BaseEvent:    events.NewBaseEvent(EventTypeDeploymentCompleted, deploymentID),
		DeploymentID: deploymentID,
		ProjectID:    projectID,
	}
}

// DeploymentFailed is raised when a deployment fails
type DeploymentFailed struct {
	events.BaseEvent
	DeploymentID string
	ProjectID    string
	Error        string
}

func NewDeploymentFailed(deploymentID, projectID, errMsg string) *DeploymentFailed {
	return &DeploymentFailed{
		BaseEvent:    events.NewBaseEvent(EventTypeDeploymentFailed, deploymentID),
		DeploymentID: deploymentID,
		ProjectID:    projectID,
		Error:        errMsg,
	}
}

