-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetAllChirps :many
SELECT chirps.*
FROM chirps
ORDER BY chirps.created_at ASC;

-- name: GetChirpByID :one
SELECT chirps.*
FROM chirps
WHERE id = $1;

-- name: GetUserChirps :many
SELECT chirps.*
FROM chirps
INNER JOIN users
ON chirps.user_id = users.id
WHERE chirps.user_id = $1
ORDER BY chirps.created_at ASC;
