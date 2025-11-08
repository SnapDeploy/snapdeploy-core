-- +goose Up
-- Add custom_domain column to projects table
ALTER TABLE projects ADD COLUMN custom_domain TEXT NOT NULL DEFAULT '';

-- Create unique index on custom_domain to prevent duplicates
CREATE UNIQUE INDEX idx_projects_custom_domain ON projects(custom_domain) WHERE custom_domain != '';

-- Add comment
COMMENT ON COLUMN projects.custom_domain IS 'Custom subdomain prefix for the project (e.g., "my-app" becomes "my-app.snapdeploy.app")';

-- +goose Down
-- Remove index and column
DROP INDEX IF EXISTS idx_projects_custom_domain;
ALTER TABLE projects DROP COLUMN IF EXISTS custom_domain;

