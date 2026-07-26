package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeRateLimitStore struct {
	counts map[string]int64
	err    error
}

func newFakeRateLimitStore() *fakeRateLimitStore {
	return &fakeRateLimitStore{counts: map[string]int64{}}
}

func (f *fakeRateLimitStore) Incr(_ context.Context, key string, _ time.Duration) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.counts[key]++
	return f.counts[key], nil
}

func rateLimitRouter(store RateLimitStore, limit int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login", RateLimit(store, "auth", limit, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func doLogin(r *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRateLimitAllowsUpToLimitThenRejects(t *testing.T) {
	r := rateLimitRouter(newFakeRateLimitStore(), 3)

	for i := 0; i < 3; i++ {
		if w := doLogin(r); w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	w := doLogin(r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 over limit, got %d", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got != "60" {
		t.Fatalf("expected Retry-After 60, got %q", got)
	}
}

func TestRateLimitIsPerIP(t *testing.T) {
	r := rateLimitRouter(newFakeRateLimitStore(), 1)

	if w := doLogin(r); w.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w.Code)
	}
	if w := doLogin(r); w.Code != http.StatusTooManyRequests {
		t.Fatalf("second request from same IP: expected 429, got %d", w.Code)
	}

	// Другой IP не задет чужим лимитом.
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("other IP: expected 200, got %d", w.Code)
	}
}

func TestRateLimitFailsOpenOnStoreError(t *testing.T) {
	store := &fakeRateLimitStore{err: errors.New("redis down")}
	r := rateLimitRouter(store, 1)

	for i := 0; i < 5; i++ {
		if w := doLogin(r); w.Code != http.StatusOK {
			t.Fatalf("request %d: expected fail-open 200, got %d", i+1, w.Code)
		}
	}
}
