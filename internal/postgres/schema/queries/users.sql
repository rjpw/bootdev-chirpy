-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, password)
VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1; 

-- name: UpdateUser :one
UPDATE users
SET updated_at = $2, email = $3
WHERE email = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: UpgradeUser :one
UPDATE users
SET is_chirpy_red_member = true
WHERE id = $1
RETURNING *;

