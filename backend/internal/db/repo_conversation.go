package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
)

var ErrNotFound = errors.New("not found")

type ConversationRepo struct {
	pool *pgxpool.Pool
}

func NewConversationRepo(pool *pgxpool.Pool) *ConversationRepo {
	return &ConversationRepo{pool: pool}
}

type CreateConversationParams struct {
	UserID       string
	Title        string
	Model        string
	SystemPrompt *string
	Temperature  float64
}

func (r *ConversationRepo) Create(ctx context.Context, p CreateConversationParams) (Conversation, error) {
	id := ulid.Make().String()
	row := r.pool.QueryRow(ctx, `
		INSERT INTO conversations (id, user_id, title, model, system_prompt, temperature)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, title, model, system_prompt, temperature, created_at, updated_at
	`, id, p.UserID, p.Title, p.Model, p.SystemPrompt, p.Temperature)

	return scanConversation(row)
}

// ListByUser returns conversations scoped to a user, sorted by most recent.
func (r *ConversationRepo) ListByUser(ctx context.Context, userID string, limit int) ([]Conversation, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, title, model, system_prompt, temperature, created_at, updated_at
		FROM conversations
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Conversation, 0, limit)
	for rows.Next() {
		conv, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, conv)
	}
	return out, rows.Err()
}

// GetByUser fetches a conversation iff it belongs to userID — otherwise ErrNotFound.
// Treating wrong-owner as 404 (bukan 403) supaya nggak bocorin existence of resource.
func (r *ConversationRepo) GetByUser(ctx context.Context, id, userID string) (Conversation, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, title, model, system_prompt, temperature, created_at, updated_at
		FROM conversations WHERE id = $1 AND user_id = $2
	`, id, userID)
	conv, err := scanConversation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, ErrNotFound
	}
	return conv, err
}

type UpdateConversationParams struct {
	Title        *string
	Model        *string
	SystemPrompt **string // pointer to pointer so we can distinguish nil vs not-set
	Temperature  *float64
}

func (r *ConversationRepo) UpdateByUser(ctx context.Context, id, userID string, p UpdateConversationParams) (Conversation, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE conversations
		SET title         = COALESCE($3, title),
		    model         = COALESCE($4, model),
		    system_prompt = CASE WHEN $5::boolean THEN $6 ELSE system_prompt END,
		    temperature   = COALESCE($7, temperature),
		    updated_at    = $8
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, title, model, system_prompt, temperature, created_at, updated_at
	`,
		id,
		userID,
		p.Title,
		p.Model,
		p.SystemPrompt != nil,
		derefStringPtr(p.SystemPrompt),
		p.Temperature,
		time.Now(),
	)
	conv, err := scanConversation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, ErrNotFound
	}
	return conv, err
}

func (r *ConversationRepo) TouchByUser(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ConversationRepo) DeleteByUser(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM conversations WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// scanConversation accepts either pgx.Row or pgx.Rows (both have Scan).
type scanner interface {
	Scan(dest ...any) error
}

func scanConversation(row scanner) (Conversation, error) {
	var c Conversation
	err := row.Scan(&c.ID, &c.UserID, &c.Title, &c.Model, &c.SystemPrompt, &c.Temperature, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func derefStringPtr(pp **string) *string {
	if pp == nil {
		return nil
	}
	return *pp
}
