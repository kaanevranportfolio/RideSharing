package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/rideshare-platform/services/geo-service/internal/config"
	grpcServer "github.com/rideshare-platform/services/geo-service/internal/grpc"
	"github.com/rideshare-platform/services/geo-service/internal/handler"
	"github.com/rideshare-platform/services/geo-service/internal/repository"
	"github.com/rideshare-platform/services/geo-service/internal/service"
	"github.com/rideshare-platform/shared/database"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
	geopb "github.com/rideshare-platform/shared/proto/geo"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize logger
	appLogger := logger.NewLogger(cfg.LogLevel, cfg.Environment)

	appLogger.WithFields(logger.Fields{
		"service":   "geo-service",
		"version":   "1.0.0",
		"grpc_port": cfg.GRPCPort,
		"http_port": cfg.HTTPPort,
	}).Info("Starting Geospatial/ETA Service")

	// Initialize database connections
	mongoDB, err := database.NewMongoDB(&cfg.Database, appLogger)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to connect to MongoDB")
	}
	defer func() {
		if err := mongoDB.Close(context.Background()); err != nil {
			appLogger.WithError(err).Error("Failed to close MongoDB connection")
		}
	}()

	redisDB, err := database.NewRedisDB(cfg.Redis, appLogger)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to connect to Redis")
	}
	defer redisDB.Close()

	// Initialize repositories
	driverLocationRepo := repository.NewDriverLocationRepository(mongoDB, appLogger)
	cacheRepo := repository.NewCacheRepository(redisDB, appLogger)

	// Initialize services
	geoService := service.NewGeospatialService(cfg, appLogger, driverLocationRepo, cacheRepo, mongoDB.Client, redisDB.Client)

	// Test the service with sample data
	testService(geoService, appLogger)

	// Initialize HTTP handler
	geoHandler := &handler.GeoHandler{
		Logger:     appLogger,
		GeoService: geoService,
	}

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Register routes
	geoHandler.RegisterRoutes(router)

	// Start gRPC server with health
	grpcSrv := grpc.NewServer()
	geoGrpcServer := grpcServer.NewServer(*geoService, *appLogger)
	geopb.RegisterGeospatialServiceServer(grpcSrv, geoGrpcServer)
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	reflection.Register(grpcSrv)
	go func() {
		lis, err := net.Listen("tcp", ":"+strconv.Itoa(cfg.GRPCPort))
		if err != nil {
			appLogger.WithError(err).Fatal("Failed to listen on gRPC port")
		}
		appLogger.WithFields(logger.Fields{
			"port": cfg.GRPCPort,
		}).Info("Starting gRPC server")
		if err := grpcSrv.Serve(lis); err != nil {
			appLogger.WithError(err).Fatal("Failed to start gRPC server")
		}
	}()

	// Start HTTP server
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.HTTPPort),
		Handler: router,
	}

	go func() {
		appLogger.WithFields(logger.Fields{
			"port": cfg.HTTPPort,
		}).Info("Starting HTTP server")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	appLogger.Logger.Info("Service started successfully. Press Ctrl+C to stop.")

	// Wait for interrupt signal
	<-sigChan
	appLogger.Logger.Info("Received interrupt signal, starting graceful shutdown...")

	// Give time for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server gracefully
	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.WithError(err).Error("Failed to shutdown HTTP server gracefully")
	}

	// Shutdown gRPC server gracefully
	grpcSrv.GracefulStop()

	// Perform cleanup operations
	select {
	case <-shutdownCtx.Done():
		appLogger.Logger.Warn("Graceful shutdown timeout exceeded")
	default:
		appLogger.Logger.Info("Service stopped gracefully")
	}
}

// testService demonstrates the geospatial service functionality
func testService(geoService *service.GeospatialService, logger *logger.Logger) {
	ctx := context.Background()

	logger.Logger.Info("Testing Geospatial Service functionality...")

	// Test locations (New York City area)
	origin := models.Location{
		Latitude:  40.7128,
		Longitude: -74.0060,
		Timestamp: time.Now(),
	}

	destination := models.Location{
		Latitude:  40.7589,
		Longitude: -73.9851,
		Timestamp: time.Now(),
	}

	// Test distance calculation
	logger.Logger.Info("Testing distance calculation...")
	distance, err := geoService.CalculateDistance(ctx, origin, destination, "haversine")
	if err != nil {
		logger.WithError(err).Error("Distance calculation failed")
	} else {
		logger.Logger.WithFields(map[string]interface{}{
			"distance_km": distance.DistanceKm,
			"bearing":     distance.BearingDegrees,
			"method":      distance.CalculationMethod,
		}).Info("Distance calculation successful")
	}

	// Test ETA calculation
	logger.Logger.Info("Testing ETA calculation...")
	eta, err := geoService.CalculateETA(ctx, origin, destination, "car", time.Now(), true)
	if err != nil {
		logger.WithError(err).Error("ETA calculation failed")
	} else {
		logger.Logger.WithFields(map[string]interface{}{
			"duration_minutes": eta.DurationSeconds / 60,
			"distance_km":      eta.DistanceMeters / 1000,
			"vehicle_type":     "car",
		}).Info("ETA calculation successful")
	}

	// Test nearby drivers search
	logger.Logger.Info("Testing nearby drivers search...")
	drivers, err := geoService.FindNearbyDrivers(ctx, origin, 5.0, 10, []string{"sedan", "suv"}, true)
	if err != nil {
		logger.WithError(err).Error("Nearby drivers search failed")
	} else {
		logger.Logger.WithFields(map[string]interface{}{
			"drivers_found": len(drivers),
			"search_radius": 5.0,
		}).Info("Nearby drivers search successful")
	}

	// Test driver location update
	logger.Logger.Info("Testing driver location update...")
	err = geoService.UpdateDriverLocation(ctx, "test_driver_001", origin, "online", "test_vehicle_001")
	if err != nil {
		logger.WithError(err).Error("Driver location update failed")
	} else {
		logger.Logger.Info("Driver location update successful")
	}

	// Test geohash generation
	logger.Logger.Info("Testing geohash generation...")
	geohash, err := geoService.GenerateGeohash(ctx, origin, 7)
	if err != nil {
		logger.WithError(err).Error("Geohash generation failed")
	} else {
		logger.Logger.WithFields(map[string]interface{}{
			"geohash":   geohash,
			"precision": 7,
		}).Info("Geohash generation successful")
	}

	logger.Logger.Info("Geospatial Service testing completed!")
}
