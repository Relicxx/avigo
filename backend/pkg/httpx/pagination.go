package httpx

import (
	"fmt"
	"strconv"

	"github.com/Relicxx/avigo/internal/apperr"
	"github.com/gin-gonic/gin"
)

// Pagination читает limit/offset из query-параметров.
// При отсутствии limit используется defaultLimit, значения выше maxLimit обрезаются.
func Pagination(c *gin.Context, defaultLimit, maxLimit int) (limit, offset int, err error) {
	limit = defaultLimit

	if raw := c.Query("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 {
			return 0, 0, fmt.Errorf("%w: invalid limit", apperr.ErrInvalidInput)
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	if raw := c.Query("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return 0, 0, fmt.Errorf("%w: invalid offset", apperr.ErrInvalidInput)
		}
	}

	return limit, offset, nil
}
