-- +goose Up
-- +goose StatementBegin
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    repository_url VARCHAR(500) NOT NULL,
    build_command VARCHAR(500) NOT NULL,
    run_command VARCHAR(500) NOT NULL,
    language VARCHAR(50) NOT NULL CHECK (
        language IN ('NODE', 'NODE_TS', 'NEXTJS', 'GO', 'PYTHON')
    ),
    created_at TIMESTAMP
    WITH
        TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP
    WITH
        TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_projects_user_id ON projects (user_id);

CREATE INDEX idx_projects_repository_url ON projects (repository_url);

-- Unique constraint: one project per repository URL per user
CREATE UNIQUE INDEX idx_projects_user_repository ON projects (user_id, repository_url);

-- Create trigger to automatically update updated_at timestamps
CREATE TRIGGER update_projects_updated_at BEFORE
UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column ();

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;

DROP INDEX IF EXISTS idx_projects_user_repository;

DROP INDEX IF EXISTS idx_projects_repository_url;

DROP INDEX IF EXISTS idx_projects_user_id;

DROP TABLE IF EXISTS projects;

-- +goose StatementEnd