package user

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type repo struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repo{db: db}
}

func (r *repo) Create(ctx context.Context, u *User) error {
	query := `INSERT INTO users (email, password_hash)
	VALUES ($1, $2)
	RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query, u.Email, u.PasswordHash).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *repo) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	query := `SELECT id, email, password_hash, created_at
	FROM users
	WHERE email = $1`

	err := r.db.QueryRow(
		ctx,
		query,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}
