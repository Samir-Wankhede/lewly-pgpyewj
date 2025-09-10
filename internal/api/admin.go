package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/samirwankhede/lewly-pgpyewj/internal/auth"
	"github.com/samirwankhede/lewly-pgpyewj/internal/service"
)

type AdminHandler struct {
	svc    *service.AdminService
	secret string
}

func NewAdminHandler(svc *service.AdminService, secret string) *AdminHandler {
	return &AdminHandler{svc: svc, secret: secret}
}

func (h *AdminHandler) Register(r *gin.Engine) {
	g := r.Group("/admin")
	g.Use(auth.Middleware(h.secret, true))
	g.POST("/events", h.createEvent)
}

func (h *AdminHandler) createEvent(c *gin.Context) {
	var in service.AdminEvent
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e, err := h.svc.CreateEvent(c, in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, e)
}
