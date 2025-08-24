package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rideshare-platform/services/payment-service/internal/repository"
	"github.com/rideshare-platform/services/payment-service/internal/service"
	"github.com/rideshare-platform/services/payment-service/internal/types"
	"github.com/rideshare-platform/shared/logger"
)

func main() {
	// Create logger
	logr := logger.NewLogger("info", "development")

	// Initialize mock repositories
	paymentRepo := repository.NewMockPaymentRepository()
	paymentMethodRepo := repository.NewMockPaymentMethodRepository()
	refundRepo := repository.NewMockRefundRepository()

	// Initialize fraud detection service
	fraudService := service.NewSimpleFraudDetectionService(*logr)

	// Initialize payment service
	paymentService := service.NewPaymentService(
		paymentRepo,
		paymentMethodRepo,
		refundRepo,
		fraudService,
		*logr,
	)

	// Setup router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "payment-service",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "1.0.0",
			"features": []string{
				"payment_processing",
				"fraud_detection",
				"multiple_payment_methods",
				"refund_processing",
				"transaction_logging",
			},
		})
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Payment processing
		v1.POST("/payments", func(c *gin.Context) {
			var req types.ProcessPaymentRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid request body",
					"details": err.Error(),
				})
				return
			}

			response, err := paymentService.ProcessPayment(c.Request.Context(), &req)
			if err != nil {
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
		})

		// Refund processing
		v1.POST("/refunds", func(c *gin.Context) {
			var req types.RefundPaymentRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid request body",
					"details": err.Error(),
				})
				return
			}

			response, err := paymentService.ProcessRefund(c.Request.Context(), &req)
			if err != nil {
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
		})

		// Payment methods
		v1.POST("/payment-methods", func(c *gin.Context) {
			var req types.AddPaymentMethodRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid request body",
					"details": err.Error(),
				})
				return
			}

			response, err := paymentService.AddPaymentMethod(c.Request.Context(), &req)
			if err != nil {
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
		})

		// Get user payment methods
		v1.GET("/users/:user_id/payment-methods", func(c *gin.Context) {
			userID := c.Param("user_id")
			methods, err := paymentService.GetUserPaymentMethods(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to retrieve payment methods",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"payment_methods": methods,
				"count":           len(methods),
			})
		})

		// Get payment
		v1.GET("/payments/:payment_id", func(c *gin.Context) {
			paymentID := c.Param("payment_id")
			payment, err := paymentService.GetPayment(c.Request.Context(), paymentID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Payment not found",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"payment": payment,
			})
		})

		// Get payment statistics (mock)
		v1.GET("/stats", func(c *gin.Context) {
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
		})
	}

	// Setup HTTP server
	server := &http.Server{
		Addr:    ":8005",
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Payment service starting on port :8005")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down payment service...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Payment service shut down successfully")
}
