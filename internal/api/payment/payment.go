package payment

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/samirwankhede/lewly-pgpyewj/internal/service/payment"
)

type PaymentHandler struct {
	log *zap.Logger
	svc *payment.PaymentService
}

func NewPaymentHandler(log *zap.Logger, svc *payment.PaymentService) *PaymentHandler {
	return &PaymentHandler{log: log, svc: svc}
}

func (h *PaymentHandler) Register(r *gin.Engine) {
	// Public payment endpoints (called by payment providers)
	r.POST("/v1/payment/booking", h.processBookingPayment)
	r.POST("/v1/payment/refund", h.processRefund)

	// Admin endpoints for event cancellation refunds
	admin := r.Group("/admin")
	admin.Use(h.authMiddleware())
	{
		admin.POST("/events/:id/refund", h.processEventCancellationRefund)
	}
}

func (h *PaymentHandler) processBookingPayment(c *gin.Context) {
	var req payment.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.ProcessBookingPayment(c.Request.Context(), req)
	if err != nil {
		if err == payment.ErrBookingNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}
		if err == payment.ErrInvalidAmount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount"})
			return
		}
		if err == payment.ErrAlreadyPaid {
			c.JSON(http.StatusConflict, gin.H{"error": "Booking already paid"})
			return
		}
		h.log.Error("Payment processing failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if resp.Success {
		c.JSON(http.StatusOK, resp)
	} else {
		c.JSON(http.StatusPaymentRequired, resp)
	}
}

func (h *PaymentHandler) processRefund(c *gin.Context) {
	var req payment.RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.ProcessCancellationRefund(c.Request.Context(), req)
	if err != nil {
		if err == payment.ErrBookingNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}
		h.log.Error("Refund processing failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if resp.Success {
		c.JSON(http.StatusOK, resp)
	} else {
		c.JSON(http.StatusPaymentRequired, resp)
	}
}

func (h *PaymentHandler) processEventCancellationRefund(c *gin.Context) {
	eventID := c.Param("id")

	err := h.svc.ProcessEventCancellationRefund(c.Request.Context(), eventID)
	if err != nil {
		h.log.Error("Event cancellation refund failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event cancellation refunds processed successfully"})
}

func (h *PaymentHandler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simple auth check - in real implementation, use proper JWT middleware
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// For now, just continue
		// In real implementation, parse JWT token and verify admin privileges
		c.Next()
	}
}
