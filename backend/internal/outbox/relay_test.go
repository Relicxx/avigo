package outbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeStore выдаёт заранее подготовленные сообщения пачками,
// имитируя claim-and-mark логику PgStore.
type fakeStore struct {
	mu      sync.Mutex
	pending []Message
	fail    error
	calls   int
}

func (s *fakeStore) ProcessPending(ctx context.Context, limit int, publish PublishFunc) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++
	if s.fail != nil {
		return 0, s.fail
	}

	batch := s.pending
	if len(batch) > limit {
		batch = batch[:limit]
	}

	published := 0
	for _, m := range batch {
		if err := publish(ctx, m); err != nil {
			s.pending = s.pending[published:]
			return published, err
		}
		published++
	}
	s.pending = s.pending[published:]
	return published, nil
}

type fakePublisher struct {
	mu     sync.Mutex
	topics []string
	keys   []string
	failOn string // ключ, на котором публикация падает
}

func (p *fakePublisher) Publish(_ context.Context, topic string, key, _ []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.failOn != "" && string(key) == p.failOn {
		return errors.New("broker unavailable")
	}
	p.topics = append(p.topics, topic)
	p.keys = append(p.keys, string(key))
	return nil
}

func (p *fakePublisher) publishedCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.topics)
}

func seed(n int) []Message {
	msgs := make([]Message, 0, n)
	for i := 1; i <= n; i++ {
		msgs = append(msgs, Message{ID: int64(i), Topic: "listing.created", Key: "1", Payload: []byte(`{}`)})
	}
	return msgs
}

func TestRelayDrainsBacklogAcrossBatches(t *testing.T) {
	store := &fakeStore{pending: seed(25)}
	pub := &fakePublisher{}
	relay := NewRelay(store, pub, time.Hour, 10)

	relay.drain(context.Background())

	if got := pub.publishedCount(); got != 25 {
		t.Fatalf("expected 25 published messages, got %d", got)
	}
	if len(store.pending) != 0 {
		t.Fatalf("expected empty outbox, %d messages left", len(store.pending))
	}
	// 25 сообщений при пачке в 10: две полные пачки заставляют опросить ещё
	// раз, третья (неполная) останавливает цикл.
	if store.calls != 3 {
		t.Fatalf("expected 3 ProcessPending calls, got %d", store.calls)
	}
}

func TestRelayStopsBatchOnPublishError(t *testing.T) {
	store := &fakeStore{pending: []Message{
		{ID: 1, Topic: "t", Key: "ok-1", Payload: []byte(`{}`)},
		{ID: 2, Topic: "t", Key: "bad", Payload: []byte(`{}`)},
		{ID: 3, Topic: "t", Key: "ok-2", Payload: []byte(`{}`)},
	}}
	pub := &fakePublisher{failOn: "bad"}
	relay := NewRelay(store, pub, time.Hour, 10)

	relay.drain(context.Background())

	// Публикуется только сообщение до сбоя; упавшее и всё после него
	// остаются в outbox до следующего опроса.
	if got := pub.publishedCount(); got != 1 {
		t.Fatalf("expected 1 published message, got %d", got)
	}
	if len(store.pending) != 2 {
		t.Fatalf("expected 2 pending messages after failure, got %d", len(store.pending))
	}
}

func TestRelayStoreErrorDoesNotLoop(t *testing.T) {
	store := &fakeStore{fail: errors.New("db down")}
	relay := NewRelay(store, &fakePublisher{}, time.Hour, 10)

	relay.drain(context.Background())

	if store.calls != 1 {
		t.Fatalf("expected a single call on store error, got %d", store.calls)
	}
}

func TestRelayRunStopsOnContextCancel(t *testing.T) {
	store := &fakeStore{pending: seed(3)}
	pub := &fakePublisher{}
	relay := NewRelay(store, pub, 5*time.Millisecond, 10)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		relay.Run(ctx)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for pub.publishedCount() < 3 {
		select {
		case <-deadline:
			t.Fatal("relay did not publish seeded messages in time")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("relay did not stop after context cancel")
	}
}
