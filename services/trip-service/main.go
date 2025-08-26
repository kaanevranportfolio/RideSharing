package main

import (
	"log"
	"net"
	"net/http"

	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"google.golang.org/grpc"

	"github.com/rideshare-platform/services/trip-service/internal/handler"
	"github.com/rideshare-platform/services/trip-service/internal/service"
	"github.com/rideshare-platform/shared/logger"
	trippb "github.com/rideshare-platform/shared/proto/trip"
)

func main() {
	// Create logger
	logr := logger.NewLogger("info", "development")
	logr.Info("Starting Trip Service...")

	// Create service
	tripService := service.NewBasicTripService(logr)

	// Create gRPC handler
	grpcHandler := handler.NewGRPCTripHandler(tripService, logr)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	trippb.RegisterTripServiceServer(grpcServer, grpcHandler)
	// Register gRPC health service
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	// Start gRPC server
	listener, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Failed to listen on port 50053: %v", err)
	}

	logr.Info("Trip Service gRPC server listening on port 50053")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
	// Minimal HTTP health endpoint
	go func() {
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "healthy", "service": "trip-service"}`))
		})
		if err := http.ListenAndServe(":8085", nil); err != nil {
			log.Fatalf("Failed to start HTTP health server: %v", err)
		}
	}()

}
