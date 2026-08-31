package outbox

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Enqueue кладёт событие в outbox в рамках переданной транзакции —
// вызывается репозиториями рядом с доменной записью.
func Enqueue(ctx context.Context, tx pgx.Tx, topic, key string, payload []byte) error {
	if _, err := tx.Exec(ctx,
		`INSERT INTO outbox (topic, key, payload) VALUES ($1, $2, $3)`,
		topic, key, payload,
	); err != nil {
		return fmt.Errorf("enqueue outbox event %s: %w", topic, err)
	}
	return nil
}

// PgStore разбирает таблицу outbox для relay.
type PgStore struct {
	db *pgxpool.Pool
}

func NewPgStore(db *pgxpool.Pool) *PgStore {
	return &PgStore{db: db}
}

// ProcessPending захватывает до limit неопубликованных сообщений через
// FOR UPDATE SKIP LOCKED (безопасно при нескольких инстансах API),
// публикует их в порядке вставки и помечает доставленные — в одной
// транзакции. При ошибке публикации пачка останавливается на упавшем
// сообщении: всё до него помечается опубликованным, остальное повторится
// на следующем опросе. Крэш между publish и commit приводит к повторной
// отправке — отсюда at-least-once.
func (s *PgStore) ProcessPending(ctx context.Context, limit int, publish PublishFunc) (published int, err error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback после commit — no-op

	rows, err := tx.Query(ctx, `
		SELECT id, topic, key, payload
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY id
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return 0, fmt.Errorf("select pending outbox messages: %w", err)
	}

	msgs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Message, error) {
		var m Message
		err := row.Scan(&m.ID, &m.Topic, &m.Key, &m.Payload)
		return m, err
	})
	if err != nil {
		return 0, fmt.Errorf("scan outbox messages: %w", err)
	}
	if len(msgs) == 0 {
		return 0, tx.Commit(ctx)
	}

	var publishErr error
	ids := make([]int64, 0, len(msgs))
	for _, m := range msgs {
		if pubErr := publish(ctx, m); pubErr != nil {
			publishErr = fmt.Errorf("publish outbox message %d: %w", m.ID, pubErr)
			break
		}
		ids = append(ids, m.ID)
	}

	if len(ids) > 0 {
		if _, err := tx.Exec(ctx,
			`UPDATE outbox SET published_at = NOW() WHERE id = ANY($1)`, ids); err != nil {
			return 0, fmt.Errorf("mark outbox messages published: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit outbox tx: %w", err)
	}

	return len(ids), publishErr
}
