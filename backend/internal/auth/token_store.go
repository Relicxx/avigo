package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenStore хранит выданные refresh-токены (по jti). Токен одноразовый:
// Consume атомарно читает и удаляет запись, реализуя ротацию.
type TokenStore interface {
	Save(ctx context.Context, jti string, userID int64, ttl time.Duration) error
	// Consume возвращает userID и удаляет запись. Если jti нет (отозван,
	// истёк или уже использован) — возвращает ErrInvalidToken.
	Consume(ctx context.Context, jti string) (int64, error)
	Delete(ctx context.Context, jti string) error
}

type redisTokenStore struct {
	client *redis.Client
}

// NewRedisTokenStore создаёт TokenStore поверх Redis.
func NewRedisTokenStore(client *redis.Client) TokenStore {
	return &redisTokenStore{client: client}
}

func refreshKey(jti string) string {
	return "auth:refresh:" + jti
}

func (s *redisTokenStore) Save(ctx context.Context, jti string, userID int64, ttl time.Duration) error {
	if err := s.client.Set(ctx, refreshKey(jti), userID, ttl).Err(); err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

func (s *redisTokenStore) Consume(ctx context.Context, jti string) (int64, error) {
	val, err := s.client.GetDel(ctx, refreshKey(jti)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, ErrInvalidToken
	}
	if err != nil {
		return 0, fmt.Errorf("consume refresh token: %w", err)
	}
	userID, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse stored user id: %w", err)
	}
	return userID, nil
}

func (s *redisTokenStore) Delete(ctx context.Context, jti string) error {
	if err := s.client.Del(ctx, refreshKey(jti)).Err(); err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}
