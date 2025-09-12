package users

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/samirwankhede/lewly-pgpyewj/internal/store/bookings"
)

type UsersHandler struct{ repo *bookings.BookingsRepository }

func NewUsersHandler(repo *bookings.BookingsRepository) *UsersHandler {
	return &UsersHandler{repo: repo}
}

func (h *UsersHandler) Register(r *gin.Engine) {
	r.GET("/v1/users/:id/bookings", h.listBookings)
}

func (h *UsersHandler) listBookings(c *gin.Context) {
	uid := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, err := h.repo.ListByUser(c.Request.Context(), uid, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bookings": items, "limit": limit, "offset": offset})
}
