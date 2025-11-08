-- +goose Up
-- Add database support fields to projects table
ALTER TABLE projects ADD COLUMN require_db BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE projects ADD COLUMN migration_command TEXT;

COMMENT ON COLUMN projects.require_db IS 'Whether this project requires a dedicated PostgreSQL database';
COMMENT ON COLUMN projects.migration_command IS 'Optional command to run database migrations (e.g., "npm run migrate")';

-- +goose Down
ALTER TABLE projects DROP COLUMN IF EXISTS migration_command;
ALTER TABLE projects DROP COLUMN IF EXISTS require_db;

