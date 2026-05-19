-- Minggu 3 — Auth + per-user settings.
-- 1) users table (Google OAuth: google_sub jadi unique key utama, email duplikatif kalau user ganti Google account)
-- 2) per-user default settings (model, temperature, system_prompt)
-- 3) tambah user_id FK ke conversations (NULLABLE dulu untuk backfill row lama; nanti diset NOT NULL setelah migrasi data manual)

CREATE TABLE users (
    id                  TEXT PRIMARY KEY,
    google_sub          TEXT NOT NULL UNIQUE,
    email               TEXT NOT NULL,
    name                TEXT NOT NULL DEFAULT '',
    avatar_url          TEXT NOT NULL DEFAULT '',
    -- Per-user defaults (settings page)
    default_model       TEXT NOT NULL DEFAULT 'llama-3.3-70b-versatile',
    default_temperature REAL NOT NULL DEFAULT 0.7,
    system_prompt       TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX users_email_idx ON users (email);

-- Add user_id FK ke conversations.
-- NULLABLE supaya migration nggak break row lama; production app harus reject row tanpa user_id via app logic.
ALTER TABLE conversations
    ADD COLUMN user_id TEXT REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX conv_user_idx ON conversations (user_id, updated_at DESC);
