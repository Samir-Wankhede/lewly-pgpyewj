package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/samirwankhede/lewly-pgpyewj/internal/store"
)

type WaitlistHandler struct{ repo *store.WaitlistRepository }

func NewWaitlistHandler(repo *store.WaitlistRepository) *WaitlistHandler {
	return &WaitlistHandler{repo: repo}
}

func (h *WaitlistHandler) Register(r *gin.Engine) {
	r.POST("/v1/waitlist/:event_id/join", h.join)
	r.POST("/v1/waitlist/:event_id/:user_id/optout", h.optout)
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
	pos, err := h.repo.Add(c, eventID, body.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"position": pos})
}

func (h *WaitlistHandler) optout(c *gin.Context) {
	eventID := c.Param("event_id")
	userID := c.Param("user_id")
	if err := h.repo.OptOut(c, eventID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"opted_out": true})
}
