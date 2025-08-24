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
	"github.com/rideshare-platform/services/trip-service/internal/service"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/middleware"
)

func main() {
	// Initialize logger
	logger := logger.NewLogger()
	logger.Info("Starting Enhanced Trip Service")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Initialize enhanced trip service (using in-memory storage for now)
	enhancedTripService := service.NewEnhancedTripService(
		nil, // tripRepo - using nil for now (service has internal storage)
		nil, // eventRepo - using nil for now (service has internal storage)
		logger,
		nil, // geoService - using nil for now
		nil, // priceService - using nil for now
	)

	// Setup HTTP router
	router := mux.NewRouter()

	// Add middleware
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.RecoveryMiddleware(logger))

	// Register API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Basic trip endpoints
	api.HandleFunc("/trips", createTripHandler(enhancedTripService)).Methods("POST")
	api.HandleFunc("/trips/{id}", getTripHandler(enhancedTripService)).Methods("GET")
	api.HandleFunc("/trips/{id}/status", updateTripStatusHandler(enhancedTripService)).Methods("PUT")
	api.HandleFunc("/trips/{id}/location", updateTripLocationHandler(enhancedTripService)).Methods("POST")
	api.HandleFunc("/trips/{id}/history", getTripHistoryHandler(enhancedTripService)).Methods("GET")

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
		logger.WithField("port", cfg.Server.Port).Info("Enhanced Trip Service HTTP server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Enhanced Trip Service...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	} else {
		logger.Info("Enhanced Trip Service stopped gracefully")
	}
}

func createTripHandler(service *service.EnhancedTripService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Implementation would decode CreateTripRequest and call service.CreateTrip
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"error": "CreateTrip handler not implemented yet"}`))
	}
}

func getTripHandler(service *service.EnhancedTripService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Implementation would extract trip ID and call service.GetTrip
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"error": "GetTrip handler not implemented yet"}`))
	}
}

func updateTripStatusHandler(service *service.EnhancedTripService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Implementation would decode UpdateTripStatusRequest and call service.UpdateTripStatus
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"error": "UpdateTripStatus handler not implemented yet"}`))
	}
}

func updateTripLocationHandler(service *service.EnhancedTripService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Implementation would decode TripLocationUpdate and call service.UpdateTripLocation
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"error": "UpdateTripLocation handler not implemented yet"}`))
	}
}

func getTripHistoryHandler(service *service.EnhancedTripService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Implementation would extract trip ID and call service.GetTripHistory
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"error": "GetTripHistory handler not implemented yet"}`))
	}
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
		"status": "healthy",
		"service": "enhanced-trip-service",
		"timestamp": "%s",
		"version": "1.0.0"
	}`, time.Now().Format(time.RFC3339))
}
