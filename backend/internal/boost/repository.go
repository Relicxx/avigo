package boost

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Relicxx/avigo/internal/outbox"
	"github.com/Relicxx/avigo/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	// CreateActive в одной транзакции проверяет, что активного буста нет,
	// создаёт новый и выставляет listings.is_boosted = true.
	// При активном бусте возвращает ErrAlreadyBoosted.
	CreateActive(ctx context.Context, b *Boost) error
	// ListingOwner возвращает user_id владельца объявления.
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

func (r *repo) CreateActive(ctx context.Context, b *Boost) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin boost tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback после commit — no-op

	var hasActive bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM boosts WHERE listing_id = $1 AND expires_at > NOW())`,
		b.ListingID,
	).Scan(&hasActive)
	if err != nil {
		return fmt.Errorf("check active boost: %w", err)
	}
	if hasActive {
		return ErrAlreadyBoosted
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO boosts (listing_id, user_id, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		b.ListingID, b.UserID, b.ExpiresAt,
	).Scan(&b.ID, &b.CreatedAt)
	if err != nil {
		return storage.MapError("create boost", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE listings SET is_boosted = TRUE WHERE id = $1`,
		b.ListingID,
	); err != nil {
		return fmt.Errorf("mark listing boosted: %w", err)
	}

	// Событие boost.created уходит в outbox той же транзакцией:
	// доставку в Kafka выполняет relay.
	if err := outbox.Enqueue(ctx, tx, "boost.created",
		strconv.FormatInt(b.ID, 10),
		[]byte(fmt.Sprintf(`{"id":%d,"listing_id":%d,"user_id":%d}`, b.ID, b.ListingID, b.UserID)),
	); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit boost tx: %w", err)
	}

	return nil
}

// ListingOwner возвращает user_id владельца объявления.
func (r *repo) ListingOwner(ctx context.Context, listingID int64) (int64, error) {
	query := `SELECT user_id FROM listings WHERE id = $1`

	var ownerID int64
	if err := r.db.QueryRow(ctx, query, listingID).Scan(&ownerID); err != nil {
		return 0, storage.MapError("get listing owner", err)
	}

	return ownerID, nil
}
