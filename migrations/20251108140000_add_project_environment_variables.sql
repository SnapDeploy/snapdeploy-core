-- +goose Up
-- Create project_environment_variables table for storing encrypted environment variables
CREATE TABLE project_environment_variables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value TEXT NOT NULL,  -- Encrypted value
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure unique keys per project
    UNIQUE(project_id, key)
);

-- Create index for faster lookups by project
CREATE INDEX idx_env_vars_project_id ON project_environment_variables(project_id);

-- Add comments
COMMENT ON TABLE project_environment_variables IS 'Stores encrypted environment variables for projects';
COMMENT ON COLUMN project_environment_variables.value IS 'Encrypted environment variable value (AES-256-GCM)';

-- +goose Down
DROP INDEX IF EXISTS idx_env_vars_project_id;
DROP TABLE IF EXISTS project_environment_variables;

