package listing

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, l *Listing) error
	GetByID(ctx context.Context, id int64) (*Listing, error)
	List(ctx context.Context, category string, minPrice, maxPrice float64) ([]*Listing, error)
	Update(ctx context.Context, l *Listing) error
	Delete(ctx context.Context, id int64) error
}

type repo struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repo{db: db}
}

func (r *repo) Create(ctx context.Context, l *Listing) error {
	query := `INSERT INTO listings (user_id, title, description, price, category)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id, is_boosted, created_at`

	err := r.db.QueryRow(ctx,
		query,
		l.UserID,
		l.Title,
		l.Description,
		l.Price,
		l.Category).Scan(&l.ID, &l.IsBoosted, &l.CreatedAt)
	if err != nil {
		return err
	}

	return fmt.Errorf("create listing: %w", err)
}

func (r *repo) GetByID(ctx context.Context, id int64) (*Listing, error) {
	return nil, nil
}

func (r *repo) List(
	ctx context.Context,
	category string, minPrice,
	maxPrice float64,
) ([]*Listing, error) {
	return nil, nil
}

func (r *repo) Update(ctx context.Context, l *Listing) error {
	return nil
}

func (r *repo) Delete(ctx context.Context, id int64) error {
	return nil
}
