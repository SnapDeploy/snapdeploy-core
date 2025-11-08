package dto

// CreateEnvVarRequest represents the request to create/update an environment variable
type CreateEnvVarRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// UpdateEnvVarRequest represents the request to update an environment variable
type UpdateEnvVarRequest struct {
	Value string `json:"value" binding:"required"`
}

// EnvVarResponse represents an environment variable in API responses
// NOTE: Value is ALWAYS masked for security - never exposed to frontend
type EnvVarResponse struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Key       string `json:"key"`
	Value     string `json:"value"` // Masked: "f*******t"
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// EnvVarListResponse represents a list of environment variables
type EnvVarListResponse struct {
	EnvironmentVariables []*EnvVarResponse `json:"environment_variables"`
	Count                int64             `json:"count"`
}

