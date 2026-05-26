-- Minggu 5 — RAG citation persistence.
-- Simpan sources (chunk references) yang dipakai untuk generate assistant message,
-- supaya citation tetap muncul setelah reload (bukan cuma di live stream).
-- Nullable: user message & assistant message tanpa RAG = NULL.

ALTER TABLE messages ADD COLUMN sources JSONB;
