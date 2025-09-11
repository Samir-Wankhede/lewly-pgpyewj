package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samirwankhede/lewly-pgpyewj/internal/auth"
	"github.com/samirwankhede/lewly-pgpyewj/internal/service"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store"
)

type AdminHandler struct {
	repo   *store.AnalyticsRepository
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
	r.GET("/analytics", h.summary)
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

func (h *AdminHandler) summary(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	var from, to time.Time
	var err error
	if fromStr == "" {
		from = time.Now().Add(-24 * time.Hour)
	} else {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad from"})
			return
		}
	}
	if toStr == "" {
		to = time.Now()
	} else {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad to"})
			return
		}
	}
	a, err := h.repo.Summary(c, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, a)
}
