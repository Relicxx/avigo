package listing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Relicxx/avigo/internal/apperr"
)

type mockRepo struct {
	createErr  error
	updateErr  error
	deleteErr  error
	listed     []*Listing
	listCalls  int
	lastFilter Filter
}

func (m *mockRepo) Create(_ context.Context, l *Listing) error {
	if m.createErr != nil {
		return m.createErr
	}
	l.ID = 1
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, _ int64) (*Listing, error) {
	return nil, apperr.ErrNotFound
}

func (m *mockRepo) List(_ context.Context, f Filter) ([]*Listing, error) {
	m.listCalls++
	m.lastFilter = f
	return m.listed, nil
}

func (m *mockRepo) Update(_ context.Context, _ *Listing) error { return m.updateErr }

func (m *mockRepo) Delete(_ context.Context, _, _ int64) error { return m.deleteErr }

type stubCache struct {
	store         map[string][]byte
	invalidations int
}

func newStubCache() *stubCache { return &stubCache{store: map[string][]byte{}} }

func (c *stubCache) Get(_ context.Context, key string) ([]byte, bool) {
	v, ok := c.store[key]
	return v, ok
}

func (c *stubCache) Set(_ context.Context, key string, value []byte, _ time.Duration) {
	c.store[key] = value
}

func (c *stubCache) Version(_ context.Context) int64 { return 0 }

func (c *stubCache) Invalidate(_ context.Context) { c.invalidations++ }

func TestUpdateForeignListingReturnsNotFound(t *testing.T) {
	repo := &mockRepo{updateErr: apperr.ErrNotFound}
	cache := newStubCache()
	s := NewService(repo, cache)

	err := s.Update(context.Background(), &Listing{ID: 1, UserID: 99})
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if cache.invalidations != 0 {
		t.Fatal("cache must not be invalidated on failed update")
	}
}

func TestDeleteForeignListingReturnsNotFound(t *testing.T) {
	repo := &mockRepo{deleteErr: apperr.ErrNotFound}
	cache := newStubCache()
	s := NewService(repo, cache)

	err := s.Delete(context.Background(), 1, 99)
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if cache.invalidations != 0 {
		t.Fatal("cache must not be invalidated on failed delete")
	}
}

func TestCreateInvalidatesCache(t *testing.T) {
	repo := &mockRepo{}
	cache := newStubCache()
	s := NewService(repo, cache)

	if err := s.Create(context.Background(), &Listing{UserID: 1, Title: "test"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if cache.invalidations != 1 {
		t.Fatalf("expected 1 cache invalidation, got %d", cache.invalidations)
	}
}

func TestListCachesResult(t *testing.T) {
	repo := &mockRepo{listed: []*Listing{{ID: 1, Title: "cached"}}}
	cache := newStubCache()
	s := NewService(repo, cache)
	ctx := context.Background()

	first, err := s.List(ctx, Filter{Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	second, err := s.List(ctx, Filter{Limit: 20})
	if err != nil {
		t.Fatalf("list from cache: %v", err)
	}
	if repo.listCalls != 1 {
		t.Fatalf("expected 1 repo call, got %d", repo.listCalls)
	}
	if len(first) != 1 || len(second) != 1 || second[0].Title != "cached" {
		t.Fatalf("unexpected results: %v / %v", first, second)
	}
}

func TestListPassesSearchQueryToRepo(t *testing.T) {
	repo := &mockRepo{}
	s := NewService(repo, newStubCache())

	if _, err := s.List(context.Background(), Filter{Limit: 20, Query: "iphone 13"}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.lastFilter.Query != "iphone 13" {
		t.Fatalf("expected query to reach repo, got %q", repo.lastFilter.Query)
	}
}

func TestListCacheKeyIncludesSearchQuery(t *testing.T) {
	repo := &mockRepo{listed: []*Listing{{ID: 1}}}
	s := NewService(repo, newStubCache())
	ctx := context.Background()

	// Разные q не должны попадать в один кэш-ключ.
	if _, err := s.List(ctx, Filter{Limit: 20, Query: "iphone"}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if _, err := s.List(ctx, Filter{Limit: 20, Query: "macbook"}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.listCalls != 2 {
		t.Fatalf("expected 2 repo calls for different queries, got %d", repo.listCalls)
	}

	// Повтор того же q обслуживается из кэша.
	if _, err := s.List(ctx, Filter{Limit: 20, Query: "iphone"}); err != nil {
		t.Fatalf("list from cache: %v", err)
	}
	if repo.listCalls != 2 {
		t.Fatalf("expected repeated query to hit cache, got %d repo calls", repo.listCalls)
	}
}
