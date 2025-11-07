-- +goose Up
-- +goose StatementBegin
CREATE TABLE deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    commit_hash VARCHAR(40) NOT NULL,
    branch VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (
        status IN (
            'PENDING',
            'BUILDING',
            'DEPLOYING',
            'DEPLOYED',
            'FAILED',
            'ROLLED_BACK'
        )
    ),
    logs TEXT DEFAULT '',
    created_at TIMESTAMP
    WITH
        TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP
    WITH
        TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_deployments_project_id ON deployments (project_id);

CREATE INDEX idx_deployments_user_id ON deployments (user_id);

CREATE INDEX idx_deployments_status ON deployments (status);

CREATE INDEX idx_deployments_created_at ON deployments (created_at DESC);

-- Create composite index for finding latest deployment by project
CREATE INDEX idx_deployments_project_created ON deployments (project_id, created_at DESC);

-- Create trigger to automatically update updated_at timestamps
CREATE TRIGGER update_deployments_updated_at BEFORE
UPDATE ON deployments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column ();

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_deployments_updated_at ON deployments;

DROP INDEX IF EXISTS idx_deployments_project_created;

DROP INDEX IF EXISTS idx_deployments_created_at;

DROP INDEX IF EXISTS idx_deployments_status;

DROP INDEX IF EXISTS idx_deployments_user_id;

DROP INDEX IF EXISTS idx_deployments_project_id;

DROP TABLE IF EXISTS deployments;

-- +goose StatementEnd