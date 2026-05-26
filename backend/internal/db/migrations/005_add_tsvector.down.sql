DROP INDEX IF EXISTS chunks_content_tsv_idx;
ALTER TABLE document_chunks DROP COLUMN IF EXISTS content_tsv;
