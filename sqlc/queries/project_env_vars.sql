-- name: CreateProjectEnvVar :one
INSERT INTO project_environment_variables (
    project_id,
    key,
    value
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetProjectEnvVars :many
SELECT * FROM project_environment_variables
WHERE project_id = $1
ORDER BY key ASC;

-- name: GetProjectEnvVar :one
SELECT * FROM project_environment_variables
WHERE project_id = $1 AND key = $2;

-- name: UpdateProjectEnvVar :one
UPDATE project_environment_variables
SET 
    value = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE project_id = $1 AND key = $2
RETURNING *;

-- name: DeleteProjectEnvVar :exec
DELETE FROM project_environment_variables
WHERE project_id = $1 AND key = $2;

-- name: DeleteAllProjectEnvVars :exec
DELETE FROM project_environment_variables
WHERE project_id = $1;

-- name: CountProjectEnvVars :one
SELECT COUNT(*) FROM project_environment_variables
WHERE project_id = $1;

