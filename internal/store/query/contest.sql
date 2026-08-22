-- name: CreateContest :one
INSERT INTO contests (slug, name, description, start_at, end_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetContest :one
SELECT * FROM contests WHERE id = ?;

-- name: GetContestBySlug :one
SELECT * FROM contests WHERE slug = ?;

-- name: ListContests :many
SELECT * FROM contests ORDER BY start_at DESC;

-- name: UpdateContest :execrows
UPDATE contests SET
    slug        = COALESCE(sqlc.narg('slug'), slug),
    name        = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    start_at    = COALESCE(sqlc.narg('start_at'), start_at),
    end_at      = COALESCE(sqlc.narg('end_at'), end_at)
WHERE id = sqlc.arg('id');
