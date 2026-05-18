-- Minggu 2 schema (single-user mode).
-- Minggu 3 nanti tambah `users` table + FK ke `user_id`.

CREATE TYPE message_role AS ENUM ('user', 'assistant', 'system');

CREATE TABLE conversations (
    id              TEXT PRIMARY KEY,
    title           TEXT NOT NULL DEFAULT 'New chat',
    model           TEXT NOT NULL DEFAULT 'claude-sonnet-4-6',
    system_prompt   TEXT,
    temperature     REAL NOT NULL DEFAULT 0.7,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX conv_updated_at_idx ON conversations (updated_at DESC);

CREATE TABLE messages (
    id                  TEXT PRIMARY KEY,
    conversation_id     TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role                message_role NOT NULL,
    content             TEXT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX msg_conv_idx ON messages (conversation_id, created_at ASC);
