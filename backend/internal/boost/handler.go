package boost

import (
	"strconv"

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
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to boost listing"})
		return
	}

	c.JSON(201, boost)
}
