-- name: CreateDifficulty :one
INSERT INTO difficulties (name, points) VALUES (?, ?) RETURNING *;

-- name: GetDifficultyByName :one
SELECT * FROM difficulties WHERE name = ?;

-- name: ListDifficulties :many
SELECT * FROM difficulties ORDER BY points, name;

-- name: CreateProblem :one
INSERT INTO problems (contest_id, difficulty_id, slug, name, author, parts, points_multiplier, points_adder)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetProblemBySlug :one
SELECT sqlc.embed(problems), sqlc.embed(contests), sqlc.embed(difficulties)
FROM problems
JOIN contests     ON contests.id = problems.contest_id
JOIN difficulties ON difficulties.id = problems.difficulty_id
WHERE problems.slug = ?;

-- name: ListProblemsByContest :many
SELECT sqlc.embed(problems), sqlc.embed(difficulties)
FROM problems
JOIN difficulties ON difficulties.id = problems.difficulty_id
WHERE problems.contest_id = ?
ORDER BY difficulties.points, problems.name;

-- name: UpdateProblem :execrows
UPDATE problems SET
    slug              = COALESCE(sqlc.narg('slug'), slug),
    name              = COALESCE(sqlc.narg('name'), name),
    author            = COALESCE(sqlc.narg('author'), author),
    parts             = COALESCE(sqlc.narg('parts'), parts),
    points_multiplier = COALESCE(sqlc.narg('points_multiplier'), points_multiplier),
    points_adder      = COALESCE(sqlc.narg('points_adder'), points_adder),
    difficulty_id     = COALESCE(sqlc.narg('difficulty_id'), difficulty_id)
WHERE id = sqlc.arg('id');

-- name: DeleteProblem :execrows
DELETE FROM problems WHERE id = ?;

-- name: UpsertProblemInput :exec
INSERT INTO problem_inputs (problem_id, user_id, input) VALUES (?, ?, ?)
ON CONFLICT (problem_id, user_id) DO UPDATE SET input = excluded.input;

-- name: UpsertProblemOutput :exec
INSERT INTO problem_outputs (problem_id, user_id, part, output) VALUES (?, ?, ?, ?)
ON CONFLICT (problem_id, user_id, part) DO UPDATE SET output = excluded.output;

-- name: GetProblemInput :one
SELECT input FROM problem_inputs WHERE problem_id = ? AND user_id = ?;

-- name: GetProblemOutput :one
SELECT output FROM problem_outputs WHERE problem_id = ? AND user_id = ? AND part = ?;

-- name: GetSolvedParts :one
SELECT CAST(COALESCE((SELECT parts FROM solves WHERE user_id = ? AND problem_id = ?), 0) AS INTEGER);

-- name: UpsertSolve :exec
INSERT INTO solves (user_id, problem_id, parts, solved_at) VALUES (?, ?, ?, ?)
ON CONFLICT (user_id, problem_id) DO UPDATE
    SET parts = excluded.parts, solved_at = excluded.solved_at
    WHERE excluded.parts > solves.parts;
