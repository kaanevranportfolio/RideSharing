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
	"github.com/rideshare-platform/services/trip-service/internal/repository"
	"github.com/rideshare-platform/services/trip-service/internal/service"
	"github.com/rideshare-platform/shared/logger"
)

func main() {
	// Create logger
	logr := logger.NewLogger("info", "development")

	// Initialize mock repositories (for now)
	eventStore := repository.NewMockEventStore()
	readModel := repository.NewMockReadModel()

	// Initialize advanced trip service
	tripService := service.NewAdvancedTripService(eventStore, readModel, *logr)

	// Setup router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "trip-service",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "1.0.0",
		})
	})

	// Trip endpoints
	v1 := router.Group("/api/v1")
	{
		v1.POST("/trips", func(c *gin.Context) {
			// Simple trip creation for testing
			c.JSON(http.StatusCreated, gin.H{
				"message":   "Trip service with advanced lifecycle management",
				"service":   "trip-service",
				"features":  []string{"event_sourcing", "cqrs", "state_machine"},
				"timestamp": time.Now().Format(time.RFC3339),
			})
		})

		v1.GET("/trips/:trip_id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Get trip endpoint (mock implementation)",
				"trip_id": c.Param("trip_id"),
			})
		})
		v1.GET("/trips/active", func(c *gin.Context) {
			trips, err := tripService.GetActiveTrips(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"active_trips": trips,
				"count":        len(trips),
			})
		})
	}

	// Setup HTTP server
	server := &http.Server{
		Addr:    ":8006",
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Trip service starting on port :8006")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down trip service...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Trip service shut down successfully")
}
