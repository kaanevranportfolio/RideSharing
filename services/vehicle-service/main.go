package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rideshare-platform/services/vehicle-service/internal/config"
	"github.com/rideshare-platform/services/vehicle-service/internal/handler"
	"github.com/rideshare-platform/services/vehicle-service/internal/repository"
	"github.com/rideshare-platform/services/vehicle-service/internal/service"
	"github.com/rideshare-platform/shared/database"
	"github.com/rideshare-platform/shared/events"
	"github.com/rideshare-platform/shared/grpc"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	appLogger := logger.NewLogger(cfg.LogLevel, cfg.Environment)

	appLogger.WithFields(logger.Fields{
		"service": "vehicle-service",
		"version": "1.0.0",
		"port":    cfg.GRPCPort,
	}).Info("Starting Vehicle Management Service")

	// Initialize database connections
	postgresDB, err := database.NewPostgresDB(&cfg.Database, appLogger)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to connect to PostgreSQL")
	}
	defer postgresDB.Close()

	redisDB, err := database.NewRedisDB(&cfg.Database, appLogger)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to connect to Redis")
	}
	defer redisDB.Close()

	// Initialize event system
	eventBus := events.NewInMemoryEventBus(appLogger)
	eventStore := events.NewInMemoryEventStore(appLogger)
	eventPublisher := events.NewEventPublisher(eventBus, eventStore, appLogger)
	defer eventPublisher.Close()

	// Initialize repositories
	vehicleRepo := repository.NewVehicleRepository(postgresDB, appLogger)
	cacheRepo := repository.NewCacheRepository(redisDB, appLogger)

	// Initialize services
	vehicleService := service.NewVehicleService(vehicleRepo, cacheRepo, eventPublisher, appLogger)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTSecret, appLogger)
	metricsMiddleware := middleware.NewMetricsMiddleware("vehicle-service", appLogger)

	// Initialize gRPC server
	grpcConfig := grpc.DefaultServerConfig()
	grpcConfig.Port = cfg.GRPCPort
	grpcServer := grpc.NewServer(grpcConfig, appLogger)

	// Register gRPC handlers
	vehicleHandler := handler.NewVehicleHandler(vehicleService, appLogger)
	vehicleHandler.RegisterWithServer(grpcServer.GetServer())

	// Start gRPC server in a goroutine
	go func() {
		if err := grpcServer.Start(); err != nil {
			appLogger.WithError(err).Fatal("Failed to start gRPC server")
		}
	}()

	// Initialize HTTP server for health checks and metrics
	httpServer := handler.NewHTTPServer(cfg.HTTPPort, authMiddleware, metricsMiddleware, appLogger)
	go func() {
		if err := httpServer.Start(); err != nil {
			appLogger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Logger.Info("Shutting down Vehicle Management Service")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		appLogger.WithError(err).Error("Failed to shutdown HTTP server")
	}

	// Shutdown gRPC server
	grpcServer.Stop()

	appLogger.Logger.Info("Vehicle Management Service stopped")
}
