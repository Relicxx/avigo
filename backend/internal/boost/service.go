package boost

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Boost(ctx context.Context, listingID, userID int64) (*Boost, error) {
	b := &Boost{ListingID: listingID, UserID: userID}
	if err := s.repo.Create(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}
