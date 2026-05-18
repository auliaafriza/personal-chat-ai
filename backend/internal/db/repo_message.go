package db

import (
	"context"

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
}

func (r *MessageRepo) Create(ctx context.Context, p CreateMessageParams) (Message, error) {
	id := ulid.Make().String()
	row := r.pool.QueryRow(ctx, `
		INSERT INTO messages (id, conversation_id, role, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, conversation_id, role, content, created_at
	`, id, p.ConversationID, p.Role, p.Content)

	var m Message
	err := row.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt)
	return m, err
}

func (r *MessageRepo) ListByConversation(ctx context.Context, conversationID string) ([]Message, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, conversation_id, role, content, created_at
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
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
