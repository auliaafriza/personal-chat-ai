-- Minggu 6 — Hybrid search (tsvector untuk BM25-like full-text).
--
-- Pakai config 'simple' karena content kemungkinan multilingual (Indonesia/Inggris)
-- dan Postgres nggak punya stemmer Indonesian built-in. 'simple' = no stemming,
-- just lowercase + tokenize.
--
-- Generated column = auto-fill saat insert + auto-recompute kalau content berubah.
-- Sintaks ini support sejak Postgres 12.

ALTER TABLE document_chunks
    ADD COLUMN content_tsv tsvector
    GENERATED ALWAYS AS (to_tsvector('simple', content)) STORED;

CREATE INDEX chunks_content_tsv_idx ON document_chunks USING GIN (content_tsv);
