package boost

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

func (s *Service) Boost(ctx context.Context, listingID, userID int64) (*Boost, error) {
	b := &Boost{ListingID: listingID, UserID: userID}
	if err := s.repo.Create(ctx, b); err != nil {
		return nil, err
	}
	s.producer.Publish(ctx, "boost.created",
		[]byte(fmt.Sprintf("%d", b.ID)),
		[]byte(fmt.Sprintf(`{"id":%d,"listing_id":%d,"user_id":%d}`, b.ID, b.ListingID, b.UserID)),
	)
	return b, nil
}
