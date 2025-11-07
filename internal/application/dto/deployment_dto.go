package dto

// CreateDeploymentRequest represents the request to create a deployment
type CreateDeploymentRequest struct {
	ProjectID  string `json:"project_id" binding:"required"`
	CommitHash string `json:"commit_hash" binding:"required"`
	Branch     string `json:"branch" binding:"required"`
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
	ID         string `json:"id"`
	ProjectID  string `json:"project_id"`
	UserID     string `json:"user_id"`
	CommitHash string `json:"commit_hash"`
	Branch     string `json:"branch"`
	Status     string `json:"status"`
	Logs       string `json:"logs"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// DeploymentListResponse represents a paginated list of deployments
type DeploymentListResponse struct {
	Deployments []*DeploymentResponse `json:"deployments"`
	Pagination  PaginationResponse    `json:"pagination"`
}

