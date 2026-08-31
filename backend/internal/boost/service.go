package boost

import (
	"context"
	"fmt"
	"time"

	"github.com/Relicxx/avigo/internal/apperr"
)

var (
	// ErrNotOwner возвращается при попытке поднять чужое объявление.
	ErrNotOwner = fmt.Errorf("%w: listing does not belong to user", apperr.ErrForbidden)
	// ErrAlreadyBoosted возвращается, если у объявления уже есть активный буст.
	ErrAlreadyBoosted = fmt.Errorf("%w: listing already has an active boost", apperr.ErrConflict)
)

type Service struct {
	repo     Repository
	duration time.Duration
}

// NewService создаёт сервис буста. duration — срок действия буста
// (задаётся конфигурацией, а не зашит в репозиторий).
func NewService(repo Repository, duration time.Duration) *Service {
	return &Service{repo: repo, duration: duration}
}

func (s *Service) Boost(ctx context.Context, listingID, userID int64) (*Boost, error) {
	ownerID, err := s.repo.ListingOwner(ctx, listingID)
	if err != nil {
		return nil, err
	}
	if ownerID != userID {
		return nil, ErrNotOwner
	}

	// Событие boost.created репозиторий пишет в outbox в той же
	// транзакции, что и сам буст; доставку в Kafka выполняет relay.
	b := &Boost{
		ListingID: listingID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.duration),
	}
	if err := s.repo.CreateActive(ctx, b); err != nil {
		return nil, err
	}

	return b, nil
}
