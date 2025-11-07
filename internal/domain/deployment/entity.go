package deployment

import (
	"fmt"
	"time"

	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/domain/user"
)

// Deployment is a domain entity representing a deployment of a project
type Deployment struct {
	id         DeploymentID
	projectID  project.ProjectID
	userID     user.UserID
	commitHash CommitHash
	branch     Branch
	status     DeploymentStatus
	logs       DeploymentLog
	createdAt  time.Time
	updatedAt  time.Time
}

// NewDeployment creates a new Deployment entity
func NewDeployment(
	projectID project.ProjectID,
	userID user.UserID,
	commitHash, branch string,
) (*Deployment, error) {
	hash, err := NewCommitHash(commitHash)
	if err != nil {
		return nil, fmt.Errorf("invalid commit hash: %w", err)
	}

	br, err := NewBranch(branch)
	if err != nil {
		return nil, fmt.Errorf("invalid branch: %w", err)
	}

	now := time.Now()
	return &Deployment{
		id:         NewDeploymentID(),
		projectID:  projectID,
		userID:     userID,
		commitHash: hash,
		branch:     br,
		status:     StatusPending,
		logs:       NewDeploymentLog(""),
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

// Reconstitute recreates a Deployment entity from persistence
func Reconstitute(
	id string,
	projectID project.ProjectID,
	userID user.UserID,
	commitHash, branch, status, logs string,
	createdAt, updatedAt time.Time,
) (*Deployment, error) {
	deploymentID, err := ParseDeploymentID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid deployment ID: %w", err)
	}

	hash, err := NewCommitHash(commitHash)
	if err != nil {
		return nil, fmt.Errorf("invalid commit hash: %w", err)
	}

	br, err := NewBranch(branch)
	if err != nil {
		return nil, fmt.Errorf("invalid branch: %w", err)
	}

	stat, err := NewDeploymentStatus(status)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	return &Deployment{
		id:         deploymentID,
		projectID:  projectID,
		userID:     userID,
		commitHash: hash,
		branch:     br,
		status:     stat,
		logs:       NewDeploymentLog(logs),
		createdAt:  createdAt,
		updatedAt:  updatedAt,
	}, nil
}

// UpdateStatus updates the deployment status
func (d *Deployment) UpdateStatus(newStatus DeploymentStatus) error {
	if !isValidStatusTransition(d.status, newStatus) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStatusTransition, d.status, newStatus)
	}

	d.status = newStatus
	d.updatedAt = time.Now()
	return nil
}

// AppendLog appends a line to the deployment logs
func (d *Deployment) AppendLog(line string) {
	d.logs.AppendLine(line)
	d.updatedAt = time.Now()
}

// SetLogs sets the deployment logs (useful for bulk updates)
func (d *Deployment) SetLogs(logs string) {
	d.logs = NewDeploymentLog(logs)
	d.updatedAt = time.Now()
}

// BelongsToUser checks if the deployment belongs to the specified user
func (d *Deployment) BelongsToUser(userID user.UserID) bool {
	return d.userID.Equals(userID)
}

// BelongsToProject checks if the deployment belongs to the specified project
func (d *Deployment) BelongsToProject(projectID project.ProjectID) bool {
	return d.projectID.Equals(projectID)
}

// isValidStatusTransition checks if a status transition is valid
func isValidStatusTransition(from, to DeploymentStatus) bool {
	// Allow same status (idempotent updates)
	if from == to {
		return true
	}

	transitions := map[DeploymentStatus][]DeploymentStatus{
		StatusPending:    {StatusBuilding, StatusFailed},
		StatusBuilding:   {StatusDeploying, StatusFailed},
		StatusDeploying:  {StatusDeployed, StatusFailed},
		StatusDeployed:   {StatusRolledBack},
		StatusFailed:     {StatusPending}, // Allow retry
		StatusRolledBack: {StatusPending}, // Allow redeployment
	}

	allowedTransitions, exists := transitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == to {
			return true
		}
	}

	return false
}

// Getters

func (d *Deployment) ID() DeploymentID {
	return d.id
}

func (d *Deployment) ProjectID() project.ProjectID {
	return d.projectID
}

func (d *Deployment) UserID() user.UserID {
	return d.userID
}

func (d *Deployment) CommitHash() CommitHash {
	return d.commitHash
}

func (d *Deployment) Branch() Branch {
	return d.branch
}

func (d *Deployment) Status() DeploymentStatus {
	return d.status
}

func (d *Deployment) Logs() DeploymentLog {
	return d.logs
}

func (d *Deployment) CreatedAt() time.Time {
	return d.createdAt
}

func (d *Deployment) UpdatedAt() time.Time {
	return d.updatedAt
}

// String returns string representation (for debugging)
func (d *Deployment) String() string {
	return fmt.Sprintf("Deployment{id: %s, projectID: %s, status: %s}",
		d.id.String(), d.projectID.String(), d.status.String())
}

