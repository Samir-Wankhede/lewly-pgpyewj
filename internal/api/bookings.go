package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/samirwankhede/lewly-pgpyewj/internal/service"
)

type BookingsHandler struct{ svc *service.BookingsService }

func NewBookingsHandler(svc *service.BookingsService) *BookingsHandler {
	return &BookingsHandler{svc: svc}
}

func (h *BookingsHandler) Register(r *gin.Engine) {
	r.POST("/v1/events/:id/book", h.book)
	r.POST("/v1/bookings/:id/cancel", h.cancel)
}

func (h *BookingsHandler) book(c *gin.Context) {
	eventID := c.Param("id")
	var req service.BookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, code, err := h.svc.Create(c, eventID, req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(code, resp)
}

func (h *BookingsHandler) cancel(c *gin.Context) {
	id := c.Param("id")
	resp, code, err := h.svc.Cancel(c, id)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}
	c.JSON(code, resp)
}
