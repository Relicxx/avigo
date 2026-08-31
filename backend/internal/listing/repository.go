package listing

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Relicxx/avigo/internal/apperr"
	"github.com/Relicxx/avigo/internal/outbox"
	"github.com/Relicxx/avigo/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Filter — параметры выборки объявлений. Nil-цены означают отсутствие фильтра
// (в отличие от «0 = нет фильтра», ноль — валидная цена).
// Query — полнотекстовый поиск по title + description; пустая строка — без поиска.
type Filter struct {
	Category string
	Query    string
	MinPrice *float64
	MaxPrice *float64
	Limit    int
	Offset   int
}

type Repository interface {
	Create(ctx context.Context, l *Listing) error
	GetByID(ctx context.Context, id int64) (*Listing, error)
	List(ctx context.Context, f Filter) ([]*Listing, error)
	Update(ctx context.Context, l *Listing) error
	Delete(ctx context.Context, id, userID int64) error
}

type repo struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repo{db: db}
}

// Create вставляет объявление и событие listing.created в outbox
// в одной транзакции: событие не потеряется при недоступности Kafka.
func (r *repo) Create(ctx context.Context, l *Listing) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create listing tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback после commit — no-op

	query := `INSERT INTO listings (user_id, title, description, price, category)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id, is_boosted, created_at`

	err = tx.QueryRow(ctx,
		query,
		l.UserID,
		l.Title,
		l.Description,
		l.Price,
		l.Category).Scan(&l.ID, &l.IsBoosted, &l.CreatedAt)
	if err != nil {
		return fmt.Errorf("create listing: %w", err)
	}

	if err := outbox.Enqueue(ctx, tx, "listing.created",
		strconv.FormatInt(l.ID, 10),
		[]byte(fmt.Sprintf(`{"id":%d,"user_id":%d}`, l.ID, l.UserID)),
	); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create listing tx: %w", err)
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

func (r *repo) List(ctx context.Context, f Filter) ([]*Listing, error) {
	// Поиск идёт по generated-колонке search_tsv (GIN-индекс, конфигурация
	// 'simple'). websearch_to_tsquery безопасно парсит пользовательский ввод:
	// синтаксическая ошибка в запросе не роняет выборку.
	query := `SELECT l.id, l.user_id, l.title, l.description, l.price, l.category, l.is_boosted, l.created_at
	FROM listings l
	WHERE ($1 = '' OR l.category = $1)
		AND ($2::numeric IS NULL OR l.price >= $2)
		AND ($3::numeric IS NULL OR l.price <= $3)
		AND ($4 = '' OR l.search_tsv @@ websearch_to_tsquery('simple', $4))
	ORDER BY (l.is_boosted AND EXISTS (
			SELECT 1 FROM boosts b
			WHERE b.listing_id = l.id AND b.expires_at > NOW()
		)) DESC,
		l.created_at DESC
	LIMIT $5 OFFSET $6`

	rows, err := r.db.Query(ctx, query, f.Category, f.MinPrice, f.MaxPrice, f.Query, f.Limit, f.Offset)
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
