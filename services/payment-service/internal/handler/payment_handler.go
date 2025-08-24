package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rideshare-platform/services/payment-service/internal/service"
	"github.com/rideshare-platform/services/payment-service/internal/types"
	"github.com/rideshare-platform/shared/logger"
)

// PaymentHandler handles HTTP requests for payment operations
type PaymentHandler struct {
	paymentService *service.PaymentService
	logger         logger.Logger
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(paymentService *service.PaymentService, logger logger.Logger) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		logger:         logger,
	}
}

// ProcessPayment handles payment processing requests
func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	var req types.ProcessPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Basic validation
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Amount must be greater than zero",
		})
		return
	}

	if req.Currency == "" {
		req.Currency = "USD" // Default currency
	}

	response, err := h.paymentService.ProcessPayment(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to process payment", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Payment processing failed",
		})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// ProcessRefund handles refund requests
func (h *PaymentHandler) ProcessRefund(c *gin.Context) {
	var req types.RefundPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	response, err := h.paymentService.ProcessRefund(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to process refund", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Refund processing failed",
		})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// AddPaymentMethod handles adding new payment methods
func (h *PaymentHandler) AddPaymentMethod(c *gin.Context) {
	var req types.AddPaymentMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	response, err := h.paymentService.AddPaymentMethod(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to add payment method", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to add payment method",
		})
		return
	}

	if response.Success {
		c.JSON(http.StatusCreated, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// GetUserPaymentMethods retrieves payment methods for a user
func (h *PaymentHandler) GetUserPaymentMethods(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	methods, err := h.paymentService.GetUserPaymentMethods(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user payment methods", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve payment methods",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_methods": methods,
		"count":           len(methods),
	})
}

// GetPayment retrieves a specific payment
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	paymentID := c.Param("payment_id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Payment ID is required",
		})
		return
	}

	payment, err := h.paymentService.GetPayment(c.Request.Context(), paymentID)
	if err != nil {
		h.logger.Error("Failed to get payment", "error", err, "payment_id", paymentID)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Payment not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment": payment,
	})
}

// GetUserPayments retrieves payments for a user with pagination
func (h *PaymentHandler) GetUserPayments(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	payments, err := h.paymentService.GetUserPayments(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get user payments", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve user payments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payments": payments,
		"count":    len(payments),
		"limit":    limit,
		"offset":   offset,
	})
}

// GetTripPayments retrieves all payments for a trip
func (h *PaymentHandler) GetTripPayments(c *gin.Context) {
	tripID := c.Param("trip_id")
	if tripID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Trip ID is required",
		})
		return
	}

	payments, err := h.paymentService.GetTripPayments(c.Request.Context(), tripID)
	if err != nil {
		h.logger.Error("Failed to get trip payments", "error", err, "trip_id", tripID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve trip payments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trip_id":  tripID,
		"payments": payments,
		"count":    len(payments),
	})
}

// HealthCheck provides service health status
func (h *PaymentHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "payment-service",
		"version": "1.0.0",
		"timestamp": gin.H{
			"utc": gin.H{
				"time": c.Request.Header.Get("X-Request-Time"),
			},
		},
		"features": []string{
			"payment_processing",
			"fraud_detection",
			"multiple_payment_methods",
			"refund_processing",
			"transaction_logging",
		},
	})
}

// GetPaymentStats provides payment statistics (for admin/dashboard)
func (h *PaymentHandler) GetPaymentStats(c *gin.Context) {
	// This would typically aggregate payment data from the database
	// For now, return mock statistics
	c.JSON(http.StatusOK, gin.H{
		"daily_stats": gin.H{
			"total_payments":      1247,
			"successful_payments": 1189,
			"failed_payments":     58,
			"total_amount":        "₹2,847,392.50",
			"refunds_processed":   23,
			"fraud_blocked":       12,
		},
		"payment_methods": gin.H{
			"credit_card":    "45%",
			"digital_wallet": "32%",
			"debit_card":     "18%",
			"bank_transfer":  "4%",
			"cash":           "1%",
		},
		"processing_times": gin.H{
			"average_ms": 245,
			"p95_ms":     890,
			"p99_ms":     1420,
		},
	})
}
