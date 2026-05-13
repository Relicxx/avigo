package listing

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, l *Listing) error {
	return s.repo.Create(ctx, l)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Listing, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context,
	category string,
	minPrice,
	maxPrice float64,
) ([]*Listing, error) {
	return s.repo.List(ctx, category, minPrice, maxPrice)
}

func (s *Service) Update(ctx context.Context, l *Listing) error {
	return s.repo.Update(ctx, l)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
