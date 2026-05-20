DROP INDEX IF EXISTS chunks_embedding_idx;
DROP INDEX IF EXISTS chunks_user_idx;
DROP INDEX IF EXISTS chunks_document_idx;
DROP TABLE IF EXISTS document_chunks;

DROP INDEX IF EXISTS documents_user_idx;
DROP TABLE IF EXISTS documents;

-- Note: nggak drop vector extension karena mungkin dipakai migration lain.
