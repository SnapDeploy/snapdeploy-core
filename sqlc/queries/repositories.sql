-- name: GetRepositoriesByUserID :many
SELECT * FROM repositories
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: SearchRepositoriesByUserID :many
SELECT * FROM repositories
WHERE user_id = $1
  AND (
    name LIKE '%' || $2 || '%' OR
    full_name LIKE '%' || $2 || '%' OR
    description LIKE '%' || $2 || '%'
  )
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountRepositoriesByUserID :one
SELECT COUNT(*) FROM repositories
WHERE user_id = $1;

-- name: CountSearchRepositoriesByUserID :one
SELECT COUNT(*) FROM repositories
WHERE user_id = $1
  AND (
    name LIKE '%' || $2 || '%' OR
    full_name LIKE '%' || $2 || '%' OR
    description LIKE '%' || $2 || '%'
  );

-- name: GetRepositoryByURL :one
SELECT * FROM repositories
WHERE url = $1;

-- name: UpsertRepository :one
INSERT INTO repositories (
    user_id,
    github_id,
    name,
    full_name,
    description,
    url,
    html_url,
    private,
    fork,
    stargazers_count,
    watchers_count,
    forks_count,
    default_branch,
    language
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
ON CONFLICT (url) 
DO UPDATE SET
    name = EXCLUDED.name,
    full_name = EXCLUDED.full_name,
    description = EXCLUDED.description,
    html_url = EXCLUDED.html_url,
    private = EXCLUDED.private,
    fork = EXCLUDED.fork,
    stargazers_count = EXCLUDED.stargazers_count,
    watchers_count = EXCLUDED.watchers_count,
    forks_count = EXCLUDED.forks_count,
    default_branch = EXCLUDED.default_branch,
    language = EXCLUDED.language,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteRepository :exec
DELETE FROM repositories
WHERE id = $1;
