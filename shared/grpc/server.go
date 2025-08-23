package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"github.com/rideshare-platform/shared/logger"
)

// ServerConfig holds gRPC server configuration
type ServerConfig struct {
	Port                int
	MaxRecvMsgSize      int
	MaxSendMsgSize      int
	ConnectionTimeout   time.Duration
	MaxConnectionIdle   time.Duration
	MaxConnectionAge    time.Duration
	MaxConnectionAgeGrace time.Duration
	Time                time.Duration
	Timeout             time.Duration
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:                  50051,
		MaxRecvMsgSize:        4 * 1024 * 1024, // 4MB
		MaxSendMsgSize:        4 * 1024 * 1024, // 4MB
		ConnectionTimeout:     5 * time.Second,
		MaxConnectionIdle:     15 * time.Second,
		MaxConnectionAge:      30 * time.Second,
		MaxConnectionAgeGrace: 5 * time.Second,
		Time:                  5 * time.Second,
		Timeout:               1 * time.Second,
	}
}

// Server wraps gRPC server with additional functionality
type Server struct {
	server *grpc.Server
	config *ServerConfig
	logger *logger.Logger
}

// NewServer creates a new gRPC server
func NewServer(config *ServerConfig, log *logger.Logger) *Server {
	// Server options
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(config.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(config.MaxSendMsgSize),
		grpc.ConnectionTimeout(config.ConnectionTimeout),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     config.MaxConnectionIdle,
			MaxConnectionAge:      config.MaxConnectionAge,
			MaxConnectionAgeGrace: config.MaxConnectionAgeGrace,
			Time:                  config.Time,
			Timeout:               config.Timeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.UnaryInterceptor(unaryServerInterceptor(log)),
		grpc.StreamInterceptor(streamServerInterceptor(log)),
	}

	server := grpc.NewServer(opts...)
	
	// Enable reflection for development
	reflection.Register(server)

	return &Server{
		server: server,
		config: config,
		logger: log,
	}
}

// GetServer returns the underlying gRPC server
func (s *Server) GetServer() *grpc.Server {
	return s.server
}

// Start starts the gRPC server
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.config.Port, err)
	}

	s.logger.WithFields(logger.Fields{
		"port": s.config.Port,
	}).Info("Starting gRPC server")

	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %w", err)
	}

	return nil
}

// Stop stops the gRPC server gracefully
func (s *Server) Stop() {
	s.logger.Logger.Info("Stopping gRPC server")
	s.server.GracefulStop()
}

// ForceStop stops the gRPC server immediately
func (s *Server) ForceStop() {
	s.logger.Logger.Info("Force stopping gRPC server")
	s.server.Stop()
}

// unaryServerInterceptor provides logging and metrics for unary RPCs
func unaryServerInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Call the handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Log the request
		log.LogGRPCRequest(ctx, info.FullMethod, duration, err)

		return resp, err
	}
}

// streamServerInterceptor provides logging and metrics for streaming RPCs
func streamServerInterceptor(log *logger.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// Call the handler
		err := handler(srv, stream)

		// Calculate duration
		duration := time.Since(start)

		// Log the request
		log.LogGRPCRequest(stream.Context(), info.FullMethod, duration, err)

		return err
	}
}

// HealthServer implements gRPC health checking
type HealthServer struct {
	logger *logger.Logger
}

// NewHealthServer creates a new health server
func NewHealthServer(log *logger.Logger) *HealthServer {
	return &HealthServer{
		logger: log,
	}
}

// Check implements the health check
func (h *HealthServer) Check(ctx context.Context, req interface{}) (interface{}, error) {
	h.logger.WithContext(ctx).Debug("Health check requested")
	
	// Simple health check - in production, check dependencies
	return map[string]string{
		"status": "SERVING",
	}, nil
}

// Watch implements the health watch (streaming)
func (h *HealthServer) Watch(req interface{}, stream grpc.ServerStream) error {
	h.logger.WithContext(stream.Context()).Debug("Health watch requested")
	
	// Send periodic health updates
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			// Send health status
			// In a real implementation, you would send proper health status messages
			h.logger.WithContext(stream.Context()).Debug("Sending health status")
		}
	}
}

// ErrorHandler provides standardized error handling
type ErrorHandler struct {
	logger *logger.Logger
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(log *logger.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: log,
	}
}

// HandleError converts application errors to gRPC errors
func (eh *ErrorHandler) HandleError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	eh.logger.WithContext(ctx).WithError(err).Error("gRPC error occurred")

	// Convert to appropriate gRPC status
	switch {
	case isNotFoundError(err):
		return status.Error(codes.NotFound, err.Error())
	case isValidationError(err):
		return status.Error(codes.InvalidArgument, err.Error())
	case isUnauthorizedError(err):
		return status.Error(codes.Unauthenticated, err.Error())
	case isForbiddenError(err):
		return status.Error(codes.PermissionDenied, err.Error())
	case isConflictError(err):
		return status.Error(codes.AlreadyExists, err.Error())
	case isTimeoutError(err):
		return status.Error(codes.DeadlineExceeded, err.Error())
	default:
		return status.Error(codes.Internal, "Internal server error")
	}
}

// Helper functions to identify error types
func isNotFoundError(err error) bool {
	// Implement based on your error types
	return false
}

func isValidationError(err error) bool {
	// Implement based on your error types
	return false
}

func isUnauthorizedError(err error) bool {
	// Implement based on your error types
	return false
}

func isForbiddenError(err error) bool {
	// Implement based on your error types
	return false
}

func isConflictError(err error) bool {
	// Implement based on your error types
	return false
}

func isTimeoutError(err error) bool {
	// Implement based on your error types
	return false
}

// ServiceRegistry manages service registration and discovery
type ServiceRegistry struct {
	services map[string]string // service name -> address
	logger   *logger.Logger
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(log *logger.Logger) *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]string),
		logger:   log,
	}
}

// Register registers a service
func (sr *ServiceRegistry) Register(serviceName, address string) {
	sr.services[serviceName] = address
	sr.logger.WithFields(logger.Fields{
		"service": serviceName,
		"address": address,
	}).Info("Service registered")
}

// Discover discovers a service address
func (sr *ServiceRegistry) Discover(serviceName string) (string, bool) {
	address, exists := sr.services[serviceName]
	if exists {
		sr.logger.WithFields(logger.Fields{
			"service": serviceName,
			"address": address,
		}).Debug("Service discovered")
	} else {
		sr.logger.WithFields(logger.Fields{
			"service": serviceName,
		}).Warn("Service not found")
	}
	return address, exists
}

// Unregister unregisters a service
func (sr *ServiceRegistry) Unregister(serviceName string) {
	delete(sr.services, serviceName)
	sr.logger.WithFields(logger.Fields{
		"service": serviceName,
	}).Info("Service unregistered")
}

// ListServices lists all registered services
func (sr *ServiceRegistry) ListServices() map[string]string {
	return sr.services
}

// ServerManager manages multiple gRPC servers
type ServerManager struct {
	servers map[string]*Server
	logger  *logger.Logger
}

// NewServerManager creates a new server manager
func NewServerManager(log *logger.Logger) *ServerManager {
	return &ServerManager{
		servers: make(map[string]*Server),
		logger:  log,
	}
}

// AddServer adds a server to the manager
func (sm *ServerManager) AddServer(name string, server *Server) {
	sm.servers[name] = server
	sm.logger.WithFields(logger.Fields{
		"server": name,
	}).Info("Server added to manager")
}

// StartAll starts all servers
func (sm *ServerManager) StartAll() error {
	for name, server := range sm.servers {
		go func(serverName string, srv *Server) {
			if err := srv.Start(); err != nil {
				sm.logger.WithError(err).WithFields(logger.Fields{
					"server": serverName,
				}).Error("Failed to start server")
			}
		}(name, server)
	}
	return nil
}

// StopAll stops all servers gracefully
func (sm *ServerManager) StopAll() {
	for name, server := range sm.servers {
		sm.logger.WithFields(logger.Fields{
			"server": name,
		}).Info("Stopping server")
		server.Stop()
	}
}

// ForceStopAll force stops all servers
func (sm *ServerManager) ForceStopAll() {
	for name, server := range sm.servers {
		sm.logger.WithFields(logger.Fields{
			"server": name,
		}).Info("Force stopping server")
		server.ForceStop()
	}
}

// GetServer gets a server by name
func (sm *ServerManager) GetServer(name string) (*Server, bool) {
	server, exists := sm.servers[name]
	return server, exists
}