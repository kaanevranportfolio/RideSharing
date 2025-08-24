package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rideshare-platform/services/api-gateway/internal/grpc"
)

// Simple HTTP handlers for now, we'll add GraphQL later
func main() {
	log.Println("🚀 Starting Rideshare API Gateway...")

	// Initialize gRPC client manager
	grpcClient := grpc.NewClientManager()
	if err := grpcClient.Initialize(); err != nil {
		log.Printf("Failed to initialize gRPC clients: %v", err)
		// Continue anyway for graceful degradation
	}

	// Create HTTP router
	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		health := grpcClient.HealthCheck(r.Context())
		w.Header().Set("Content-Type", "application/json")

		allHealthy := true
		for _, healthy := range health {
			if !healthy {
				allHealthy = false
				break
			}
		}

		if allHealthy {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "healthy", "services": "all connected"}`))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status": "degraded", "message": "some services unavailable"}`))
		}
	}).Methods("GET")

	// Service status endpoint
	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		status := grpcClient.GetConnectionStatus()
		w.Header().Set("Content-Type", "application/json")

		response := `{"connections": {`
		first := true
		for service, state := range status {
			if !first {
				response += ","
			}
			response += `"` + service + `": "` + state + `"`
			first = false
		}
		response += `}}`

		w.Write([]byte(response))
	}).Methods("GET")

	// WebSocket upgrade helper
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	// WebSocket endpoint for real-time updates
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade WebSocket: %v", err)
			return
		}
		defer conn.Close()

		// Simple ping-pong for now
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("WebSocket read error: %v", err)
				break
			}

			// Echo the message back
			if err := conn.WriteMessage(messageType, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				break
			}
		}
	})

	// REST API endpoints (simplified for now)
	api := router.PathPrefix("/api/v1").Subrouter()

	// User endpoints
	api.HandleFunc("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID := vars["id"]

		if grpcClient.UserClient == nil {
			http.Error(w, "User service unavailable", http.StatusServiceUnavailable)
			return
		}

		// This would call the gRPC service
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": "` + userID + `", "status": "mock response - gRPC integration needed"}`))
	}).Methods("GET")

	// Trip endpoints
	api.HandleFunc("/trips/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tripID := vars["id"]

		if grpcClient.TripClient == nil {
			http.Error(w, "Trip service unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": "` + tripID + `", "status": "mock response - gRPC integration needed"}`))
	}).Methods("GET")

	// Price estimate endpoint
	api.HandleFunc("/pricing/estimate", func(w http.ResponseWriter, r *http.Request) {
		if grpcClient.PricingClient == nil {
			http.Error(w, "Pricing service unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"estimated_fare": 15.50, "currency": "USD", "status": "mock response"}`))
	}).Methods("POST")

	// Driver matching endpoint
	api.HandleFunc("/matching/nearby-drivers", func(w http.ResponseWriter, r *http.Request) {
		if grpcClient.MatchingClient == nil {
			http.Error(w, "Matching service unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"drivers": [], "status": "mock response - gRPC integration needed"}`))
	}).Methods("POST")

	// Payment endpoints
	api.HandleFunc("/payments", func(w http.ResponseWriter, r *http.Request) {
		if grpcClient.PaymentClient == nil {
			http.Error(w, "Payment service unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"payment_id": "pay_123", "status": "mock response"}`))
	}).Methods("POST")

	// CORS middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Start server
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("✅ API Gateway listening on :8080")
	log.Println("📊 Health check: http://localhost:8080/health")
	log.Println("📈 Status check: http://localhost:8080/status")
	log.Println("🔌 WebSocket: ws://localhost:8080/ws")
	log.Println("📡 REST API: http://localhost:8080/api/v1")

	// Graceful shutdown
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		log.Println("🛑 Shutting down API Gateway...")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		srv.Shutdown(ctx)
		grpcClient.Close()
	}()

	// Start serving
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("✅ API Gateway stopped gracefully")
}
