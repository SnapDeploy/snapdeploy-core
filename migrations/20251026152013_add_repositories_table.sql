-- +goose Up
-- +goose StatementBegin
CREATE TABLE repositories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    github_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    description TEXT,
    url VARCHAR(500) UNIQUE NOT NULL,
    html_url VARCHAR(500),
    private BOOLEAN DEFAULT false,
    fork BOOLEAN DEFAULT false,
    stargazers_count INTEGER DEFAULT 0,
    watchers_count INTEGER DEFAULT 0,
    forks_count INTEGER DEFAULT 0,
    default_branch VARCHAR(100),
    language VARCHAR(100),
    created_at TIMESTAMP
    WITH
        TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP
    WITH
        TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_repositories_user_id ON repositories (user_id);

CREATE INDEX idx_repositories_url ON repositories (url);

CREATE UNIQUE INDEX idx_repositories_github_id_user_id ON repositories (github_id, user_id);

-- Create trigger to automatically update updated_at timestamps
CREATE TRIGGER update_repositories_updated_at BEFORE
UPDATE ON repositories FOR EACH ROW EXECUTE FUNCTION update_updated_at_column ();

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_repositories_updated_at ON repositories;

DROP INDEX IF EXISTS idx_repositories_github_id_user_id;

DROP INDEX IF EXISTS idx_repositories_url;

DROP INDEX IF EXISTS idx_repositories_user_id;

DROP TABLE IF EXISTS repositories;

-- +goose StatementEnd