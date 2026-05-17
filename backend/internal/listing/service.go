package listing

import (
	"context"
	"fmt"

	"github.com/Relicxx/avigo/internal/kafka"
)

type Service struct {
	repo     Repository
	producer *kafka.Producer
}

func NewService(repo Repository, producer *kafka.Producer) *Service {
	return &Service{repo: repo, producer: producer}
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
	return s.repo.List(ctx, category, minPrice, maxPrice)
}

func (s *Service) Update(ctx context.Context, l *Listing) error {
	return s.repo.Update(ctx, l)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
