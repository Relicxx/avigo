package chat

import (
	"context"
	"fmt"

	"github.com/Relicxx/avigo/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Send(ctx context.Context, m *Message) error
	// GetForUser возвращает сообщения по объявлению, в которых участвует userID
	// (как отправитель или получатель). Чужая переписка не видна.
	GetForUser(ctx context.Context, listingID, userID int64, limit, offset int) ([]*Message, error)
	// HasMessageFrom проверяет, писал ли senderID получателю receiverID по объявлению.
	HasMessageFrom(ctx context.Context, listingID, senderID, receiverID int64) (bool, error)
	// ListingOwner возвращает user_id владельца объявления
	// (apperr.ErrNotFound, если объявления не существует).
	ListingOwner(ctx context.Context, listingID int64) (int64, error)
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
		return storage.MapError("send message", err)
	}
	return nil
}

func (r *repo) GetForUser(ctx context.Context, listingID, userID int64, limit, offset int) ([]*Message, error) {
	query := `SELECT id, listing_id, sender_id, receiver_id, body, created_at
	FROM messages
	WHERE listing_id = $1 AND (sender_id = $2 OR receiver_id = $2)
	ORDER BY created_at ASC, id ASC
	LIMIT $3 OFFSET $4`
	rows, err := r.db.Query(ctx, query, listingID, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	msgs := []*Message{}
	for rows.Next() {
		m := &Message{}
		if err := rows.Scan(&m.ID, &m.ListingID, &m.SenderID, &m.ReceiverID, &m.Body, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *repo) HasMessageFrom(ctx context.Context, listingID, senderID, receiverID int64) (bool, error) {
	query := `SELECT EXISTS (
		SELECT 1 FROM messages
		WHERE listing_id = $1 AND sender_id = $2 AND receiver_id = $3
	)`
	var exists bool
	if err := r.db.QueryRow(ctx, query, listingID, senderID, receiverID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check conversation: %w", err)
	}
	return exists, nil
}

func (r *repo) ListingOwner(ctx context.Context, listingID int64) (int64, error) {
	query := `SELECT user_id FROM listings WHERE id = $1`
	var ownerID int64
	if err := r.db.QueryRow(ctx, query, listingID).Scan(&ownerID); err != nil {
		return 0, storage.MapError("get listing owner", err)
	}
	return ownerID, nil
}
