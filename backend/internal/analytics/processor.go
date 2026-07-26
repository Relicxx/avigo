// Package analytics агрегирует доменные события Kafka в суточные счётчики.
package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// Counter инкрементирует именованный счётчик.
type Counter interface {
	Inc(ctx context.Context, key string) error
}

// event — общий payload событий listing.created и boost.created.
type event struct {
	ID        int64 `json:"id"`
	ListingID int64 `json:"listing_id"`
	UserID    int64 `json:"user_id"`
}

// Processor превращает события в метрики продукта:
// по каждому топику ведётся суточный счётчик stats:<topic>:<YYYY-MM-DD>.
type Processor struct {
	counter Counter
	now     func() time.Time
}

func NewProcessor(counter Counter) *Processor {
	return &Processor{counter: counter, now: time.Now}
}

// Handle валидирует payload события и инкрементирует счётчик дня.
func (p *Processor) Handle(ctx context.Context, topic string, payload []byte) error {
	var e event
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode %s event: %w", topic, err)
	}
	if e.ID == 0 || e.UserID == 0 {
		return fmt.Errorf("invalid %s event: id and user_id are required", topic)
	}

	key := fmt.Sprintf("stats:%s:%s", topic, p.now().UTC().Format(time.DateOnly))
	if err := p.counter.Inc(ctx, key); err != nil {
		return fmt.Errorf("increment %s: %w", key, err)
	}

	slog.Info("event processed", "topic", topic, "id", e.ID, "user_id", e.UserID)
	return nil
}
