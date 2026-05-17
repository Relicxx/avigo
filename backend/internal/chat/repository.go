package chat

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Send(ctx context.Context, m *Message) error
	GetByListing(ctx context.Context, listingID int64) ([]*Message, error)
}

type repo struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repo{db: db}
}

func (r *repo) Send(ctx context.Context, m *Message) error {
	query := `INSERT INTO messages (listing_id, sender_id, receiver_id, body)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at`
	err := r.db.QueryRow(ctx, query, m.ListingID, m.SenderID, m.ReceiverID, m.Body).
		Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

func (r *repo) GetByListing(ctx context.Context, listingID int64) ([]*Message, error) {
	query := `SELECT id, listing_id, sender_id, receiver_id, body, created_at
	FROM messages
	WHERE listing_id = $1 ORDER BY created_at ASC`
	rows, err := r.db.Query(ctx, query, listingID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()
	var msgs []*Message
	for rows.Next() {
		m := &Message{}
		if err := rows.Scan(&m.ID, &m.ListingID, &m.SenderID, &m.ReceiverID, &m.Body, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
