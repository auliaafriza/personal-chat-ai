package db

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
)

type DocumentRepo struct {
	pool *pgxpool.Pool
}

func NewDocumentRepo(pool *pgxpool.Pool) *DocumentRepo {
	return &DocumentRepo{pool: pool}
}

type CreateDocumentParams struct {
	UserID         string
	Title          string
	SourceType     string
	SourceSize     int
	Content        string
	ChunkCount     int
	EmbeddingModel string
}

// Insert document + all chunks in a single transaction. Chunks slice is parallel
// to embeddings slice (same len; chunks[i] ⇔ embeddings[i]).
func (r *DocumentRepo) CreateWithChunks(
	ctx context.Context,
	p CreateDocumentParams,
	chunks []ChunkInput,
	embeddings [][]float32,
) (Document, error) {
	if len(chunks) != len(embeddings) {
		return Document{}, fmt.Errorf("chunks/embeddings length mismatch: %d vs %d", len(chunks), len(embeddings))
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Document{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	docID := ulid.Make().String()
	row := tx.QueryRow(ctx, `
		INSERT INTO documents (id, user_id, title, source_type, source_size, content, chunk_count, embedding_model)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, title, source_type, source_size, content, chunk_count, embedding_model, created_at
	`, docID, p.UserID, p.Title, p.SourceType, p.SourceSize, p.Content, p.ChunkCount, p.EmbeddingModel)

	var doc Document
	if err := row.Scan(
		&doc.ID, &doc.UserID, &doc.Title, &doc.SourceType, &doc.SourceSize,
		&doc.Content, &doc.ChunkCount, &doc.EmbeddingModel, &doc.CreatedAt,
	); err != nil {
		return Document{}, fmt.Errorf("insert document: %w", err)
	}

	// Batch insert chunks. pgx batch jadi 1 round-trip per chunk; untuk MVP cukup.
	batch := &pgx.Batch{}
	for i, ch := range chunks {
		chunkID := ulid.Make().String()
		vec := vectorLiteral(embeddings[i])
		batch.Queue(`
			INSERT INTO document_chunks (id, document_id, user_id, position, heading, content, embedding)
			VALUES ($1, $2, $3, $4, $5, $6, $7::vector)
		`, chunkID, docID, p.UserID, ch.Position, ch.Heading, ch.Content, vec)
	}

	br := tx.SendBatch(ctx, batch)
	for i := 0; i < len(chunks); i++ {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return Document{}, fmt.Errorf("insert chunk %d: %w", i, err)
		}
	}
	if err := br.Close(); err != nil {
		return Document{}, fmt.Errorf("close batch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Document{}, fmt.Errorf("commit: %w", err)
	}

	return doc, nil
}

// ChunkInput — minimal subset of DocumentChunk untuk CreateWithChunks (tanpa ID/IDs yang di-generate DB).
type ChunkInput struct {
	Position int
	Heading  string
	Content  string
}

// ListByUser returns documents (tanpa content, biar list ringan).
func (r *DocumentRepo) ListByUser(ctx context.Context, userID string, limit int) ([]Document, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, title, source_type, source_size, '' as content, chunk_count, embedding_model, created_at
		FROM documents
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Document, 0, limit)
	for rows.Next() {
		var d Document
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.Title, &d.SourceType, &d.SourceSize,
			&d.Content, &d.ChunkCount, &d.EmbeddingModel, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *DocumentRepo) GetByUser(ctx context.Context, id, userID string) (Document, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, title, source_type, source_size, content, chunk_count, embedding_model, created_at
		FROM documents WHERE id = $1 AND user_id = $2
	`, id, userID)

	var d Document
	if err := row.Scan(
		&d.ID, &d.UserID, &d.Title, &d.SourceType, &d.SourceSize,
		&d.Content, &d.ChunkCount, &d.EmbeddingModel, &d.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Document{}, ErrNotFound
		}
		return Document{}, err
	}
	return d, nil
}

// CountChunksByUser returns the total number of chunks the user has across all
// documents. Dipakai chat handler untuk decide apakah RAG perlu di-jalankan.
func (r *DocumentRepo) CountChunksByUser(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM document_chunks WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}

func (r *DocumentRepo) DeleteByUser(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListChunksByDocument returns chunks (ordered by position) for a document.
// Caller harus verify ownership via GetByUser dulu.
func (r *DocumentRepo) ListChunksByDocument(ctx context.Context, docID string) ([]DocumentChunk, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, document_id, user_id, position, heading, content, created_at
		FROM document_chunks
		WHERE document_id = $1
		ORDER BY position ASC
	`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DocumentChunk, 0, 32)
	for rows.Next() {
		var c DocumentChunk
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.UserID, &c.Position, &c.Heading, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SearchHybrid does Reciprocal Rank Fusion (RRF) atas:
//   - vector top-N (cosine `<=>`) via HNSW index
//   - BM25-like top-N (Postgres ts_rank dengan `websearch_to_tsquery`) via GIN index
//
// RRF formula: score = Σ (1 / (k + rank_i)) untuk tiap retriever yang ngasih
// chunk itu. Standard k=60. Hasilnya stable & parameter-free (nggak perlu
// normalisasi skor antar retriever yang scale-nya beda).
//
// `candidateLimit` = berapa banyak per-retriever yang dipertimbangkan (typically 20).
// `topK` = berapa hasil final yang di-return. Kalau topK > unique chunks dari
// kedua retriever, return semua yang ada.
//
// Output sorted by RRFScore desc. SearchResult.Similarity diset ke RRFScore
// supaya konsisten dengan code path yang belum di-rerank. Setelah rerank,
// caller boleh overwrite Similarity = RerankScore.
func (r *DocumentRepo) SearchHybrid(
	ctx context.Context,
	userID, query string,
	queryEmbedding []float32,
	candidateLimit, topK int,
) ([]SearchResult, error) {
	if candidateLimit <= 0 {
		candidateLimit = 20
	}
	if topK <= 0 {
		topK = 5
	}
	if topK > candidateLimit*2 {
		topK = candidateLimit * 2
	}

	vec := vectorLiteral(queryEmbedding)
	const rrfK = 60

	// Single SQL biar atomic (vector + BM25 + RRF). FULL OUTER JOIN supaya
	// chunk yang cuma muncul di salah satu retriever tetap dipertimbangkan.
	const sql = `
		WITH vector_results AS (
		    SELECT
		        c.id, c.document_id, c.user_id, c.position, c.heading, c.content, c.created_at,
		        d.title AS document_title,
		        1 - (c.embedding <=> $1::vector) AS vector_score,
		        ROW_NUMBER() OVER (ORDER BY c.embedding <=> $1::vector ASC) AS vector_rank
		    FROM document_chunks c
		    JOIN documents d ON d.id = c.document_id
		    WHERE c.user_id = $2
		    ORDER BY c.embedding <=> $1::vector ASC
		    LIMIT $4
		),
		bm25_results AS (
		    SELECT
		        c.id, c.document_id, c.user_id, c.position, c.heading, c.content, c.created_at,
		        d.title AS document_title,
		        ts_rank(c.content_tsv, q.query) AS bm25_score,
		        ROW_NUMBER() OVER (ORDER BY ts_rank(c.content_tsv, q.query) DESC) AS bm25_rank
		    FROM document_chunks c
		    JOIN documents d ON d.id = c.document_id,
		         websearch_to_tsquery('simple', $3) AS q(query)
		    WHERE c.user_id = $2 AND c.content_tsv @@ q.query
		    ORDER BY ts_rank(c.content_tsv, q.query) DESC
		    LIMIT $4
		)
		SELECT
		    COALESCE(v.id, b.id)                 AS id,
		    COALESCE(v.document_id, b.document_id),
		    COALESCE(v.user_id, b.user_id),
		    COALESCE(v.position, b.position),
		    COALESCE(v.heading, b.heading),
		    COALESCE(v.content, b.content),
		    COALESCE(v.created_at, b.created_at),
		    COALESCE(v.document_title, b.document_title),
		    COALESCE(v.vector_score, 0)          AS vector_score,
		    COALESCE(b.bm25_score, 0)            AS bm25_score,
		    COALESCE(1.0 / ($5::float + v.vector_rank), 0)
		      + COALESCE(1.0 / ($5::float + b.bm25_rank), 0) AS rrf_score
		FROM vector_results v
		FULL OUTER JOIN bm25_results b ON v.id = b.id
		ORDER BY rrf_score DESC
		LIMIT $6
	`

	rows, err := r.pool.Query(ctx, sql, vec, userID, query, candidateLimit, rrfK, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]SearchResult, 0, topK)
	for rows.Next() {
		var sr SearchResult
		if err := rows.Scan(
			&sr.ID, &sr.DocumentID, &sr.UserID, &sr.Position, &sr.Heading, &sr.Content, &sr.CreatedAt,
			&sr.DocumentTitle, &sr.VectorScore, &sr.BM25Score, &sr.RRFScore,
		); err != nil {
			return nil, err
		}
		sr.Similarity = sr.RRFScore
		out = append(out, sr)
	}
	return out, rows.Err()
}

// vectorLiteral renders a float32 slice as pgvector text literal "[0.123,0.456,...]"
// — bukan pakai pgvector-go binding karena kita mau zero extra deps di pgx.
func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.Grow(len(v) * 10)
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(x), 'f', 6, 32))
	}
	b.WriteByte(']')
	return b.String()
}
