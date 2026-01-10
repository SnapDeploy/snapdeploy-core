-- name: GetUserSettingsByUserID :one
SELECT * FROM user_settings
WHERE user_id = $1 LIMIT 1;

-- name: UpsertUserSettings :one
INSERT INTO user_settings (id, user_id, default_branch)
VALUES ($1, $2, $3)
ON CONFLICT (user_id) 
DO UPDATE SET 
    default_branch = EXCLUDED.default_branch,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: CreateUserSettings :one
INSERT INTO user_settings (id, user_id, default_branch)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateUserSettings :one
UPDATE user_settings
SET default_branch = $1, updated_at = CURRENT_TIMESTAMP
WHERE user_id = $2
RETURNING *;

-- name: DeleteUserSettings :exec
DELETE FROM user_settings
WHERE user_id = $1;
