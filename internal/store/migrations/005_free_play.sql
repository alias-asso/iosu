ALTER TABLE contests ADD COLUMN mode TEXT NOT NULL DEFAULT 'normal'
    CHECK (mode IN ('normal', 'free_play'));
ALTER TABLE contests ADD COLUMN infinite BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE problem_free_play_inputs (
    problem_id INTEGER PRIMARY KEY REFERENCES problems(id) ON DELETE CASCADE,
    input      TEXT NOT NULL
);

CREATE TABLE problem_free_play_outputs (
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    part       INTEGER NOT NULL,
    output     TEXT NOT NULL,
    PRIMARY KEY (problem_id, part)
);

DROP INDEX idx_generation_tasks_run;
DROP INDEX idx_generation_tasks_status;
DROP INDEX idx_generation_tasks_active_pair;
ALTER TABLE generation_tasks RENAME TO generation_tasks_old;

CREATE TABLE generation_tasks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id      INTEGER NOT NULL REFERENCES generation_runs(id) ON DELETE CASCADE,
    problem_id  INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    user_id     INTEGER REFERENCES users(id) ON DELETE CASCADE,
    shared      BOOLEAN NOT NULL DEFAULT FALSE,
    status      TEXT    NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'skipped')),
    error       TEXT    NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    started_at  INTEGER,
    finished_at INTEGER,
    CHECK ((shared = FALSE AND user_id IS NOT NULL) OR (shared = TRUE AND user_id IS NULL))
);

INSERT INTO generation_tasks
    (id, run_id, problem_id, user_id, shared, status, error, created_at, started_at, finished_at)
SELECT id, run_id, problem_id, user_id, FALSE, status, error, created_at, started_at, finished_at
FROM generation_tasks_old;

DROP TABLE generation_tasks_old;

CREATE INDEX idx_generation_tasks_run ON generation_tasks(run_id);
CREATE INDEX idx_generation_tasks_status ON generation_tasks(status, id);
CREATE UNIQUE INDEX idx_generation_tasks_active_user
ON generation_tasks(problem_id, user_id)
WHERE shared = FALSE AND status IN ('queued', 'running');
CREATE UNIQUE INDEX idx_generation_tasks_active_shared
ON generation_tasks(problem_id)
WHERE shared = TRUE AND status IN ('queued', 'running');
