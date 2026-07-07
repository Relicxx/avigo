package chat

import (
	"strconv"

	"github.com/Relicxx/avigo/pkg/httpx"
	"github.com/gin-gonic/gin"
)

const (
	defaultMessagesLimit = 50
	maxMessagesLimit     = 200
)

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Send(c *gin.Context) {
	var req struct {
		ListingID int64 `json:"listing_id" binding:"required"`
		// ReceiverID учитывается только когда отправитель — владелец объявления.
		ReceiverID int64  `json:"receiver_id"`
		Body       string `json:"body" binding:"required,max=2000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	senderID := c.GetInt64("user_id")
	m, err := h.service.Send(c.Request.Context(), senderID, req.ListingID, req.ReceiverID, req.Body)
	if err != nil {
		httpx.Error(c, err)
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

	limit, offset, err := httpx.Pagination(c, defaultMessagesLimit, maxMessagesLimit)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	userID := c.GetInt64("user_id")
	msgs, err := h.service.GetForUser(c.Request.Context(), id, userID, limit, offset)
	if err != nil {
		httpx.Error(c, err)
		return
	}
	c.JSON(200, msgs)
}
