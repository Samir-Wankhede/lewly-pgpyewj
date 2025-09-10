package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samirwankhede/lewly-pgpyewj/internal/store"
)

type AdminAnalyticsHandler struct{ repo *store.AnalyticsRepository }

func NewAdminAnalyticsHandler(repo *store.AnalyticsRepository) *AdminAnalyticsHandler {
	return &AdminAnalyticsHandler{repo: repo}
}

func (h *AdminAnalyticsHandler) Register(r *gin.RouterGroup) {
	r.GET("/analytics", h.summary)
}

func (h *AdminAnalyticsHandler) summary(c *gin.Context) {
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
