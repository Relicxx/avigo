// Package health содержит probes готовности сервиса.
package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Check — именованная проверка зависимости.
type Check struct {
	Name  string
	Probe func(ctx context.Context) error
}

// Readiness возвращает handler готовности: 200, когда все зависимости
// отвечают, иначе 503 с именами упавших проверок. Детали ошибок
// не отдаются наружу — только в лог.
func Readiness(timeout time.Duration, checks ...Check) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		failed := make([]string, 0)
		for _, check := range checks {
			if err := check.Probe(ctx); err != nil {
				slog.Warn("readiness check failed", "check", check.Name, "error", err)
				failed = append(failed, check.Name)
			}
		}

		if len(failed) > 0 {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "failed": failed})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
