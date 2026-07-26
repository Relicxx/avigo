package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func readyRouter(checks ...Check) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/readyz", Readiness(time.Second, checks...))
	return r
}

func doReady(r *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func ok(_ context.Context) error   { return nil }
func fail(_ context.Context) error { return errors.New("connection refused") }

func TestReadinessAllHealthy(t *testing.T) {
	r := readyRouter(
		Check{Name: "postgres", Probe: ok},
		Check{Name: "redis", Probe: ok},
	)
	if w := doReady(r); w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestReadinessReportsFailedChecks(t *testing.T) {
	r := readyRouter(
		Check{Name: "postgres", Probe: ok},
		Check{Name: "redis", Probe: fail},
	)

	w := doReady(r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}

	var resp struct {
		Status string   `json:"status"`
		Failed []string `json:"failed"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Failed) != 1 || resp.Failed[0] != "redis" {
		t.Fatalf("expected failed=[redis], got %v", resp.Failed)
	}
	// Детали ошибки (адреса, DSN) не должны утекать в ответ.
	if body := w.Body.String(); strings.Contains(body, "connection refused") {
		t.Fatalf("error details must not leak to response: %s", body)
	}
}
