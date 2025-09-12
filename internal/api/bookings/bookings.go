package bookings

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	jwtMiddleware "github.com/samirwankhede/lewly-pgpyewj/internal/middleware"
	"github.com/samirwankhede/lewly-pgpyewj/internal/service/bookings"
)

type BookingsHandler struct {
	svc    *bookings.BookingsService
	secret string
}

func NewBookingsHandler(svc *bookings.BookingsService, secret string) *BookingsHandler {
	return &BookingsHandler{svc: svc, secret: secret}
}

func (h *BookingsHandler) Register(r *gin.Engine) {
	r.POST("/v1/events/:id/book", h.book)
	r.GET("/v1/bookings/:id/status", h.getStatus)
	r.GET("/v1/events/:id/seats", h.getAvailableSeats)
	r.POST("/v1/bookings/:id/cancel", h.cancel)

	// Protected routes
	protected := r.Group("/v1/bookings")
	protected.Use(jwtMiddleware.Middleware(h.secret, true))
	{
		protected.GET("/user/:user_id", h.listUserBookings)
	}
}

func (h *BookingsHandler) book(c *gin.Context) {
	eventID := c.Param("id")
	var req bookings.BookingRequest
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

func (h *BookingsHandler) getStatus(c *gin.Context) {
	id := c.Param("id")
	status, err := h.svc.GetBookingStatus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if status == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": status})
}

func (h *BookingsHandler) getAvailableSeats(c *gin.Context) {
	eventID := c.Param("id")
	seats, err := h.svc.GetAvailableSeats(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"seats": seats})
}

func (h *BookingsHandler) listUserBookings(c *gin.Context) {
	userID := c.GetString("uid")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	bookings, err := h.svc.ListUserBookings(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bookings": bookings, "limit": limit, "offset": offset})
}

func (h *BookingsHandler) cancel(c *gin.Context) {
	id := c.Param("id")
	resp, code, err := h.svc.Cancel(c.Request.Context(), id)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}
	c.JSON(code, resp)
}
