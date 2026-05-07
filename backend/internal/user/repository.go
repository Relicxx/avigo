package user

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type repo struct {
	db *pgxpool.Pool
}

func (r *repo) Create(ctx context.Context, u *User) error {
	return nil
}

func (r *repo) GetByEmail(ctx context.Context, email string) (*User, error) {
	return nil, nil
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repo{db: db}
}
