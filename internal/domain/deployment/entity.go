package deployment

import (
	"fmt"
	"time"

	"snapdeploy-core/internal/domain/project"
	"snapdeploy-core/internal/domain/user"
)

// Deployment is a domain entity representing a deployment of a project
type Deployment struct {
	id            DeploymentID
	projectID     project.ProjectID
	userID        user.UserID
	commitHash    CommitHash
	branch        Branch
	status        DeploymentStatus
	logs          DeploymentLog
	expiresAt     *time.Time // When the deployment expires (nil = no expiration)
	extendedCount int        // Number of times the TTL has been extended
	createdAt     time.Time
	updatedAt     time.Time
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
	// Set default TTL (expires_at will be set when deployment becomes DEPLOYED)
	return &Deployment{
		id:            NewDeploymentID(),
		projectID:    projectID,
		userID:       userID,
		commitHash:   hash,
		branch:       br,
		status:       StatusPending,
		logs:         NewDeploymentLog(""),
		expiresAt:    nil, // Will be set when status changes to DEPLOYED
		extendedCount: 0,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// Reconstitute recreates a Deployment entity from persistence
func Reconstitute(
	id string,
	projectID project.ProjectID,
	userID user.UserID,
	commitHash, branch, status, logs string,
	expiresAt *time.Time,
	extendedCount int,
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
		id:            deploymentID,
		projectID:    projectID,
		userID:       userID,
		commitHash:   hash,
		branch:       br,
		status:       stat,
		logs:         NewDeploymentLog(logs),
		expiresAt:    expiresAt,
		extendedCount: extendedCount,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}, nil
}

// UpdateStatus updates the deployment status
func (d *Deployment) UpdateStatus(newStatus DeploymentStatus) error {
	if !isValidStatusTransition(d.status, newStatus) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStatusTransition, d.status, newStatus)
	}

	d.status = newStatus
	d.updatedAt = time.Now()

	// Set expiration time when deployment becomes active
	if newStatus == StatusDeployed && d.expiresAt == nil {
		expiresAt := time.Now().Add(time.Duration(DefaultTTLHours) * time.Hour)
		d.expiresAt = &expiresAt
	}

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
		StatusDeployed:   {StatusRolledBack, StatusExpired}, // Can expire or be rolled back
		StatusFailed:     {StatusPending},                   // Allow retry
		StatusRolledBack: {StatusPending},                   // Allow redeployment
		StatusExpired:    {},                                // Terminal state - no transitions allowed
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

// ExtendTTL extends the deployment's time-to-live by the default TTL hours
func (d *Deployment) ExtendTTL() error {
	if d.status != StatusDeployed {
		return ErrCannotExtendNonDeployed
	}

	if d.extendedCount >= MaxExtensions {
		return ErrMaxExtensionsReached
	}

	// Extend from current expiration time (or now if somehow nil)
	baseTime := time.Now()
	if d.expiresAt != nil {
		baseTime = *d.expiresAt
	}

	newExpiry := baseTime.Add(time.Duration(DefaultTTLHours) * time.Hour)
	d.expiresAt = &newExpiry
	d.extendedCount++
	d.updatedAt = time.Now()

	return nil
}

// IsExpired checks if the deployment has passed its expiration time
func (d *Deployment) IsExpired() bool {
	if d.expiresAt == nil {
		return false
	}
	return time.Now().After(*d.expiresAt)
}

// CanExtend returns true if the deployment can be extended
func (d *Deployment) CanExtend() bool {
	return d.status == StatusDeployed && d.extendedCount < MaxExtensions
}

// TimeUntilExpiry returns the duration until the deployment expires
// Returns 0 if already expired or no expiration set
func (d *Deployment) TimeUntilExpiry() time.Duration {
	if d.expiresAt == nil {
		return 0
	}
	remaining := time.Until(*d.expiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
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

func (d *Deployment) ExpiresAt() *time.Time {
	return d.expiresAt
}

func (d *Deployment) ExtendedCount() int {
	return d.extendedCount
}

// String returns string representation (for debugging)
func (d *Deployment) String() string {
	return fmt.Sprintf("Deployment{id: %s, projectID: %s, status: %s}",
		d.id.String(), d.projectID.String(), d.status.String())
}

