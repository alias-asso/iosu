-- name: CreateGenerationRun :one
INSERT INTO generation_runs (contest_id, source, mode, status, created_at)
VALUES (?, ?, ?, 'queued', ?)
RETURNING *;

-- name: CreateGenerationTask :one
INSERT INTO generation_tasks (run_id, problem_id, user_id, status, error, created_at, finished_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: HasActiveGenerationTask :one
SELECT CAST(EXISTS (
    SELECT 1 FROM generation_tasks
    WHERE problem_id = ? AND user_id = ? AND status IN ('queued', 'running')
) AS BOOLEAN);

-- name: ClaimGenerationTask :one
UPDATE generation_tasks
SET status = 'running', started_at = ?, finished_at = NULL, error = ''
WHERE id = (
    SELECT id FROM generation_tasks WHERE status = 'queued' ORDER BY id LIMIT 1
)
RETURNING *;

-- name: GetGenerationTaskDetail :one
SELECT sqlc.embed(generation_tasks), sqlc.embed(generation_runs),
       sqlc.embed(users), sqlc.embed(problems), sqlc.embed(contests)
FROM generation_tasks
JOIN generation_runs ON generation_runs.id = generation_tasks.run_id
JOIN users ON users.id = generation_tasks.user_id
JOIN problems ON problems.id = generation_tasks.problem_id
JOIN contests ON contests.id = problems.contest_id
WHERE generation_tasks.id = ?;

-- name: UpdateGenerationTask :exec
UPDATE generation_tasks
SET status = ?, error = ?, finished_at = ?
WHERE id = ?;

-- name: MarkGenerationRunStarted :exec
UPDATE generation_runs
SET status = 'running', started_at = COALESCE(started_at, ?)
WHERE id = ? AND status = 'queued';

-- name: GenerationTaskCounts :one
SELECT
    CAST(COUNT(*) AS INTEGER) AS total,
    CAST(COUNT(*) FILTER (WHERE status = 'queued') AS INTEGER) AS queued,
    CAST(COUNT(*) FILTER (WHERE status = 'running') AS INTEGER) AS running,
    CAST(COUNT(*) FILTER (WHERE status = 'succeeded') AS INTEGER) AS succeeded,
    CAST(COUNT(*) FILTER (WHERE status = 'failed') AS INTEGER) AS failed,
    CAST(COUNT(*) FILTER (WHERE status = 'skipped') AS INTEGER) AS skipped
FROM generation_tasks
WHERE run_id = ?;

-- name: FinishGenerationRun :exec
UPDATE generation_runs
SET status = ?, finished_at = ?
WHERE id = ?;

-- name: FinishIdleGenerationRuns :exec
UPDATE generation_runs
SET status = CASE
        WHEN EXISTS (
            SELECT 1 FROM generation_tasks
            WHERE run_id = generation_runs.id AND status = 'failed'
        ) THEN 'completed_with_errors'
        ELSE 'completed'
    END,
    finished_at = ?
WHERE status IN ('queued', 'running')
  AND NOT EXISTS (
      SELECT 1 FROM generation_tasks
      WHERE run_id = generation_runs.id AND status IN ('queued', 'running')
  );

-- name: RecoverGenerationTasks :exec
UPDATE generation_tasks
SET status = 'queued', started_at = NULL, error = ''
WHERE status = 'running';

-- name: RecoverGenerationRuns :exec
UPDATE generation_runs
SET status = 'queued', started_at = NULL, finished_at = NULL
WHERE status = 'running';

-- name: ListGenerationRuns :many
SELECT generation_runs.*,
       contests.name AS contest_name,
       CAST(COUNT(generation_tasks.id) AS INTEGER) AS total,
       CAST(COUNT(generation_tasks.id) FILTER (WHERE generation_tasks.status = 'queued') AS INTEGER) AS queued,
       CAST(COUNT(generation_tasks.id) FILTER (WHERE generation_tasks.status = 'running') AS INTEGER) AS running,
       CAST(COUNT(generation_tasks.id) FILTER (WHERE generation_tasks.status = 'succeeded') AS INTEGER) AS succeeded,
       CAST(COUNT(generation_tasks.id) FILTER (WHERE generation_tasks.status = 'failed') AS INTEGER) AS failed,
       CAST(COUNT(generation_tasks.id) FILTER (WHERE generation_tasks.status = 'skipped') AS INTEGER) AS skipped
FROM generation_runs
JOIN contests ON contests.id = generation_runs.contest_id
LEFT JOIN generation_tasks ON generation_tasks.run_id = generation_runs.id
WHERE generation_runs.status IN ('queued', 'running')
   OR generation_runs.id IN (SELECT id FROM generation_runs ORDER BY id DESC LIMIT ?)
GROUP BY generation_runs.id
ORDER BY generation_runs.id DESC
;

-- name: GetGenerationRun :one
SELECT generation_runs.*, contests.name AS contest_name
FROM generation_runs
JOIN contests ON contests.id = generation_runs.contest_id
WHERE generation_runs.id = ?;

-- name: ListGenerationTasksByRun :many
SELECT generation_tasks.*, users.username, problems.name AS problem_name
FROM generation_tasks
JOIN users ON users.id = generation_tasks.user_id
JOIN problems ON problems.id = generation_tasks.problem_id
WHERE generation_tasks.run_id = ?
ORDER BY generation_tasks.id;
