package db

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
)

type MessageRepo struct {
	pool *pgxpool.Pool
}

func NewMessageRepo(pool *pgxpool.Pool) *MessageRepo {
	return &MessageRepo{pool: pool}
}

type CreateMessageParams struct {
	ConversationID string
	Role           MessageRole
	Content        string
	Sources        []Source // optional — RAG citations untuk assistant message
}

func (r *MessageRepo) Create(ctx context.Context, p CreateMessageParams) (Message, error) {
	id := ulid.Make().String()

	// Marshal sources ke JSON string (nil → NULL). Pass sebagai *string supaya
	// $5::jsonb cast bekerja; []byte akan di-encode pgx sebagai bytea.
	var sourcesJSON *string
	if len(p.Sources) > 0 {
		b, err := json.Marshal(p.Sources)
		if err != nil {
			return Message{}, err
		}
		s := string(b)
		sourcesJSON = &s
	}

	row := r.pool.QueryRow(ctx, `
		INSERT INTO messages (id, conversation_id, role, content, sources)
		VALUES ($1, $2, $3, $4, $5::jsonb)
		RETURNING id, conversation_id, role, content, sources, created_at
	`, id, p.ConversationID, p.Role, p.Content, sourcesJSON)

	return scanMessage(row)
}

func (r *MessageRepo) ListByConversation(ctx context.Context, conversationID string) ([]Message, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, conversation_id, role, content, sources, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Message, 0, 32)
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func scanMessage(row scanner) (Message, error) {
	var m Message
	var sourcesRaw []byte // jsonb → raw JSON bytes (nil kalau NULL)
	if err := row.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &sourcesRaw, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	if len(sourcesRaw) > 0 {
		// Best-effort: kalau corrupt, biarkan Sources nil daripada gagal request.
		_ = json.Unmarshal(sourcesRaw, &m.Sources)
	}
	return m, nil
}
