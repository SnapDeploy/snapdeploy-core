-- +goose Up
-- +goose StatementBegin
-- Add database_url column to store per-deployment database credentials
-- This replaces exposing admin credentials and stores the unique credentials
-- generated for each deployment
ALTER TABLE deployments ADD COLUMN database_url TEXT DEFAULT NULL;

-- Add comment to explain the column
COMMENT ON COLUMN deployments.database_url IS 'Per-deployment database connection URL with unique credentials (only set for projects requiring a database)';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE deployments DROP COLUMN IF EXISTS database_url;
-- +goose StatementEnd
