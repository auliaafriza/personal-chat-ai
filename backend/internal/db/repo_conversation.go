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
	Title        string
	Model        string
	SystemPrompt *string
	Temperature  float64
}

func (r *ConversationRepo) Create(ctx context.Context, p CreateConversationParams) (Conversation, error) {
	id := ulid.Make().String()
	row := r.pool.QueryRow(ctx, `
		INSERT INTO conversations (id, title, model, system_prompt, temperature)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, model, system_prompt, temperature, created_at, updated_at
	`, id, p.Title, p.Model, p.SystemPrompt, p.Temperature)

	return scanConversation(row)
}

func (r *ConversationRepo) List(ctx context.Context, limit int) ([]Conversation, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, model, system_prompt, temperature, created_at, updated_at
		FROM conversations
		ORDER BY updated_at DESC
		LIMIT $1
	`, limit)
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

func (r *ConversationRepo) Get(ctx context.Context, id string) (Conversation, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, title, model, system_prompt, temperature, created_at, updated_at
		FROM conversations WHERE id = $1
	`, id)
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

func (r *ConversationRepo) Update(ctx context.Context, id string, p UpdateConversationParams) (Conversation, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE conversations
		SET title         = COALESCE($2, title),
		    model         = COALESCE($3, model),
		    system_prompt = CASE WHEN $4::boolean THEN $5 ELSE system_prompt END,
		    temperature   = COALESCE($6, temperature),
		    updated_at    = $7
		WHERE id = $1
		RETURNING id, title, model, system_prompt, temperature, created_at, updated_at
	`,
		id,
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

func (r *ConversationRepo) Touch(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *ConversationRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM conversations WHERE id = $1`, id)
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
	err := row.Scan(&c.ID, &c.Title, &c.Model, &c.SystemPrompt, &c.Temperature, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func derefStringPtr(pp **string) *string {
	if pp == nil {
		return nil
	}
	return *pp
}
