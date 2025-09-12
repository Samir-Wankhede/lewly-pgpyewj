package admin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	jwtMiddleware "github.com/samirwankhede/lewly-pgpyewj/internal/middleware"
	"github.com/samirwankhede/lewly-pgpyewj/internal/service/admin"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store"
)

type AdminHandler struct {
	repo   *store.AnalyticsRepository
	svc    *admin.AdminService
	secret string
}

func NewAdminHandler(svc *admin.AdminService, secret string) *AdminHandler {
	return &AdminHandler{svc: svc, secret: secret}
}

func (h *AdminHandler) Register(r *gin.Engine) {
	g := r.Group("/admin")
	g.Use(jwtMiddleware.Middleware(h.secret, true))
	{
		g.POST("/events", h.createEvent)
		g.PUT("/events/:id", h.updateEvent)
		g.POST("/events/:id/cancel", h.cancelEvent)
		g.GET("/analytics", h.summary)
		g.POST("/users/:id/admin", h.createAdmin)
		g.DELETE("/users/:id/admin", h.removeAdmin)
		g.DELETE("/users/:id", h.removeUser)
	}
}

func (h *AdminHandler) createEvent(c *gin.Context) {
	var in admin.AdminEvent
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
	a, err := h.svc.GetSummary(c.Request.Context(), from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *AdminHandler) updateEvent(c *gin.Context) {
	eventID := c.Param("id")
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.svc.UpdateEvent(c.Request.Context(), eventID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Event updated successfully"})
}

func (h *AdminHandler) cancelEvent(c *gin.Context) {
	eventID := c.Param("id")
	err := h.svc.CancelEvent(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Event cancelled successfully"})
}

func (h *AdminHandler) createAdmin(c *gin.Context) {
	userID := c.Param("id")
	err := h.svc.CreateAdminFromUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User promoted to admin successfully"})
}

func (h *AdminHandler) removeAdmin(c *gin.Context) {
	userID := c.Param("id")
	err := h.svc.RemoveAdmin(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Admin privileges removed successfully"})
}

func (h *AdminHandler) removeUser(c *gin.Context) {
	userID := c.Param("id")
	err := h.svc.RemoveUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User removed successfully"})
}
