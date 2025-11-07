package deployment

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// DeploymentID is a value object representing a deployment's unique identifier
type DeploymentID struct {
	value uuid.UUID
}

// NewDeploymentID creates a new DeploymentID
func NewDeploymentID() DeploymentID {
	return DeploymentID{value: uuid.New()}
}

// ParseDeploymentID parses a string into a DeploymentID
func ParseDeploymentID(id string) (DeploymentID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return DeploymentID{}, fmt.Errorf("invalid deployment ID format: %w", err)
	}
	return DeploymentID{value: uid}, nil
}

func (id DeploymentID) String() string {
	return id.value.String()
}

func (id DeploymentID) UUID() uuid.UUID {
	return id.value
}

func (id DeploymentID) Equals(other DeploymentID) bool {
	return id.value == other.value
}

// DeploymentStatus represents the status of a deployment
type DeploymentStatus string

const (
	StatusPending    DeploymentStatus = "PENDING"
	StatusBuilding   DeploymentStatus = "BUILDING"
	StatusDeploying  DeploymentStatus = "DEPLOYING"
	StatusDeployed   DeploymentStatus = "DEPLOYED"
	StatusFailed     DeploymentStatus = "FAILED"
	StatusRolledBack DeploymentStatus = "ROLLED_BACK"
)

// NewDeploymentStatus creates a new DeploymentStatus with validation
func NewDeploymentStatus(status string) (DeploymentStatus, error) {
	status = strings.ToUpper(strings.TrimSpace(status))

	switch DeploymentStatus(status) {
	case StatusPending, StatusBuilding, StatusDeploying, StatusDeployed, StatusFailed, StatusRolledBack:
		return DeploymentStatus(status), nil
	default:
		return "", fmt.Errorf("invalid deployment status: %s (must be one of: PENDING, BUILDING, DEPLOYING, DEPLOYED, FAILED, ROLLED_BACK)", status)
	}
}

func (s DeploymentStatus) String() string {
	return string(s)
}

func (s DeploymentStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusBuilding, StatusDeploying, StatusDeployed, StatusFailed, StatusRolledBack:
		return true
	default:
		return false
	}
}

func (s DeploymentStatus) IsTerminal() bool {
	return s == StatusDeployed || s == StatusFailed || s == StatusRolledBack
}

// CommitHash represents a Git commit hash
type CommitHash struct {
	value string
}

// NewCommitHash creates a new CommitHash with validation
func NewCommitHash(hash string) (CommitHash, error) {
	hash = strings.TrimSpace(hash)

	if hash == "" {
		return CommitHash{}, fmt.Errorf("commit hash cannot be empty")
	}

	// Allow special Git references like HEAD, main, etc.
	specialRefs := []string{"HEAD", "head", "main", "master", "develop"}
	hashUpper := strings.ToUpper(hash)
	for _, ref := range specialRefs {
		if hashUpper == strings.ToUpper(ref) {
			return CommitHash{value: hash}, nil
		}
	}

	// Git commit hashes are typically 7-40 characters (short or full SHA-1)
	if len(hash) < 7 || len(hash) > 40 {
		return CommitHash{}, fmt.Errorf("commit hash must be between 7 and 40 characters (or use HEAD, main, master, develop)")
	}

	// Check if it contains only hexadecimal characters
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return CommitHash{}, fmt.Errorf("commit hash must contain only hexadecimal characters")
		}
	}

	return CommitHash{value: strings.ToLower(hash)}, nil
}

func (h CommitHash) String() string {
	return h.value
}

func (h CommitHash) Equals(other CommitHash) bool {
	return h.value == other.value
}

// Branch represents a Git branch name
type Branch struct {
	value string
}

// NewBranch creates a new Branch with validation
func NewBranch(branch string) (Branch, error) {
	branch = strings.TrimSpace(branch)

	if branch == "" {
		return Branch{}, fmt.Errorf("branch name cannot be empty")
	}

	if len(branch) > 255 {
		return Branch{}, fmt.Errorf("branch name too long (max 255 characters)")
	}

	return Branch{value: branch}, nil
}

func (b Branch) String() string {
	return b.value
}

func (b Branch) Equals(other Branch) bool {
	return b.value == other.value
}

// DeploymentLog represents the deployment logs
type DeploymentLog struct {
	value string
}

// NewDeploymentLog creates a new DeploymentLog
func NewDeploymentLog(log string) DeploymentLog {
	return DeploymentLog{value: log}
}

func (l DeploymentLog) String() string {
	return l.value
}

// AppendLine appends a line to the log
func (l *DeploymentLog) AppendLine(line string) {
	if l.value != "" {
		l.value += "\n"
	}
	l.value += line
}
