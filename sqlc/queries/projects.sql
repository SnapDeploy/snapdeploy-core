-- name: GetProjectByID :one
SELECT * FROM projects
WHERE id = $1;

-- name: GetProjectsByUserID :many
SELECT * FROM projects
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetProjectByRepositoryURL :one
SELECT * FROM projects
WHERE user_id = $1 AND repository_url = $2;

-- name: CountProjectsByUserID :one
SELECT COUNT(*) FROM projects
WHERE user_id = $1;

-- name: CreateProject :one
INSERT INTO projects (
    user_id,
    repository_url,
    install_command,
    build_command,
    run_command,
    language
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateProject :one
UPDATE projects
SET
    repository_url = $2,
    install_command = $3,
    build_command = $4,
    run_command = $5,
    language = $6,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1;

-- name: ExistsProjectByRepositoryURL :one
SELECT EXISTS(
    SELECT 1 FROM projects
    WHERE user_id = $1 AND repository_url = $2
);

