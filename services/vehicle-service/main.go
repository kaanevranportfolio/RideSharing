package main

import (
	"log"
	"net/http"

	"net"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
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

	// Start HTTP server
	port := ":8080"
	go func() {
		log.Printf("Vehicle service starting on port %s", port)
		if err := r.Run(port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Start gRPC health server
	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	go func() {
		lis, err := net.Listen("tcp", ":50052")
		if err != nil {
			log.Fatalf("Failed to listen on gRPC port: %v", err)
		}
		log.Printf("gRPC server listening on port %s", "50052")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	select {} // Block forever
}
