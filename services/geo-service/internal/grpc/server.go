package grpc

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/rideshare-platform/services/geo-service/internal/service"
	"github.com/rideshare-platform/shared/logger"
	geopb "github.com/rideshare-platform/shared/proto/geo"
)

// Server represents the gRPC server for geospatial service
type Server struct {
	geopb.UnimplementedGeospatialServiceServer
	geoService service.GeospatialService
	logger     logger.Logger
	grpcServer *grpc.Server
}

// NewServer creates a new gRPC server instance
func NewServer(geoService service.GeospatialService, logger logger.Logger) *Server {
	return &Server{
		geoService: geoService,
		logger:     logger,
	}
}

// Start starts the gRPC server on the specified port
func (s *Server) Start(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	// Create gRPC server with options
	s.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(s.loggingInterceptor),
	)

	// Register the geospatial service
	geopb.RegisterGeospatialServiceServer(s.grpcServer, s)

	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s.grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection for debugging
	reflection.Register(s.grpcServer)

	s.logger.WithFields(logger.Fields{
		"port":    port,
		"service": "geo-service-grpc",
	}).Info("Starting gRPC server")

	return s.grpcServer.Serve(lis)
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.logger.Info("Stopping gRPC server")
		s.grpcServer.GracefulStop()
	}
}

// loggingInterceptor provides request logging for gRPC calls
func (s *Server) loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	s.logger.WithFields(logger.Fields{
		"method": info.FullMethod,
	}).Info("gRPC request received")

	resp, err := handler(ctx, req)

	if err != nil {
		s.logger.WithFields(logger.Fields{
			"method": info.FullMethod,
			"error":  err.Error(),
		}).Error("gRPC request failed")
	} else {
		s.logger.WithFields(logger.Fields{
			"method": info.FullMethod,
		}).Info("gRPC request completed")
	}

	return resp, err
}
