-- +goose Up
-- +goose StatementBegin
-- Rename cognito_user_id column to clerk_user_id
ALTER TABLE users
RENAME COLUMN cognito_user_id TO clerk_user_id;

-- Update the index name
DROP INDEX IF EXISTS idx_users_cognito_user_id;

CREATE INDEX idx_users_clerk_user_id ON users (clerk_user_id);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
-- Revert the changes
ALTER TABLE users
RENAME COLUMN clerk_user_id TO cognito_user_id;

-- Revert the index
DROP INDEX IF EXISTS idx_users_clerk_user_id;

CREATE INDEX idx_users_cognito_user_id ON users (cognito_user_id);

-- +goose StatementEnd