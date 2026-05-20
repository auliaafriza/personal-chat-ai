-- Minggu 4 — Embeddings + pgvector.
--
-- Enable pgvector di Neon (free tier auto-enabled, tapi statement-nya idempotent).
-- Dimensions = 512 untuk Voyage voyage-3-lite. Kalau ganti model nanti ke voyage-3
-- (1024 dim) butuh migration baru + re-embed semua chunks.

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE documents (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    source_type     TEXT NOT NULL,   -- 'txt' | 'md' | 'pdf' | 'docx' | 'paste'
    source_size     INTEGER NOT NULL DEFAULT 0, -- bytes (informational)
    content         TEXT NOT NULL,   -- raw extracted text (untuk re-chunk kalau strategi berubah)
    chunk_count     INTEGER NOT NULL DEFAULT 0,
    embedding_model TEXT NOT NULL DEFAULT 'voyage-3-lite',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX documents_user_idx ON documents (user_id, created_at DESC);

CREATE TABLE document_chunks (
    id              TEXT PRIMARY KEY,
    document_id     TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    position        INTEGER NOT NULL, -- 0-based index dalam document
    heading         TEXT NOT NULL DEFAULT '', -- closest preceding markdown heading (kalau ada)
    content         TEXT NOT NULL,
    embedding       vector(512) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX chunks_document_idx ON document_chunks (document_id, position ASC);
CREATE INDEX chunks_user_idx ON document_chunks (user_id);

-- HNSW index untuk cosine similarity. m=16, ef_construction=64 = default yang OK
-- buat dataset kecil-menengah. Untuk dataset >100k chunks pertimbangkan ef_construction lebih tinggi.
CREATE INDEX chunks_embedding_idx ON document_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
