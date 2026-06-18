package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
)

type TaskRepo struct {
	pool *pgxpool.Pool
}

func NewTaskRepo(pool *pgxpool.Pool) *TaskRepo {
	return &TaskRepo{pool: pool}
}

type CreateTaskParams struct {
	UserID      string
	Title       string
	Description string
	DueDate     *time.Time
	IsReminder  bool
}

func (r *TaskRepo) Create(ctx context.Context, p CreateTaskParams) (Task, error) {
	id := ulid.Make().String()
	row := r.pool.QueryRow(ctx, `
		INSERT INTO tasks (id, user_id, title, description, due_date, is_reminder)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, title, description, due_date, is_reminder,
		          completed, completed_at, created_at, updated_at
	`, id, p.UserID, p.Title, p.Description, p.DueDate, p.IsReminder)
	return scanTask(row)
}

// TaskFilter — list filter. All optional; combine where multiple set.
type TaskFilter struct {
	Status string // "" | "pending" | "completed"
	Due    string // "" | "overdue" | "today" | "upcoming" | "no_due"
}

func (r *TaskRepo) ListByUser(ctx context.Context, userID string, filter TaskFilter, limit int) ([]Task, error) {
	if limit <= 0 {
		limit = 100
	}

	q := `
		SELECT id, user_id, title, description, due_date, is_reminder,
		       completed, completed_at, created_at, updated_at
		FROM tasks
		WHERE user_id = $1
	`
	args := []any{userID}
	switch filter.Status {
	case "pending":
		q += ` AND completed = false`
	case "completed":
		q += ` AND completed = true`
	}

	now := time.Now()
	switch filter.Due {
	case "overdue":
		args = append(args, now)
		q += ` AND due_date IS NOT NULL AND due_date < $` + ph(len(args))
	case "today":
		tomorrow := now.Add(24 * time.Hour)
		args = append(args, now, tomorrow)
		q += ` AND due_date IS NOT NULL AND due_date >= $` + ph(len(args)-1) + ` AND due_date < $` + ph(len(args))
	case "upcoming":
		args = append(args, now)
		q += ` AND due_date IS NOT NULL AND due_date >= $` + ph(len(args))
	case "no_due":
		q += ` AND due_date IS NULL`
	}

	q += ` ORDER BY completed ASC, due_date NULLS LAST ASC, created_at ASC LIMIT $` + ph(len(args)+1)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Task, 0, limit)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TaskRepo) GetByUser(ctx context.Context, id, userID string) (Task, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, title, description, due_date, is_reminder,
		       completed, completed_at, created_at, updated_at
		FROM tasks WHERE id = $1 AND user_id = $2
	`, id, userID)
	t, err := scanTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	return t, err
}

type UpdateTaskParams struct {
	Title       *string
	Description *string
	DueDate     **time.Time // pointer to pointer: nil = nggak update, *nil = clear
	Completed   *bool
}

func (r *TaskRepo) UpdateByUser(ctx context.Context, id, userID string, p UpdateTaskParams) (Task, error) {
	var (
		dueSet     bool
		dueValue   *time.Time
		completedAt *time.Time
		now        = time.Now()
	)
	if p.DueDate != nil {
		dueSet = true
		dueValue = *p.DueDate
	}
	if p.Completed != nil && *p.Completed {
		completedAt = &now
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE tasks
		SET title        = COALESCE($3, title),
		    description  = COALESCE($4, description),
		    due_date     = CASE WHEN $5::boolean THEN $6 ELSE due_date END,
		    completed    = COALESCE($7, completed),
		    completed_at = CASE
		                       WHEN $7::boolean = true  AND completed = false THEN $8
		                       WHEN $7::boolean = false                       THEN NULL
		                       ELSE completed_at
		                   END,
		    updated_at   = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, title, description, due_date, is_reminder,
		          completed, completed_at, created_at, updated_at
	`, id, userID, p.Title, p.Description, dueSet, dueValue, p.Completed, completedAt)

	t, err := scanTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	return t, err
}

func (r *TaskRepo) DeleteByUser(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanTask(row scanner) (Task, error) {
	var t Task
	err := row.Scan(
		&t.ID, &t.UserID, &t.Title, &t.Description, &t.DueDate, &t.IsReminder,
		&t.Completed, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt,
	)
	return t, err
}

// ph turns 1..N into "1"..."N" — placeholder helper for dynamic query building.
func ph(n int) string {
	// strconv.Itoa avoid pakai fmt.Sprintf (lebih cepat & zero-alloc untuk integer kecil).
	const digits = "0123456789"
	if n < 10 {
		return string(digits[n])
	}
	return string(digits[n/10]) + string(digits[n%10])
}
