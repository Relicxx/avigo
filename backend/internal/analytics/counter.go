package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// counterTTL ограничивает срок жизни суточных счётчиков,
// чтобы Redis не рос бесконечно.
const counterTTL = 90 * 24 * time.Hour

type redisCounter struct {
	client *redis.Client
}

// NewRedisCounter создаёт Counter поверх Redis. TTL выставляется
// один раз при первом инкременте ключа.
func NewRedisCounter(client *redis.Client) Counter {
	return &redisCounter{client: client}
}

func (c *redisCounter) Inc(ctx context.Context, key string) error {
	pipe := c.client.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, counterTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("incr %s: %w", key, err)
	}
	return nil
}
