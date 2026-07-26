package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitStore инкрементирует счётчик окна и возвращает текущее значение.
// TTL окна выставляется при первом инкременте ключа.
type RateLimitStore interface {
	Incr(ctx context.Context, key string, window time.Duration) (int64, error)
}

// RateLimit ограничивает частоту запросов с одного IP фиксированным окном.
// name разделяет лимиты разных групп роутов. При ошибке хранилища
// middleware работает fail-open: запрос пропускается, ошибка логируется —
// недоступность Redis не должна ронять доступность API.
func RateLimit(store RateLimitStore, name string, limit int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("rl:%s:%s", name, c.ClientIP())

		n, err := store.Incr(c.Request.Context(), key, window)
		if err != nil {
			slog.Warn("rate limit store failed", "error", err, "key", key)
			c.Next()
			return
		}

		if n > limit {
			c.Header("Retry-After", strconv.Itoa(int(window/time.Second)))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}
