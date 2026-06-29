package boost

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, b *Boost) error
	GetActiveByListingID(ctx context.Context, listingID int64) (*Boost, error)
	ListingOwner(ctx context.Context, listingID int64) (int64, error)
}

type repo struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repo{
		db: db,
	}
}

func (r *repo) Create(ctx context.Context, b *Boost) error {
	query := `INSERT INTO boosts (listing_id, user_id,expires_at)
	VALUES ($1, $2, $3)
	RETURNING id, created_at`

	b.ExpiresAt = time.Now().Add(24 * time.Hour)
	row := r.db.QueryRow(ctx, query, b.ListingID, b.UserID, b.ExpiresAt)

	err := row.Scan(&b.ID, &b.CreatedAt)
	if err != nil {
		return fmt.Errorf("create boost: %w", err)
	}

	return nil
}

// ListingOwner возвращает user_id владельца объявления.
func (r *repo) ListingOwner(ctx context.Context, listingID int64) (int64, error) {
	query := `SELECT user_id FROM listings WHERE id = $1`

	var ownerID int64
	if err := r.db.QueryRow(ctx, query, listingID).Scan(&ownerID); err != nil {
		return 0, fmt.Errorf("get listing owner: %w", err)
	}

	return ownerID, nil
}

func (r *repo) GetActiveByListingID(ctx context.Context, listingID int64) (*Boost, error) {
	query := `SELECT id, listing_id, user_id, expires_at, created_at
	FROM boosts
	WHERE listing_id = $1 AND expires_at > NOW()
	ORDER BY created_at DESC
	LIMIT 1`

	row := r.db.QueryRow(ctx, query, listingID)
	b := &Boost{}
	err := row.Scan(&b.ID, &b.ListingID, &b.UserID, &b.ExpiresAt, &b.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get active boost: %w", err)
	}

	return b, nil
}
