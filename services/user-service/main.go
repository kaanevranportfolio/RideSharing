package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rideshare-platform/services/user-service/internal/config"
	"github.com/rideshare-platform/services/user-service/internal/handler"
	"github.com/rideshare-platform/services/user-service/internal/metrics"
	"github.com/rideshare-platform/services/user-service/internal/repository"
	"github.com/rideshare-platform/services/user-service/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting User Service on port %s", cfg.HTTPPort)

	// Connect to database
	dbConnectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseUser,
		cfg.DatabasePassword, cfg.DatabaseName, cfg.DatabaseSSLMode)

	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Printf("Connected to PostgreSQL database")

	// Initialize repository and service
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)

	// Initialize HTTP handler
	userHandler := handler.NewUserHandler(userService)

	// Setup HTTP server
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(metrics.PrometheusMiddleware())

	// Register routes
	userHandler.RegisterRoutes(router)

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Prometheus metrics endpoint
	router.GET("/api/v1/metrics", gin.WrapH(promhttp.Handler()))

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
