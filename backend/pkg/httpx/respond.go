// Package httpx содержит общие HTTP-хелперы для хендлеров.
package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Relicxx/avigo/internal/apperr"
	"github.com/gin-gonic/gin"
)

// Error маппит доменную ошибку на HTTP-статус и пишет JSON-ответ.
// Неизвестные (внутренние) ошибки логируются и отдаются клиенту
// как generic 500 без деталей.
func Error(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apperr.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, apperr.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "conflict"})
	case errors.Is(err, apperr.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case errors.Is(err, apperr.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	case errors.Is(err, apperr.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
	default:
		slog.Error("internal error",
			"error", err,
			"method", c.Request.Method,
			"path", c.FullPath(),
			"request_id", c.GetString("request_id"),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
