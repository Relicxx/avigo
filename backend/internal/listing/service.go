package listing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Relicxx/avigo/internal/kafka"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	repo     Repository
	producer *kafka.Producer
	redis    *redis.Client
}

func NewService(repo Repository, producer *kafka.Producer, redis *redis.Client) *Service {
	return &Service{repo: repo, producer: producer, redis: redis}
}

func (s *Service) Create(ctx context.Context, l *Listing) error {
	if err := s.repo.Create(ctx, l); err != nil {
		return err
	}
	s.producer.Publish(ctx, "listing.created",
		[]byte(fmt.Sprintf("%d", l.ID)),
		[]byte(fmt.Sprintf(`{"id":%d,"user_id":%d}`, l.ID, l.UserID)),
	)
	return nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Listing, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, category string, minPrice, maxPrice float64) ([]*Listing, error) {
	cacheKey := fmt.Sprintf("listings:%s:%.2f:%.2f", category, minPrice, maxPrice)

	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var listings []*Listing
		json.Unmarshal([]byte(cached), &listings)
		return listings, nil
	}

	listings, err := s.repo.List(ctx, category, minPrice, maxPrice)
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(listings)
	s.redis.Set(ctx, cacheKey, data, 5*time.Minute)

	return listings, nil
}

func (s *Service) Update(ctx context.Context, l *Listing) error {
	return s.repo.Update(ctx, l)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
