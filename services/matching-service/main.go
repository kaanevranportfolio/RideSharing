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
	"github.com/rideshare-platform/services/matching-service/internal/config"
	"github.com/rideshare-platform/services/matching-service/internal/handler"
	"github.com/rideshare-platform/services/matching-service/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting Matching Service on port %s", cfg.HTTPPort)

	// Initialize services
	matchingService := service.NewSimpleMatchingService(cfg)

	// Initialize HTTP handler
	matchingHandler := handler.NewMatchingHandler(matchingService)

	// Setup HTTP server
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Register routes
	matchingHandler.RegisterRoutes(router)

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

	log.Println("Matching Service stopped gracefully")
}
