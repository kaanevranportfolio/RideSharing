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
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/rideshare-platform/services/trip-service/internal/config"
	"github.com/rideshare-platform/services/trip-service/internal/handler"
	"github.com/rideshare-platform/services/trip-service/internal/repository"
	"github.com/rideshare-platform/services/trip-service/internal/service"
	sharedconfig "github.com/rideshare-platform/shared/config"
	"github.com/rideshare-platform/shared/database"
	"github.com/rideshare-platform/shared/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting Trip Service on port %s", cfg.HTTPPort)

	// Map service config to shared database config
	dbCfg := &sharedconfig.DatabaseConfig{
		Host:            cfg.DatabaseHost,
		Port:            cfg.DatabasePort,
		Database:        cfg.DatabaseName,
		Username:        cfg.DatabaseUser,
		Password:        cfg.DatabasePassword,
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300 * time.Second,
		ConnMaxIdleTime: 60 * time.Second,
	}

	logr := logger.NewLogger(cfg.LogLevel, cfg.Environment)
	pg, err := database.NewPostgresDB(dbCfg, logr)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	// Initialize repository and service
	tripRepo := repository.NewTripRepository(pg.DB)
	tripService := service.NewTripService(tripRepo)

	// Initialize HTTP handler
	tripHandler := handler.NewTripHandler(tripService)

	// Setup HTTP server
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Register routes
	tripHandler.RegisterRoutes(router)

	server := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("HTTP server listening on port %s", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Received interrupt signal, starting graceful shutdown...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Failed to shutdown HTTP server gracefully: %v", err)
	}

	log.Println("Trip Service stopped gracefully")
}
