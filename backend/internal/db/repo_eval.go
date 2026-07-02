package db

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
)

type EvalRepo struct {
	pool *pgxpool.Pool
}

func NewEvalRepo(pool *pgxpool.Pool) *EvalRepo {
	return &EvalRepo{pool: pool}
}

// --- eval_sets ------------------------------------------------------------

type CreateEvalSetParams struct {
	UserID      string
	Name        string
	Description string
	Queries     []EvalSetQuery
}

func (r *EvalRepo) CreateSet(ctx context.Context, p CreateEvalSetParams) (EvalSet, error) {
	id := ulid.Make().String()
	queriesJSON, _ := json.Marshal(p.Queries)
	queries := string(queriesJSON)
	row := r.pool.QueryRow(ctx, `
		INSERT INTO eval_sets (id, user_id, name, description, queries)
		VALUES ($1, $2, $3, $4, $5::jsonb)
		RETURNING id, user_id, name, description, queries, created_at, updated_at
	`, id, p.UserID, p.Name, p.Description, queries)
	return scanEvalSet(row)
}

func (r *EvalRepo) ListSetsByUser(ctx context.Context, userID string) ([]EvalSet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, name, description, queries, created_at, updated_at
		FROM eval_sets
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EvalSet{}
	for rows.Next() {
		s, err := scanEvalSet(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *EvalRepo) GetSetByUser(ctx context.Context, id, userID string) (EvalSet, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, name, description, queries, created_at, updated_at
		FROM eval_sets WHERE id = $1 AND user_id = $2
	`, id, userID)
	s, err := scanEvalSet(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return EvalSet{}, ErrNotFound
	}
	return s, err
}

func (r *EvalRepo) DeleteSetByUser(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM eval_sets WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- eval_runs ------------------------------------------------------------

type CreateEvalRunParams struct {
	UserID           string
	Kind             string
	EvalSetID        *string
	SubjectMessageID *string
	Results          map[string]any
	DurationMs       int64
}

func (r *EvalRepo) CreateRun(ctx context.Context, p CreateEvalRunParams) (EvalRun, error) {
	id := ulid.Make().String()
	resultsJSON, _ := json.Marshal(p.Results)
	results := string(resultsJSON)
	row := r.pool.QueryRow(ctx, `
		INSERT INTO eval_runs (id, user_id, kind, eval_set_id, subject_message_id, results, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
		RETURNING id, user_id, kind, eval_set_id, subject_message_id, results, duration_ms, created_at
	`, id, p.UserID, p.Kind, p.EvalSetID, p.SubjectMessageID, results, p.DurationMs)
	return scanEvalRun(row)
}

type ListRunsFilter struct {
	Kind      string
	EvalSetID string
}

func (r *EvalRepo) ListRunsByUser(ctx context.Context, userID string, filter ListRunsFilter, limit int) ([]EvalRun, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `
		SELECT id, user_id, kind, eval_set_id, subject_message_id, results, duration_ms, created_at
		FROM eval_runs
		WHERE user_id = $1
	`
	args := []any{userID}
	if filter.Kind != "" {
		args = append(args, filter.Kind)
		q += ` AND kind = $2`
	}
	if filter.EvalSetID != "" {
		args = append(args, filter.EvalSetID)
		q += ` AND eval_set_id = $` + ph(len(args))
	}
	args = append(args, limit)
	q += ` ORDER BY created_at DESC LIMIT $` + ph(len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EvalRun{}
	for rows.Next() {
		r, err := scanEvalRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// --- scanners ------------------------------------------------------------

func scanEvalSet(row scanner) (EvalSet, error) {
	var s EvalSet
	var queriesRaw []byte
	if err := row.Scan(&s.ID, &s.UserID, &s.Name, &s.Description, &queriesRaw, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return EvalSet{}, err
	}
	if len(queriesRaw) > 0 {
		_ = json.Unmarshal(queriesRaw, &s.Queries)
	}
	if s.Queries == nil {
		s.Queries = []EvalSetQuery{}
	}
	return s, nil
}

func scanEvalRun(row scanner) (EvalRun, error) {
	var r EvalRun
	var resultsRaw []byte
	if err := row.Scan(&r.ID, &r.UserID, &r.Kind, &r.EvalSetID, &r.SubjectMessageID, &resultsRaw, &r.DurationMs, &r.CreatedAt); err != nil {
		return EvalRun{}, err
	}
	if len(resultsRaw) > 0 {
		_ = json.Unmarshal(resultsRaw, &r.Results)
	}
	if r.Results == nil {
		r.Results = map[string]any{}
	}
	return r, nil
}
