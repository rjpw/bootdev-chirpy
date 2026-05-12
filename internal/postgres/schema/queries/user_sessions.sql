-- name: CreateSession :one
INSERT INTO user_sessions (id, user_id, created_at, updated_at, expires_at)
VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetSession :one
SELECT *
FROM user_sessions
WHERE id = $1;

-- name: RevokeSession :exec
UPDATE user_sessions
SET updated_at = $2, revoked_at = $2
WHERE id = $1;

-- name: DeleteSessionsByUserID :exec
DELETE FROM user_sessions
WHERE user_id = $1;

