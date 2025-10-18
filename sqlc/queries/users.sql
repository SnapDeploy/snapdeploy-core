-- name: CreateUser :one
INSERT INTO users (id, email, username, clerk_user_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByClerkID :one
SELECT * FROM users
WHERE clerk_user_id = $1 LIMIT 1;

-- name: UpdateUser :one
UPDATE users
SET email = $1, username = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $3
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;
