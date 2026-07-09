package boost

import (
	"errors"
	"strconv"

	"github.com/Relicxx/avigo/pkg/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) Boost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid listing ID"})
		return
	}

	userID := c.GetInt64("user_id")
	boost, err := h.service.Boost(c.Request.Context(), id, userID)
	if errors.Is(err, ErrNotOwner) {
		c.JSON(403, gin.H{"error": "You can boost only your own listing"})
		return
	}
	if errors.Is(err, ErrAlreadyBoosted) {
		c.JSON(409, gin.H{"error": "Listing already has an active boost"})
		return
	}
	if err != nil {
		httpx.Error(c, err)
		return
	}

	c.JSON(201, boost)
}
