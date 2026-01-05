-- name: CreateDeployment :one
INSERT INTO deployments (
    id,
    project_id,
    user_id,
    commit_hash,
    branch,
    status,
    logs,
    expires_at,
    extended_count,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
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
    expires_at = $4,
    extended_count = $5,
    updated_at = $6
WHERE id = $1;

-- name: DeleteDeployment :exec
DELETE FROM deployments
WHERE id = $1;

-- name: GetLatestDeploymentByProjectID :one
SELECT * FROM deployments
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetExpiredDeployments :many
SELECT * FROM deployments
WHERE status = 'DEPLOYED'
  AND expires_at IS NOT NULL
  AND expires_at < NOW()
ORDER BY expires_at ASC
LIMIT $1;

-- name: UpdateDeploymentExpiry :exec
UPDATE deployments
SET
    expires_at = $2,
    extended_count = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: GetActiveDeploymentByProjectID :one
SELECT * FROM deployments
WHERE project_id = $1
  AND status = 'DEPLOYED'
ORDER BY created_at DESC
LIMIT 1;

