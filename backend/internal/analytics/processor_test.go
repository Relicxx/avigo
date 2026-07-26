package analytics

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeCounter struct {
	counts map[string]int
	err    error
}

func newFakeCounter() *fakeCounter {
	return &fakeCounter{counts: map[string]int{}}
}

func (f *fakeCounter) Inc(_ context.Context, key string) error {
	if f.err != nil {
		return f.err
	}
	f.counts[key]++
	return nil
}

func newTestProcessor(c Counter) *Processor {
	p := NewProcessor(c)
	p.now = func() time.Time {
		return time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	}
	return p
}

func TestHandleIncrementsDailyCounter(t *testing.T) {
	tests := map[string]struct {
		topic   string
		payload string
		wantKey string
	}{
		"listing created": {
			topic:   "listing.created",
			payload: `{"id":7,"user_id":3}`,
			wantKey: "stats:listing.created:2026-07-27",
		},
		"boost created": {
			topic:   "boost.created",
			payload: `{"id":1,"listing_id":7,"user_id":3}`,
			wantKey: "stats:boost.created:2026-07-27",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			counter := newFakeCounter()
			p := newTestProcessor(counter)

			if err := p.Handle(context.Background(), tc.topic, []byte(tc.payload)); err != nil {
				t.Fatalf("handle: %v", err)
			}
			if got := counter.counts[tc.wantKey]; got != 1 {
				t.Fatalf("expected %s = 1, got %d (all: %v)", tc.wantKey, got, counter.counts)
			}
		})
	}
}

func TestHandleRejectsBadPayload(t *testing.T) {
	tests := map[string]string{
		"invalid json":    `{not-json`,
		"missing id":      `{"user_id":3}`,
		"missing user_id": `{"id":7}`,
		"empty object":    `{}`,
	}

	for name, payload := range tests {
		t.Run(name, func(t *testing.T) {
			counter := newFakeCounter()
			p := newTestProcessor(counter)

			if err := p.Handle(context.Background(), "listing.created", []byte(payload)); err == nil {
				t.Fatal("expected error for bad payload")
			}
			if len(counter.counts) != 0 {
				t.Fatalf("counter must not be touched, got %v", counter.counts)
			}
		})
	}
}

func TestHandlePropagatesCounterError(t *testing.T) {
	wantErr := errors.New("redis down")
	p := newTestProcessor(&fakeCounter{err: wantErr})

	err := p.Handle(context.Background(), "listing.created", []byte(`{"id":7,"user_id":3}`))
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected counter error to propagate, got %v", err)
	}
}
