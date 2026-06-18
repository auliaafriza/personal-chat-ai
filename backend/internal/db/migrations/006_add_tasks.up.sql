-- Minggu 9 — Tasks (productivity tools).
--
-- Schema sederhana: title + optional due_date + completed flag.
-- is_reminder = true kalau dibuat via remind_me tool (vs explicit task).

CREATE TABLE tasks (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    due_date      TIMESTAMPTZ,
    is_reminder   BOOLEAN NOT NULL DEFAULT false,
    completed     BOOLEAN NOT NULL DEFAULT false,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index dipake oleh list filter umum: per-user, sorted by due (overdue first).
CREATE INDEX tasks_user_due_idx
    ON tasks (user_id, completed, due_date NULLS LAST);
