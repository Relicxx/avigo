// Package outbox реализует transactional outbox: событие пишется в таблицу
// outbox в той же транзакции, что и доменная запись, а фоновый relay
// переносит его в брокер. Так событие не теряется при недоступности Kafka —
// доставка получается at-least-once, консьюмеры должны быть идемпотентны.
package outbox

import (
	"context"
	"log/slog"
	"time"
)

// Message — одно событие из таблицы outbox.
type Message struct {
	ID      int64
	Topic   string
	Key     string
	Payload []byte
}

// PublishFunc доставляет одно сообщение в брокер. Ошибка останавливает
// текущую пачку: порядок сохраняется, сообщение будет повторено
// на следующем цикле опроса.
type PublishFunc func(ctx context.Context, m Message) error

// Store выбирает пачку неопубликованных сообщений, передаёт их publish
// и помечает доставленные — всё в одной транзакции.
// Возвращает число опубликованных сообщений.
type Store interface {
	ProcessPending(ctx context.Context, limit int, publish PublishFunc) (int, error)
}

// Publisher публикует событие в брокер (реализуется kafka.Producer).
type Publisher interface {
	Publish(ctx context.Context, topic string, key, value []byte) error
}

// Relay периодически опрашивает outbox и публикует накопившиеся события.
type Relay struct {
	store     Store
	publisher Publisher
	interval  time.Duration
	batchSize int
}

func NewRelay(store Store, publisher Publisher, interval time.Duration, batchSize int) *Relay {
	return &Relay{store: store, publisher: publisher, interval: interval, batchSize: batchSize}
}

// Run опрашивает outbox до отмены контекста. Запускается отдельной
// горутиной; для graceful shutdown достаточно отменить контекст.
func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	slog.Info("outbox relay started", "interval", r.interval, "batch_size", r.batchSize)

	for {
		select {
		case <-ctx.Done():
			slog.Info("outbox relay stopped")
			return
		case <-ticker.C:
			r.drain(ctx)
		}
	}
}

// drain обрабатывает полные пачки подряд, чтобы накопившийся хвост
// разбирался за один цикл опроса, а не по одной пачке на тик.
func (r *Relay) drain(ctx context.Context) {
	for {
		n, err := r.store.ProcessPending(ctx, r.batchSize, func(ctx context.Context, m Message) error {
			return r.publisher.Publish(ctx, m.Topic, []byte(m.Key), m.Payload)
		})
		if err != nil {
			if ctx.Err() == nil {
				slog.Error("outbox batch failed, will retry", "published", n, "error", err)
			}
			return
		}
		if n < r.batchSize {
			return
		}
	}
}
