package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsRecordsRouteTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Metrics())
	r.GET("/items/:id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	before := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/items/:id", "200"))

	req := httptest.NewRequest(http.MethodGet, "/items/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	after := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/items/:id", "200"))
	if after != before+1 {
		t.Fatalf("expected counter for route template to grow by 1, got %v -> %v", before, after)
	}
}

func TestMetricsRecordsUnmatchedRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Metrics())

	before := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "unmatched", "404"))

	req := httptest.NewRequest(http.MethodGet, "/no-such-route", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	after := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "unmatched", "404"))
	if after != before+1 {
		t.Fatalf("expected unmatched counter to grow by 1, got %v -> %v", before, after)
	}
}
