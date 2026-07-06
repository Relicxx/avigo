package listing

import (
	"context"
	"fmt"

	"github.com/Relicxx/avigo/internal/apperr"
	"github.com/Relicxx/avigo/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, l *Listing) error
	GetByID(ctx context.Context, id int64) (*Listing, error)
	List(ctx context.Context, category string, minPrice, maxPrice float64) ([]*Listing, error)
	Update(ctx context.Context, l *Listing) error
	Delete(ctx context.Context, id, userID int64) error
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
		return fmt.Errorf("create listing: %w", err)
	}

	return nil
}

func (r *repo) GetByID(ctx context.Context, id int64) (*Listing, error) {
	query := `SELECT id, user_id, title, description, price, category, is_boosted, created_at
	FROM listings
	WHERE id = $1`

	l := &Listing{}

	err := r.db.QueryRow(ctx, query, id).Scan(
		&l.ID,
		&l.UserID,
		&l.Title,
		&l.Description,
		&l.Price,
		&l.Category,
		&l.IsBoosted,
		&l.CreatedAt,
	)
	if err != nil {
		return nil, storage.MapError("get listing by ID", err)
	}

	return l, nil
}

func (r *repo) List(
	ctx context.Context,
	category string, minPrice,
	maxPrice float64,
) ([]*Listing, error) {
	query := `SELECT id, user_id, title, description, price, category, is_boosted, created_at
	FROM listings
	WHERE ($1 = '' OR category = $1)
		AND ($2 = 0 OR price >= $2)
		AND ($3 = 0 OR price <= $3)
	ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, category, minPrice, maxPrice)
	if err != nil {
		return nil, fmt.Errorf("list listings: %w", err)
	}
	defer rows.Close()

	listings := []*Listing{}

	for rows.Next() {
		l := &Listing{}

		err := rows.Scan(
			&l.ID,
			&l.UserID,
			&l.Title,
			&l.Description,
			&l.Price,
			&l.Category,
			&l.IsBoosted,
			&l.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("list listings: %w", err)
		}

		listings = append(listings, l)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("list listings: %w", err)
	}

	return listings, nil
}

func (r *repo) Update(ctx context.Context, l *Listing) error {
	query := `UPDATE listings
	SET title = $1, description = $2, price = $3, category = $4
	WHERE id = $5 AND user_id = $6`

	tag, err := r.db.Exec(ctx, query, l.Title, l.Description, l.Price, l.Category, l.ID, l.UserID)
	if err != nil {
		return fmt.Errorf("update listing: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}

	return nil
}

func (r *repo) Delete(ctx context.Context, id, userID int64) error {
	query := `DELETE FROM listings WHERE id = $1 AND user_id = $2`

	tag, err := r.db.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("delete listing: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}

	return nil
}
