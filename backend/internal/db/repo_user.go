package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

type UpsertUserParams struct {
	GoogleSub string
	Email     string
	Name      string
	AvatarURL string
}

// Upsert returns the existing user (or newly-created one) keyed by google_sub.
// On existing user we refresh email/name/avatar in case Google profile changed.
func (r *UserRepo) Upsert(ctx context.Context, p UpsertUserParams) (User, error) {
	id := ulid.Make().String()
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (id, google_sub, email, name, avatar_url)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (google_sub) DO UPDATE
		SET email      = EXCLUDED.email,
		    name       = EXCLUDED.name,
		    avatar_url = EXCLUDED.avatar_url,
		    updated_at = NOW()
		RETURNING id, google_sub, email, name, avatar_url,
		          default_model, default_temperature, system_prompt,
		          created_at, updated_at
	`, id, p.GoogleSub, p.Email, p.Name, p.AvatarURL)

	return scanUser(row)
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, google_sub, email, name, avatar_url,
		       default_model, default_temperature, system_prompt,
		       created_at, updated_at
		FROM users WHERE id = $1
	`, id)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

type UpdateUserSettingsParams struct {
	DefaultModel       *string
	DefaultTemperature *float64
	SystemPrompt       *string
}

func (r *UserRepo) UpdateSettings(ctx context.Context, id string, p UpdateUserSettingsParams) (User, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE users
		SET default_model       = COALESCE($2, default_model),
		    default_temperature = COALESCE($3, default_temperature),
		    system_prompt       = COALESCE($4, system_prompt),
		    updated_at          = NOW()
		WHERE id = $1
		RETURNING id, google_sub, email, name, avatar_url,
		          default_model, default_temperature, system_prompt,
		          created_at, updated_at
	`, id, p.DefaultModel, p.DefaultTemperature, p.SystemPrompt)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func scanUser(row scanner) (User, error) {
	var u User
	err := row.Scan(
		&u.ID, &u.GoogleSub, &u.Email, &u.Name, &u.AvatarURL,
		&u.DefaultModel, &u.DefaultTemperature, &u.SystemPrompt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	return u, err
}
