package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/Relicxx/avigo/pkg/middleware"
	"github.com/redis/go-redis/v9"
)

type rateLimitStore struct {
	client *redis.Client
}

// NewRateLimitStore создаёт хранилище счётчиков rate limit поверх Redis.
// Счётчик и TTL окна ставятся атомарно (транзакционный pipeline);
// TTL — только при создании ключа, чтобы окно не продлевалось запросами.
func NewRateLimitStore(client *redis.Client) middleware.RateLimitStore {
	return &rateLimitStore{client: client}
}

func (s *rateLimitStore) Incr(ctx context.Context, key string, window time.Duration) (int64, error) {
	pipe := s.client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("rate limit incr %s: %w", key, err)
	}
	return incr.Val(), nil
}
