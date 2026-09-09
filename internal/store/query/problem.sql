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
SELECT sqlc.embed(problems), sqlc.embed(difficulties), CAST((
    SELECT COUNT(*)
    FROM users u
    WHERE EXISTS (
        SELECT 1 FROM problem_inputs pi
        WHERE pi.problem_id = problems.id AND pi.user_id = u.id
    ) AND (
        SELECT COUNT(DISTINCT po.part) FROM problem_outputs po
        WHERE po.problem_id = problems.id AND po.user_id = u.id
          AND po.part BETWEEN 1 AND problems.parts
    ) = problems.parts
) AS INTEGER) AS complete_users
FROM problems
JOIN difficulties ON difficulties.id = problems.difficulty_id
WHERE problems.contest_id = ?
ORDER BY difficulties.points, problems.name;

-- name: HasCompleteProblemData :one
SELECT CAST(EXISTS (
    SELECT 1
    FROM problems p
    WHERE p.id = sqlc.arg('problem_id')
      AND EXISTS (
        SELECT 1 FROM problem_inputs pi
        WHERE pi.problem_id = p.id AND pi.user_id = sqlc.arg('user_id')
      )
      AND (
          SELECT COUNT(DISTINCT po.part) FROM problem_outputs po
          WHERE po.problem_id = p.id AND po.user_id = sqlc.arg('user_id')
            AND po.part BETWEEN 1 AND p.parts
      ) = p.parts
) AS BOOLEAN);

-- name: ListCompleteProblemsByUser :many
SELECT sqlc.embed(problems), sqlc.embed(contests), sqlc.embed(difficulties)
FROM problems
JOIN contests ON contests.id = problems.contest_id
JOIN difficulties ON difficulties.id = problems.difficulty_id
WHERE EXISTS (
    SELECT 1 FROM problem_inputs pi
    WHERE pi.problem_id = problems.id AND pi.user_id = sqlc.arg('user_id')
) AND (
    SELECT COUNT(DISTINCT po.part) FROM problem_outputs po
    WHERE po.problem_id = problems.id AND po.user_id = sqlc.arg('user_id')
      AND po.part BETWEEN 1 AND problems.parts
) = problems.parts
ORDER BY contests.start_at DESC, difficulties.points, problems.name;

-- name: ListCompleteUsersByProblem :many
SELECT users.*
FROM users
JOIN problems ON problems.id = ?
WHERE EXISTS (
    SELECT 1 FROM problem_inputs pi
    WHERE pi.problem_id = problems.id AND pi.user_id = users.id
) AND (
    SELECT COUNT(DISTINCT po.part) FROM problem_outputs po
    WHERE po.problem_id = problems.id AND po.user_id = users.id
      AND po.part BETWEEN 1 AND problems.parts
) = problems.parts
ORDER BY users.username;

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
