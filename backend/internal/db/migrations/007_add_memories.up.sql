-- Minggu 10 — Long-term memory.
--
-- Memories = persistent facts tentang user yang di-inject ke setiap chat
-- sebagai personalisasi. Bedanya sama documents:
--   - Lebih short (1-2 kalimat)
--   - User-scoped sangat ketat (nggak ada sharing)
--   - Selalu top-3 di-inject (vs RAG yang threshold-gated)
--   - Punya category supaya bisa di-organize di UI
--
-- Vector(512) sama dengan documents pakai Voyage voyage-3-lite.

CREATE TABLE user_memories (
    id                     TEXT PRIMARY KEY,
    user_id                TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content                TEXT NOT NULL,
    category               TEXT NOT NULL DEFAULT 'general',
    embedding              vector(512) NOT NULL,
    source_conversation_id TEXT, -- nullable: nggak FK karena conversation bisa di-delete
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX memories_user_idx ON user_memories (user_id, category, created_at DESC);

CREATE INDEX memories_embedding_idx ON user_memories
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
