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

-- name: DeleteUsers :exec
TRUNCATE TABLE users CASCADE;

-- name: GetUserPasswordByEmail :one
SELECT * FROM users WHERE email=$1;

-- name: GetUserByRefreshToken :one
SELECT u.*
FROM users u
JOIN refresh_tokens r ON u.id = r.user_id
WHERE r.token = $1;

-- name: ChangeUserPassword :exec
UPDATE users SET hashed_password = $1 WHERE email = $2;

-- name: ChangeUserEmail :exec
UPDATE users SET email = $1 WHERE email = $2;

-- name: UpdateChirpyRed :exec
UPDATE users SET is_chirpy_red = TRUE WHERE id = $1;
