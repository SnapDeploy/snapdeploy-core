-- +goose Up
-- +goose StatementBegin
ALTER TABLE projects
ADD COLUMN install_command VARCHAR(500) NOT NULL DEFAULT '';

-- Remove the default after adding the column (so new rows will require it)
ALTER TABLE projects
ALTER COLUMN install_command
DROP DEFAULT;

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
ALTER TABLE projects
DROP COLUMN install_command;

-- +goose StatementEnd


