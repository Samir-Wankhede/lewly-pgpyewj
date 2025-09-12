package waitlist

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/samirwankhede/lewly-pgpyewj/internal/store/waitlist"
)

type WaitlistHandler struct{ repo *waitlist.WaitlistRepository }

func NewWaitlistHandler(repo *waitlist.WaitlistRepository) *WaitlistHandler {
	return &WaitlistHandler{repo: repo}
}

func (h *WaitlistHandler) Register(r *gin.Engine) {
	r.POST("/v1/waitlist/:event_id/join", h.join)
	r.POST("/v1/waitlist/:event_id/:user_id/optout", h.optout)
	r.GET("/v1/waitlist/:event_id/count", h.getCount)
	r.GET("/v1/waitlist/:event_id", h.list)
}

func (h *WaitlistHandler) join(c *gin.Context) {
	eventID := c.Param("event_id")
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	pos, err := h.repo.Add(c.Request.Context(), eventID, body.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"position": pos})
}

func (h *WaitlistHandler) optout(c *gin.Context) {
	eventID := c.Param("event_id")
	userID := c.Param("user_id")
	if err := h.repo.OptOut(c.Request.Context(), eventID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"opted_out": true})
}

func (h *WaitlistHandler) getCount(c *gin.Context) {
	eventID := c.Param("event_id")
	count, err := h.repo.Count(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

func (h *WaitlistHandler) list(c *gin.Context) {
	eventID := c.Param("event_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	entries, err := h.repo.ListByEvent(c.Request.Context(), eventID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"waitlist": entries, "limit": limit, "offset": offset})
}
