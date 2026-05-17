package chat

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Send(ctx context.Context, m *Message) error {
	return s.repo.Send(ctx, m)
}

func (s *Service) GetByListing(ctx context.Context, listingID int64) ([]*Message, error) {
	return s.repo.GetByListing(ctx, listingID)
}
