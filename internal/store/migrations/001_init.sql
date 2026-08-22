CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL UNIQUE COLLATE NOCASE,
    email         TEXT    NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT    NOT NULL DEFAULT '',
    activated     BOOLEAN NOT NULL DEFAULT FALSE,
    admin         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    INTEGER NOT NULL
);

CREATE TABLE activation_codes (
    code       TEXT    PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL,
    used_at    INTEGER
);

CREATE INDEX idx_activation_codes_user ON activation_codes(user_id);

CREATE TABLE contests (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    slug        TEXT    NOT NULL UNIQUE,
    name        TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    start_at    INTEGER NOT NULL,
    end_at      INTEGER NOT NULL
);

CREATE TABLE difficulties (
    id     INTEGER PRIMARY KEY AUTOINCREMENT,
    name   TEXT    NOT NULL UNIQUE,
    points INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE problems (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    contest_id        INTEGER NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    difficulty_id     INTEGER NOT NULL REFERENCES difficulties(id),
    slug              TEXT    NOT NULL UNIQUE,
    name              TEXT    NOT NULL,
    author            TEXT    NOT NULL DEFAULT '',
    parts             INTEGER NOT NULL DEFAULT 1,
    points_multiplier REAL    NOT NULL DEFAULT 1.0,
    points_adder      INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_problems_contest ON problems(contest_id);

CREATE TABLE problem_inputs (
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    input      TEXT    NOT NULL,
    PRIMARY KEY (problem_id, user_id)
);

CREATE TABLE problem_outputs (
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    part       INTEGER NOT NULL,
    output     TEXT    NOT NULL,
    PRIMARY KEY (problem_id, user_id, part)
);

CREATE TABLE solves (
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    parts      INTEGER NOT NULL,
    solved_at  INTEGER NOT NULL,
    PRIMARY KEY (user_id, problem_id)
);

CREATE TABLE site_config (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    site_title      TEXT NOT NULL DEFAULT '',
    main_text       TEXT NOT NULL DEFAULT '',
    secondary_text  TEXT NOT NULL DEFAULT '',
    current_contest TEXT NOT NULL DEFAULT '',
    help_content    TEXT NOT NULL DEFAULT '',
    rules_content   TEXT NOT NULL DEFAULT '',
    legal_content   TEXT NOT NULL DEFAULT '',
    credits_content TEXT NOT NULL DEFAULT ''
);
