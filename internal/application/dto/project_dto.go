package dto

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	RepositoryURL  string `json:"repository_url" binding:"required"`
	InstallCommand string `json:"install_command" binding:"required"`
	BuildCommand   string `json:"build_command"` // Optional
	RunCommand     string `json:"run_command" binding:"required"`
	Language       string `json:"language" binding:"required"`
	CustomDomain   string `json:"custom_domain"` // Optional - will auto-generate if empty
}

// UpdateProjectRequest represents the request to update a project
type UpdateProjectRequest struct {
	RepositoryURL  string `json:"repository_url" binding:"required"`
	InstallCommand string `json:"install_command" binding:"required"`
	BuildCommand   string `json:"build_command"` // Optional
	RunCommand     string `json:"run_command" binding:"required"`
	Language       string `json:"language" binding:"required"`
	CustomDomain   string `json:"custom_domain"` // Optional - will auto-generate if empty
}

// ProjectResponse represents a project in API responses
type ProjectResponse struct {
	ID             string `json:"id"`
	UserID         string `json:"user_id"`
	RepositoryURL  string `json:"repository_url"`
	InstallCommand string `json:"install_command"`
	BuildCommand   string `json:"build_command"`
	RunCommand     string `json:"run_command"`
	Language       string `json:"language"`
	CustomDomain   string `json:"custom_domain"`
	DeploymentURL  string `json:"deployment_url"` // Full URL like https://my-app.snapdeploy.app
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// ProjectListResponse represents a paginated list of projects
type ProjectListResponse struct {
	Projects   []*ProjectResponse `json:"projects"`
	Pagination PaginationResponse `json:"pagination"`
}
