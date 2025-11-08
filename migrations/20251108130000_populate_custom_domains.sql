-- +goose Up
-- Populate custom domains for existing projects
-- Uses first 8 characters of project ID as the subdomain
UPDATE projects 
SET custom_domain = LOWER(SUBSTRING(id::text, 1, 8))
WHERE custom_domain = '' OR custom_domain IS NULL;

-- +goose Down
-- Set custom domains back to empty (for rollback)
UPDATE projects
SET custom_domain = '';

