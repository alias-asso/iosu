-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, activated, admin, created_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: CreateUserIfMissing :execrows
INSERT OR IGNORE INTO users (username, email, password_hash, activated, admin, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ?;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: SetUserPassword :exec
UPDATE users SET password_hash = ? WHERE id = ?;

-- name: ActivateUser :exec
UPDATE users SET password_hash = ?, activated = TRUE WHERE id = ?;

-- name: SetUserAdmin :exec
UPDATE users SET admin = ? WHERE id = ?;

-- name: CreateActivationCode :exec
INSERT INTO activation_codes (code, user_id, expires_at) VALUES (?, ?, ?);

-- name: GetActivationCode :one
SELECT sqlc.embed(activation_codes), sqlc.embed(users)
FROM activation_codes
JOIN users ON users.id = activation_codes.user_id
WHERE activation_codes.code = ?;

-- name: UseActivationCode :execrows
UPDATE activation_codes SET used_at = ? WHERE code = ? AND used_at IS NULL;

-- name: ListPendingActivations :many
SELECT sqlc.embed(activation_codes), sqlc.embed(users)
FROM activation_codes
JOIN users ON users.id = activation_codes.user_id
WHERE activation_codes.used_at IS NULL AND users.activated = FALSE
ORDER BY users.username;

-- name: Leaderboard :many
SELECT u.id, u.username,
       CAST(SUM((d.points * p.points_multiplier + p.points_adder) * s.parts) AS REAL) AS score
FROM users u
JOIN solves s       ON s.user_id = u.id
JOIN problems p     ON p.id = s.problem_id
JOIN difficulties d ON d.id = p.difficulty_id
WHERE u.admin = FALSE AND u.activated = TRUE AND p.contest_id = ?
GROUP BY u.id, u.username
HAVING score > 0
ORDER BY score DESC, u.username ASC;
