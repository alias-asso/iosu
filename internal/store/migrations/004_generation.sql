ALTER TABLE contests ADD COLUMN auto_generate BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE generation_runs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    contest_id  INTEGER NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    source      TEXT    NOT NULL CHECK (source IN ('automatic', 'manual')),
    mode        TEXT    NOT NULL CHECK (mode IN ('replace', 'missing')),
    status      TEXT    NOT NULL CHECK (status IN ('queued', 'running', 'completed', 'completed_with_errors')),
    created_at  INTEGER NOT NULL,
    started_at  INTEGER,
    finished_at INTEGER
);

CREATE TABLE generation_tasks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id      INTEGER NOT NULL REFERENCES generation_runs(id) ON DELETE CASCADE,
    problem_id  INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status      TEXT    NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'skipped')),
    error       TEXT    NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    started_at  INTEGER,
    finished_at INTEGER
);

CREATE INDEX idx_generation_tasks_run ON generation_tasks(run_id);
CREATE INDEX idx_generation_tasks_status ON generation_tasks(status, id);
CREATE UNIQUE INDEX idx_generation_tasks_active_pair
ON generation_tasks(problem_id, user_id)
WHERE status IN ('queued', 'running');
