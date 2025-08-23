package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rideshare-platform/services/vehicle-service/internal/config"
	"github.com/rideshare-platform/services/vehicle-service/internal/handler"
	"github.com/rideshare-platform/services/vehicle-service/internal/repository"
	"github.com/rideshare-platform/services/vehicle-service/internal/service"
	"github.com/rideshare-platform/shared/database"
	"github.com/rideshare-platform/shared/events"
	"github.com/rideshare-platform/shared/logger"
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

	redisDB, err := database.NewRedisDB(cfg.Redis, appLogger)
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

	// Initialize Gin HTTP handler
	vehicleHandler := handler.NewVehicleHandler(vehicleService)

	router := gin.New()
	vehicleHandler.RegisterRoutes(router)

	// Register /ready endpoint for readiness probe
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ready",
			"service":   "vehicle-service",
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
		})
	})

	httpServer := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.HTTPPort),
		Handler: router,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

	if err := httpServer.Shutdown(ctx); err != nil {
		appLogger.WithError(err).Error("Failed to shutdown HTTP server")
	}

	appLogger.Logger.Info("Vehicle Management Service stopped")
}
