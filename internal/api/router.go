package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/samirwankhede/lewly-pgpyewj/internal/config"
	kafkax "github.com/samirwankhede/lewly-pgpyewj/internal/kafka"
	"github.com/samirwankhede/lewly-pgpyewj/internal/middleware"
	redisx "github.com/samirwankhede/lewly-pgpyewj/internal/redis"
	"github.com/samirwankhede/lewly-pgpyewj/internal/service"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store"
)

// RegisterRoutes wires all HTTP routes.
func RegisterRoutes(r *gin.Engine, log *zap.Logger) {
	r.Use(MetricsMiddleware())
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":        "Evently",
			"description": "A scalable event booking platform with concurrency-safe ticketing, waitlists, and admin analytics.",
			"version":     "1.0.0",
			"docs":        "/swagger/index.html",
			"endpoints":   []string{"/v1/health", "/v1/events", "/v1/bookings", "/v1/waitlist", "/admin"},
		})
	})
	r.GET("/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	RegisterDocs(r)

	// global rate limit (demo)
	r.Use(middleware.RateLimit(50, 100))

	// minimal DI wiring for events listing/get
	cfg := config.Load()
	db, err := store.NewDB(context.Background(), cfg.PostgresURL, int32(cfg.MaxDBConnections))
	if err == nil {
		// When DB is unavailable, endpoints will still serve 500 gracefully.
		eventsRepo := store.NewEventsRepository(db)
		tokens := redisx.NewTokenBucket(cfg.RedisAddr)
		eventsSvc := service.NewEventsService(log, eventsRepo, tokens)
		NewEventsHandler(log, eventsSvc).Register(r)

		// Bookings wiring
		bookRepo := store.NewBookingsRepository(db)
		producer := kafkax.NewProducer([]string{cfg.KafkaBrokers}, "bookings")
		wlRepo := store.NewWaitlistRepository(db)
		bookSvc := service.NewBookingsService(log, bookRepo, tokens, producer, wlRepo)
		NewBookingsHandler(bookSvc).Register(r)

		// Waitlist endpoints
		NewWaitlistHandler(wlRepo).Register(r)

		// Users endpoints
		NewUsersHandler(bookRepo).Register(r)

		// Admin endpoints
		adminSvc := service.NewAdminService(log, eventsRepo, tokens)
		NewAdminHandler(adminSvc, cfg.JWTSigningSecret).Register(r)

	} else {
		log.Warn("db init failed", zap.Error(err))
	}
}

// RequestLogger is a simple zap logger middleware.
func RequestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		log.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
		)
	}
}
