package chat

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Send(c *gin.Context) {
	var req struct {
		ListingID  int64  `json:"listing_id"`
		ReceiverID int64  `json:"receiver_id"`
		Body       string `json:"body"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	senderID := c.GetInt64("user_id")
	m := &Message{ListingID: req.ListingID, SenderID: senderID, ReceiverID: req.ReceiverID, Body: req.Body}
	if err := h.service.Send(c.Request.Context(), m); err != nil {
		c.JSON(500, gin.H{"error": "failed to send"})
		return
	}
	c.JSON(201, m)
}

func (h *Handler) GetByListing(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	msgs, err := h.service.GetByListing(c.Request.Context(), id)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get messages"})
		return
	}
	c.JSON(200, msgs)
}
