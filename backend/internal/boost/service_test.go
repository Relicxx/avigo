package boost

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Relicxx/avigo/internal/apperr"
)

type fakeBoostRepo struct {
	owners    map[int64]int64 // listingID -> ownerID
	createErr error
	created   []*Boost
}

func (f *fakeBoostRepo) CreateActive(_ context.Context, b *Boost) error {
	if f.createErr != nil {
		return f.createErr
	}
	b.ID = int64(len(f.created) + 1)
	b.CreatedAt = time.Now()
	f.created = append(f.created, b)
	return nil
}

func (f *fakeBoostRepo) ListingOwner(_ context.Context, listingID int64) (int64, error) {
	owner, ok := f.owners[listingID]
	if !ok {
		return 0, apperr.ErrNotFound
	}
	return owner, nil
}

func TestBoostForeignListingForbidden(t *testing.T) {
	repo := &fakeBoostRepo{owners: map[int64]int64{10: 2}}
	s := NewService(repo, time.Hour)

	_, err := s.Boost(context.Background(), 10, 1)
	if !errors.Is(err, ErrNotOwner) {
		t.Fatalf("expected ErrNotOwner, got %v", err)
	}
	if !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("ErrNotOwner must map to ErrForbidden, got %v", err)
	}
	if len(repo.created) != 0 {
		t.Fatal("boost must not be created for a foreign listing")
	}
}

func TestBoostMissingListingNotFound(t *testing.T) {
	repo := &fakeBoostRepo{owners: map[int64]int64{}}
	s := NewService(repo, time.Hour)

	_, err := s.Boost(context.Background(), 404, 1)
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestBoostAlreadyBoostedConflict(t *testing.T) {
	repo := &fakeBoostRepo{owners: map[int64]int64{10: 1}, createErr: ErrAlreadyBoosted}
	s := NewService(repo, time.Hour)

	_, err := s.Boost(context.Background(), 10, 1)
	if !errors.Is(err, ErrAlreadyBoosted) {
		t.Fatalf("expected ErrAlreadyBoosted, got %v", err)
	}
	if !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("ErrAlreadyBoosted must map to ErrConflict, got %v", err)
	}
}

func TestBoostOwnListingUsesConfiguredDuration(t *testing.T) {
	repo := &fakeBoostRepo{owners: map[int64]int64{10: 1}}
	duration := 2 * time.Hour
	s := NewService(repo, duration)

	before := time.Now()
	b, err := s.Boost(context.Background(), 10, 1)
	if err != nil {
		t.Fatalf("boost: %v", err)
	}

	want := before.Add(duration)
	if b.ExpiresAt.Before(want.Add(-time.Minute)) || b.ExpiresAt.After(want.Add(time.Minute)) {
		t.Fatalf("expires_at %v not within expected window around %v", b.ExpiresAt, want)
	}
}
