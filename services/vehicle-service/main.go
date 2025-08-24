package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create Gin router
	r := gin.Default()

	// Basic health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "vehicle-service",
		})
	})

	// Basic vehicles endpoint
	r.GET("/vehicles", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"vehicles": []gin.H{},
			"message":  "Vehicle service is running",
		})
	})

	// Start server
	port := ":8080"
	log.Printf("Vehicle service starting on port %s", port)
	if err := r.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
