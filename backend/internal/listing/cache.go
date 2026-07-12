package listing

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache — cache-aside для списков объявлений. Инвалидация версионируемая:
// Invalidate инкрементирует версию, и все старые ключи перестают читаться
// (доживают до истечения TTL) — без дорогого redis KEYS.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration)
	// Version возвращает текущую версию кэша списков (входит в ключ).
	Version(ctx context.Context) int64
	// Invalidate повышает версию, делая все закешированные списки неактуальными.
	Invalidate(ctx context.Context)
}

const cacheVersionKey = "listings:ver"

type redisCache struct {
	client *redis.Client
}

// NewRedisCache создаёт Cache поверх Redis. Кэш best-effort:
// ошибки Redis логируются и не ломают запрос.
func NewRedisCache(client *redis.Client) Cache {
	return &redisCache{client: client}
}

func (c *redisCache) Get(ctx context.Context, key string) ([]byte, bool) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			slog.Warn("cache get failed", "error", err, "key", key)
		}
		return nil, false
	}
	return data, true
}

func (c *redisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) {
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		slog.Warn("cache set failed", "error", err, "key", key)
	}
}

func (c *redisCache) Version(ctx context.Context) int64 {
	ver, err := c.client.Get(ctx, cacheVersionKey).Int64()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			slog.Warn("cache version read failed", "error", err)
		}
		return 0
	}
	return ver
}

func (c *redisCache) Invalidate(ctx context.Context) {
	if err := c.client.Incr(ctx, cacheVersionKey).Err(); err != nil {
		slog.Warn("cache invalidate failed", "error", err)
	}
}
