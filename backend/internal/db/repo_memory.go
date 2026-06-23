package db

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
)

type MemoryRepo struct {
	pool *pgxpool.Pool
}

func NewMemoryRepo(pool *pgxpool.Pool) *MemoryRepo {
	return &MemoryRepo{pool: pool}
}

type CreateMemoryParams struct {
	UserID               string
	Content              string
	Category             string
	Embedding            []float32
	SourceConversationID *string
}

func (r *MemoryRepo) Create(ctx context.Context, p CreateMemoryParams) (Memory, error) {
	id := ulid.Make().String()
	vec := vectorLiteral(p.Embedding)
	row := r.pool.QueryRow(ctx, `
		INSERT INTO user_memories (id, user_id, content, category, embedding, source_conversation_id)
		VALUES ($1, $2, $3, $4, $5::vector, $6)
		RETURNING id, user_id, content, category, source_conversation_id, created_at, updated_at
	`, id, p.UserID, p.Content, p.Category, vec, p.SourceConversationID)
	return scanMemory(row)
}

type ListMemoriesFilter struct {
	Category string // optional
	Query    string // optional substring match (for UI search; vector search separate)
}

func (r *MemoryRepo) ListByUser(ctx context.Context, userID string, filter ListMemoriesFilter, limit int) ([]Memory, error) {
	if limit <= 0 {
		limit = 200
	}
	q := `
		SELECT id, user_id, content, category, source_conversation_id, created_at, updated_at
		FROM user_memories
		WHERE user_id = $1
	`
	args := []any{userID}
	if filter.Category != "" {
		args = append(args, filter.Category)
		q += ` AND category = $2`
	}
	if filter.Query != "" {
		args = append(args, "%"+strings.ToLower(filter.Query)+"%")
		q += ` AND LOWER(content) LIKE $` + ph(len(args))
	}
	args = append(args, limit)
	q += ` ORDER BY created_at DESC LIMIT $` + ph(len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Memory, 0, limit)
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MemoryRepo) GetByUser(ctx context.Context, id, userID string) (Memory, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, content, category, source_conversation_id, created_at, updated_at
		FROM user_memories WHERE id = $1 AND user_id = $2
	`, id, userID)
	m, err := scanMemory(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Memory{}, ErrNotFound
	}
	return m, err
}

type UpdateMemoryParams struct {
	Content   *string
	Category  *string
	Embedding []float32 // kalau content berubah, caller harus re-embed
}

func (r *MemoryRepo) UpdateByUser(ctx context.Context, id, userID string, p UpdateMemoryParams) (Memory, error) {
	var (
		embSet bool
		vec    string
	)
	if len(p.Embedding) > 0 {
		embSet = true
		vec = vectorLiteral(p.Embedding)
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE user_memories
		SET content    = COALESCE($3, content),
		    category   = COALESCE($4, category),
		    embedding  = CASE WHEN $5::boolean THEN $6::vector ELSE embedding END,
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, content, category, source_conversation_id, created_at, updated_at
	`, id, userID, p.Content, p.Category, embSet, vec)

	m, err := scanMemory(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Memory{}, ErrNotFound
	}
	return m, err
}

func (r *MemoryRepo) DeleteByUser(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM user_memories WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SearchSimilar runs cosine-similarity search over user's memories.
// Lebih simpel daripada documents (no BM25 / no rerank) — memory content
// pendek, vector cukup. Result sorted by similarity desc.
func (r *MemoryRepo) SearchSimilar(ctx context.Context, userID string, queryEmbedding []float32, topK int) ([]Memory, error) {
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}
	vec := vectorLiteral(queryEmbedding)

	rows, err := r.pool.Query(ctx, `
		SELECT
		    id, user_id, content, category, source_conversation_id, created_at, updated_at,
		    1 - (embedding <=> $1::vector) AS similarity
		FROM user_memories
		WHERE user_id = $2
		ORDER BY embedding <=> $1::vector ASC
		LIMIT $3
	`, vec, userID, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Memory, 0, topK)
	for rows.Next() {
		var m Memory
		if err := rows.Scan(
			&m.ID, &m.UserID, &m.Content, &m.Category, &m.SourceConversationID, &m.CreatedAt, &m.UpdatedAt,
			&m.Similarity,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MemoryRepo) CountByUser(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_memories WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}

func scanMemory(row scanner) (Memory, error) {
	var m Memory
	err := row.Scan(&m.ID, &m.UserID, &m.Content, &m.Category, &m.SourceConversationID, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}
