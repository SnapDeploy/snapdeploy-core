-- +goose Up
-- +goose StatementBegin
ALTER TABLE projects
ALTER COLUMN build_command DROP NOT NULL;

-- Set empty string to NULL for existing rows (if any)
UPDATE projects
SET build_command = NULL
WHERE build_command = '';

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
-- Set NULL to empty string before making it NOT NULL
UPDATE projects
SET build_command = ''
WHERE build_command IS NULL;

ALTER TABLE projects
ALTER COLUMN build_command SET NOT NULL;

-- +goose StatementEnd


