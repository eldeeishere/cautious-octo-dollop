-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: CreateMessage :one

INSERT INTO messages (id, body, user_id) 
VALUES (
    gen_random_uuid(),
    $1,
    $2
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: DeleteUser :exec
DELETE FROM users;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, user_id, expires_at)
VALUES (
    $1,
    $2,
    $3
)
RETURNING token;

-- name: GetUserFromRefreshToken :one
SELECT u.*
FROM refresh_tokens rt
JOIN users u ON rt.user_id = u.id
WHERE rt.token = $1 AND rt.revoked_at IS NULL AND rt.expires_at > NOW();

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = NOW(),
    updated_at = NOW()
WHERE token = $1;

-- name: UpdateUser :one
UPDATE users
SET updated_at = NOW(),
    email = $1,
    hashed_password = $2
WHERE id = $3
RETURNING email;

-- name: DeleteChirpsByID :exec
DELETE FROM messages WHERE id = $1 AND user_id = $2;

-- name: GetMessageByID :one
SELECT * FROM messages WHERE id = $1;

-- name: GetMessages :many
SELECT * FROM messages ORDER BY created_at;

-- name: AddUserChirpyRed :exec
UPDATE users
SET is_chirpy_red = TRUE
WHERE id = $1;
