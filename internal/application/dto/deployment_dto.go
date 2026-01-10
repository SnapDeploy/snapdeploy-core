package dto

// CreateDeploymentRequest represents the request to create a deployment
type CreateDeploymentRequest struct {
	ProjectID  string `json:"project_id" binding:"required"`
	CommitHash string `json:"commit_hash" binding:"required"`
	Branch     string `json:"branch" binding:"required"`
	Force      bool   `json:"force"` // If true, replace existing active deployment
}

// UpdateDeploymentStatusRequest represents the request to update deployment status
type UpdateDeploymentStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// AppendDeploymentLogRequest represents the request to append to deployment logs
type AppendDeploymentLogRequest struct {
	LogLine string `json:"log_line" binding:"required"`
}

// DeploymentResponse represents a deployment in API responses
type DeploymentResponse struct {
	ID            string  `json:"id"`
	ProjectID     string  `json:"project_id"`
	UserID        string  `json:"user_id"`
	CommitHash    string  `json:"commit_hash"`
	Branch        string  `json:"branch"`
	Status        string  `json:"status"`
	Logs          string  `json:"logs"`
	ExpiresAt     *string `json:"expires_at,omitempty"`     // ISO 8601 timestamp when deployment expires
	ExtendedCount int     `json:"extended_count"`           // Number of times TTL has been extended
	CanExtend     bool    `json:"can_extend"`               // Whether the deployment can be extended
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// ExtendDeploymentResponse represents the response after extending a deployment's TTL
type ExtendDeploymentResponse struct {
	ID            string `json:"id"`
	ExpiresAt     string `json:"expires_at"`      // New expiration time
	ExtendedCount int    `json:"extended_count"`  // Updated extension count
	CanExtend     bool   `json:"can_extend"`      // Whether more extensions are allowed
}

// DeploymentListResponse represents a paginated list of deployments
type DeploymentListResponse struct {
	Deployments []*DeploymentResponse `json:"deployments"`
	Pagination  PaginationResponse    `json:"pagination"`
}

// ActiveDeploymentExistsResponse is returned when trying to create a deployment
// while an active one already exists for the project
type ActiveDeploymentExistsResponse struct {
	Error              string              `json:"error"`
	Message            string              `json:"message"`
	ExistingDeployment *DeploymentResponse `json:"existing_deployment"`
}
