package boost

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Relicxx/avigo/internal/apperr"
)

var (
	// ErrNotOwner возвращается при попытке поднять чужое объявление.
	ErrNotOwner = fmt.Errorf("%w: listing does not belong to user", apperr.ErrForbidden)
	// ErrAlreadyBoosted возвращается, если у объявления уже есть активный буст.
	ErrAlreadyBoosted = fmt.Errorf("%w: listing already has an active boost", apperr.ErrConflict)
)

// Publisher публикует доменные события.
type Publisher interface {
	Publish(ctx context.Context, topic string, key, value []byte) error
}

type Service struct {
	repo     Repository
	producer Publisher
	duration time.Duration
}

// NewService создаёт сервис буста. duration — срок действия буста
// (задаётся конфигурацией, а не зашит в репозиторий).
func NewService(repo Repository, producer Publisher, duration time.Duration) *Service {
	return &Service{repo: repo, producer: producer, duration: duration}
}

func (s *Service) Boost(ctx context.Context, listingID, userID int64) (*Boost, error) {
	ownerID, err := s.repo.ListingOwner(ctx, listingID)
	if err != nil {
		return nil, err
	}
	if ownerID != userID {
		return nil, ErrNotOwner
	}

	b := &Boost{
		ListingID: listingID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.duration),
	}
	if err := s.repo.CreateActive(ctx, b); err != nil {
		return nil, err
	}

	if err := s.producer.Publish(ctx, "boost.created",
		[]byte(fmt.Sprintf("%d", b.ID)),
		[]byte(fmt.Sprintf(`{"id":%d,"listing_id":%d,"user_id":%d}`, b.ID, b.ListingID, b.UserID)),
	); err != nil {
		slog.Error("publish boost.created failed", "error", err, "boost_id", b.ID)
	}

	return b, nil
}
