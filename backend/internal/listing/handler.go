package listing

import (
	"strconv"

	"github.com/Relicxx/avigo/pkg/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *gin.Context) {
	var req struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Category    string  `json:"category"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	userID := c.GetInt64("user_id")
	l := &Listing{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
	}
	err := h.service.Create(c.Request.Context(), l)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	c.JSON(201, l)
}

func (h *Handler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid listing ID"})
		return
	}

	l, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	c.JSON(200, l)
}

func (h *Handler) List(c *gin.Context) {
	category := c.Query("category")
	minPriceStr := c.Query("min_price")
	maxPriceStr := c.Query("max_price")
	var minPrice, maxPrice float64
	var err error

	if minPriceStr != "" {
		minPrice, err = strconv.ParseFloat(minPriceStr, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid min_price"})
			return
		}
	}

	if maxPriceStr != "" {
		maxPrice, err = strconv.ParseFloat(maxPriceStr, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid max_price"})
			return
		}
	}

	ls, err := h.service.List(c.Request.Context(), category, minPrice, maxPrice)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	c.JSON(200, ls)
}

func (h *Handler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid listing ID"})
		return
	}

	var req struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Category    string  `json:"category"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	userID := c.GetInt64("user_id")

	l := &Listing{
		ID:          id,
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
	}

	err = h.service.Update(c.Request.Context(), l)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	c.JSON(200, l)
}

func (h *Handler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid listing ID"})
		return
	}

	userID := c.GetInt64("user_id")
	err = h.service.Delete(c.Request.Context(), id, userID)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	c.JSON(204, nil)
}
