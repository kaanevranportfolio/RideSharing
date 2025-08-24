package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pricing-service/internal/config"
	"pricing-service/internal/handler"
	"pricing-service/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize services
	pricingService := service.NewAdvancedPricingService()

	// Initialize handlers
	pricingHandler := handler.NewPricingHandler(pricingService)

	// Setup router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "pricing-service",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "1.0.0",
		})
	})

	// Pricing endpoints
	v1 := router.Group("/api/v1")
	{
		v1.POST("/pricing/calculate", pricingHandler.CalculatePrice)
		v1.GET("/pricing/surge/:area", pricingHandler.GetSurgeMultiplier)
		v1.POST("/pricing/surge/update", pricingHandler.UpdateSurgeMultiplier)
		v1.POST("/pricing/discount/apply", pricingHandler.ApplyDiscount)
		v1.GET("/pricing/history/:trip_id", pricingHandler.GetPricingHistory)
		v1.GET("/pricing/analytics", pricingHandler.GetPricingAnalytics)
		v1.POST("/pricing/validate", pricingHandler.ValidatePrice)
	}

	// Setup HTTP server
	server := &http.Server{
		Addr:    cfg.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Pricing service starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down pricing service...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Pricing service shut down successfully")
}
