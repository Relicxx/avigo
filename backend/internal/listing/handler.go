package listing

import (
	"strconv"
	"strings"

	"github.com/Relicxx/avigo/pkg/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// createUpdateRequest — тело Create/Update. Максимальные длины соответствуют
// схеме БД: title VARCHAR(255), category VARCHAR(100).
type createUpdateRequest struct {
	Title       string  `json:"title" binding:"required,min=1,max=255"`
	Description string  `json:"description" binding:"max=5000"`
	Price       float64 `json:"price" binding:"gte=0,lt=100000000"`
	Category    string  `json:"category" binding:"max=100"`
}

func (h *Handler) Create(c *gin.Context) {
	var req createUpdateRequest

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

const (
	defaultListingsLimit = 20
	maxListingsLimit     = 100
	// maxSearchQueryLen ограничивает длину поисковой строки: этого достаточно
	// для осмысленного запроса и защищает БД от гигантских tsquery.
	maxSearchQueryLen = 100
)

// parseOptionalPrice различает «фильтр не задан» (nil) и «фильтр по 0».
func parseOptionalPrice(raw string) (*float64, bool) {
	if raw == "" {
		return nil, true
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < 0 {
		return nil, false
	}
	return &v, true
}

func (h *Handler) List(c *gin.Context) {
	f := Filter{
		Category: c.Query("category"),
		Query:    strings.TrimSpace(c.Query("q")),
	}
	if len(f.Query) > maxSearchQueryLen {
		c.JSON(400, gin.H{"error": "invalid q"})
		return
	}

	var ok bool
	if f.MinPrice, ok = parseOptionalPrice(c.Query("min_price")); !ok {
		c.JSON(400, gin.H{"error": "invalid min_price"})
		return
	}
	if f.MaxPrice, ok = parseOptionalPrice(c.Query("max_price")); !ok {
		c.JSON(400, gin.H{"error": "invalid max_price"})
		return
	}

	var err error
	f.Limit, f.Offset, err = httpx.Pagination(c, defaultListingsLimit, maxListingsLimit)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	ls, err := h.service.List(c.Request.Context(), f)
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

	var req createUpdateRequest

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
