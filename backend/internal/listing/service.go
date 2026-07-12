package listing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

const listCacheTTL = 5 * time.Minute

// Publisher публикует доменные события.
type Publisher interface {
	Publish(ctx context.Context, topic string, key, value []byte) error
}

type Service struct {
	repo     Repository
	producer Publisher
	cache    Cache
}

func NewService(repo Repository, producer Publisher, cache Cache) *Service {
	return &Service{repo: repo, producer: producer, cache: cache}
}

func (s *Service) Create(ctx context.Context, l *Listing) error {
	if err := s.repo.Create(ctx, l); err != nil {
		return err
	}
	s.cache.Invalidate(ctx)
	if err := s.producer.Publish(ctx, "listing.created",
		[]byte(fmt.Sprintf("%d", l.ID)),
		[]byte(fmt.Sprintf(`{"id":%d,"user_id":%d}`, l.ID, l.UserID)),
	); err != nil {
		slog.Error("publish listing.created failed", "error", err, "listing_id", l.ID)
	}
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
	cacheKey := fmt.Sprintf("listings:v%d:%s:%s:%s:%d:%d",
		s.cache.Version(ctx),
		f.Category, fmtPrice(f.MinPrice), fmtPrice(f.MaxPrice), f.Limit, f.Offset)

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
