-- name: CreateDeployment :one
INSERT INTO deployments (
    id,
    project_id,
    user_id,
    commit_hash,
    branch,
    status,
    logs,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetDeploymentByID :one
SELECT * FROM deployments
WHERE id = $1;

-- name: GetDeploymentsByProjectID :many
SELECT * FROM deployments
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetDeploymentsByUserID :many
SELECT * FROM deployments
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountDeploymentsByProjectID :one
SELECT COUNT(*) FROM deployments
WHERE project_id = $1;

-- name: CountDeploymentsByUserID :one
SELECT COUNT(*) FROM deployments
WHERE user_id = $1;

-- name: UpdateDeployment :exec
UPDATE deployments
SET
    status = $2,
    logs = $3,
    updated_at = $4
WHERE id = $1;

-- name: DeleteDeployment :exec
DELETE FROM deployments
WHERE id = $1;

-- name: GetLatestDeploymentByProjectID :one
SELECT * FROM deployments
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT 1;

