package listing

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

const listCacheTTL = 5 * time.Minute

type Service struct {
	repo  Repository
	cache Cache
}

func NewService(repo Repository, cache Cache) *Service {
	return &Service{repo: repo, cache: cache}
}

// Create создаёт объявление; событие listing.created репозиторий пишет
// в outbox в той же транзакции, доставку в Kafka выполняет relay.
func (s *Service) Create(ctx context.Context, l *Listing) error {
	if err := s.repo.Create(ctx, l); err != nil {
		return err
	}
	s.cache.Invalidate(ctx)
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
	// Ключ включает все параметры выборки, в том числе поисковую строку q:
	// иначе результаты разных запросов перепутались бы в кэше.
	cacheKey := fmt.Sprintf("listings:v%d:%s:%s:%s:%d:%d:%s",
		s.cache.Version(ctx),
		f.Category, fmtPrice(f.MinPrice), fmtPrice(f.MaxPrice), f.Limit, f.Offset, f.Query)

	if cached, ok := s.cache.Get(ctx, cacheKey); ok {
		var listings []*Listing
		if err := json.Unmarshal(cached, &listings); err == nil {
			return listings, nil
		}
	}

	listings, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(listings); err == nil {
		s.cache.Set(ctx, cacheKey, data, listCacheTTL)
	}

	return listings, nil
}

func (s *Service) Update(ctx context.Context, l *Listing) error {
	if err := s.repo.Update(ctx, l); err != nil {
		return err
	}
	s.cache.Invalidate(ctx)
	return nil
}

func (s *Service) Delete(ctx context.Context, id, userID int64) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return err
	}
	s.cache.Invalidate(ctx)
	return nil
}
