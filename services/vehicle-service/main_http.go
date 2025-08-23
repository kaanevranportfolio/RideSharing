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
	"github.com/rideshare-platform/services/vehicle-service/internal/config"
	"github.com/rideshare-platform/services/vehicle-service/internal/handler"
	"github.com/rideshare-platform/services/vehicle-service/internal/repository"
	"github.com/rideshare-platform/services/vehicle-service/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting Vehicle Service on port %s", cfg.HTTPPort)

	// Initialize repositories
	vehicleRepo := repository.NewMemoryVehicleRepository()
	cacheRepo := repository.NewMemoryCacheRepository()

	// Initialize services
	vehicleService := service.NewVehicleService(vehicleRepo, cacheRepo)

	// Initialize HTTP handler
	vehicleHandler := handler.NewHTTPVehicleHandler(vehicleService)

	// Setup HTTP server
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Register routes
	vehicleHandler.RegisterRoutes(router)

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

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
