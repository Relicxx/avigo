package listing

import "github.com/gin-gonic/gin"

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *gin.Context) {}
func (h *Handler) GetByID(c *gin.Context) {}
func (h *Handler) List(c *gin.Context) {}
func (h *Handler) Update(c *gin.Context) {}
func (h *Handler) Delete(c *gin.Context) {}
