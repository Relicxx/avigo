package listing

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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

func fmtPrice(p *float64) string {
	if p == nil {
		return "-"
	}
	return strconv.FormatFloat(*p, 'f', 2, 64)
}

func (s *Service) List(ctx context.Context, f Filter) ([]*Listing, error) {
	cacheKey := fmt.Sprintf("listings:%s:%s:%s:%d:%d",
		f.Category, fmtPrice(f.MinPrice), fmtPrice(f.MaxPrice), f.Limit, f.Offset)

	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var listings []*Listing
		if err := json.Unmarshal([]byte(cached), &listings); err == nil {
			return listings, nil
		}
	}

	listings, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(listings)
	s.redis.Set(ctx, cacheKey, data, 5*time.Minute)

	return listings, nil
}

func (s *Service) Update(ctx context.Context, l *Listing) error {
	if err := s.repo.Update(ctx, l); err != nil {
		return err
	}
	s.invalidateListCache(ctx)
	return nil
}

func (s *Service) Delete(ctx context.Context, id, userID int64) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return err
	}
	s.invalidateListCache(ctx)
	return nil
}

// invalidateListCache сбрасывает закешированные листинги после изменения данных,
// чтобы клиенты не получали устаревший cache-aside результат.
func (s *Service) invalidateListCache(ctx context.Context) {
	keys, err := s.redis.Keys(ctx, "listings:*").Result()
	if err != nil || len(keys) == 0 {
		return
	}
	s.redis.Del(ctx, keys...)
}
