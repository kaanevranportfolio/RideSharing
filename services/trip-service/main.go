package main

import (
	"log"
	"net"

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

	// Start gRPC server
	listener, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Failed to listen on port 50053: %v", err)
	}

	logr.Info("Trip Service gRPC server listening on port 50053")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
