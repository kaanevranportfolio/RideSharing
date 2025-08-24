package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/rideshare-platform/services/trip-service/internal/config"
	"github.com/rideshare-platform/services/trip-service/internal/handler"
	"github.com/rideshare-platform/services/trip-service/internal/repository"
	"github.com/rideshare-platform/services/trip-service/internal/service"
	"github.com/rideshare-platform/shared/database"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/middleware"
)

func main() {
	// Initialize logger
	logger := logger.NewLogger()
	logger.Info("Starting Trip Service")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Initialize MongoDB connection
	mongoClient, err := database.NewMongoClient(cfg.Database.MongoURI)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to MongoDB")
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			logger.WithError(err).Error("Error disconnecting from MongoDB")
		}
	}()

	mongoDB := mongoClient.Database(cfg.Database.DBName)

	// Initialize repositories
	tripRepo := repository.NewMongoTripRepository(mongoDB, logger)
	eventRepo := repository.NewMongoEventRepository(mongoDB, logger)

	// Initialize services
	enhancedTripService := service.NewEnhancedTripService(tripRepo, eventRepo, logger)

	// Initialize handlers
	tripHandler := handler.NewTripHandler(enhancedTripService, logger)

	// Setup HTTP router
	router := mux.NewRouter()

	// Add middleware
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.RecoveryMiddleware(logger))

	// Register API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	tripHandler.RegisterRoutes(api)

	// Health check endpoint
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")

	// Start HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("port", cfg.Server.Port).Info("Trip Service HTTP server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Trip Service...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	} else {
		logger.Info("Trip Service stopped gracefully")
	}
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
		"status": "healthy",
		"service": "trip-service",
		"timestamp": "%s",
		"version": "1.0.0"
	}`, time.Now().Format(time.RFC3339))
}
