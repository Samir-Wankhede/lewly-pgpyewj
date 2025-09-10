package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/samirwankhede/lewly-pgpyewj/internal/service"
)

type EventsHandler struct {
	log *zap.Logger
	svc *service.EventsService
}

func NewEventsHandler(log *zap.Logger, svc *service.EventsService) *EventsHandler {
	return &EventsHandler{log: log, svc: svc}
}

func (h *EventsHandler) Register(r *gin.Engine) {
	r.GET("/v1/events", h.list)
	r.GET("/v1/events/:id", h.get)
}

func (h *EventsHandler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	q := c.Query("q")
	var fromPtr, toPtr *time.Time
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			fromPtr = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			toPtr = &t
		}
	}
	items, err := h.svc.List(c, limit, offset, q, fromPtr, toPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": items, "limit": limit, "offset": offset})
}

func (h *EventsHandler) get(c *gin.Context) {
	id := c.Param("id")
	e, rem, err := h.svc.Get(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"event": e, "tokens_remaining": rem})
}
