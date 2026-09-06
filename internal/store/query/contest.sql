-- name: CreateContest :one
INSERT INTO contests (slug, name, description, start_at, end_at, unlisted)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetContest :one
SELECT * FROM contests WHERE id = ?;

-- name: GetContestBySlug :one
SELECT * FROM contests WHERE slug = ?;

-- name: ListContests :many
SELECT * FROM contests ORDER BY start_at DESC;

-- name: ListArchivedContests :many
SELECT * FROM contests WHERE unlisted = FALSE ORDER BY start_at DESC;

-- name: UpdateContest :execrows
UPDATE contests SET
    slug        = COALESCE(sqlc.narg('slug'), slug),
    name        = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    unlisted    = COALESCE(sqlc.narg('unlisted'), unlisted),
    start_at    = COALESCE(sqlc.narg('start_at'), start_at),
    end_at      = COALESCE(sqlc.narg('end_at'), end_at)
WHERE id = sqlc.arg('id');

-- name: DeleteContest :execrows
DELETE FROM contests WHERE id = ?;
