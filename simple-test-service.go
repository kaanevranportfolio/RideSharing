package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"context"
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Services  map[string]interface{} `json:"services"`
}

type TestService struct {
	postgres *sql.DB
	mongodb  *mongo.Client
	redis    *redis.Client
}

func main() {
	ts := &TestService{}

	// Initialize database connections
	ts.initConnections()

	// Setup HTTP handlers
	http.HandleFunc("/health", ts.healthHandler)
	http.HandleFunc("/test/postgres", ts.testPostgres)
	http.HandleFunc("/test/mongodb", ts.testMongoDB)
	http.HandleFunc("/test/redis", ts.testRedis)
	http.HandleFunc("/test/all", ts.testAll)

	fmt.Println("Test service starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (ts *TestService) initConnections() {
	var err error

	// PostgreSQL connection
	postgresURL := "postgres://rideshare_user:rideshare_password@localhost:5432/rideshare?sslmode=disable"
	ts.postgres, err = sql.Open("postgres", postgresURL)
	if err != nil {
		log.Printf("Failed to connect to PostgreSQL: %v", err)
	}

	// MongoDB connection
	mongoURL := "mongodb://rideshare_user:rideshare_password@localhost:27017"
	clientOptions := options.Client().ApplyURI(mongoURL)
	ts.mongodb, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Printf("Failed to connect to MongoDB: %v", err)
	}

	// Redis connection
	ts.redis = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password
		DB:       0,  // default DB
	})
}

func (ts *TestService) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  make(map[string]interface{}),
	}

	// Test PostgreSQL
	if ts.postgres != nil {
		if err := ts.postgres.Ping(); err == nil {
			response.Services["postgresql"] = "healthy"
		} else {
			response.Services["postgresql"] = fmt.Sprintf("unhealthy: %v", err)
		}
	} else {
		response.Services["postgresql"] = "not connected"
	}

	// Test MongoDB
	if ts.mongodb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := ts.mongodb.Ping(ctx, nil); err == nil {
			response.Services["mongodb"] = "healthy"
		} else {
			response.Services["mongodb"] = fmt.Sprintf("unhealthy: %v", err)
		}
	} else {
		response.Services["mongodb"] = "not connected"
	}

	// Test Redis
	if ts.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, err := ts.redis.Ping(ctx).Result(); err == nil {
			response.Services["redis"] = "healthy"
		} else {
			response.Services["redis"] = fmt.Sprintf("unhealthy: %v", err)
		}
	} else {
		response.Services["redis"] = "not connected"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ts *TestService) testPostgres(w http.ResponseWriter, r *http.Request) {
	if ts.postgres == nil {
		http.Error(w, "PostgreSQL not connected", http.StatusServiceUnavailable)
		return
	}

	var count int
	err := ts.postgres.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"service":    "postgresql",
		"status":     "healthy",
		"user_count": count,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ts *TestService) testMongoDB(w http.ResponseWriter, r *http.Request) {
	if ts.mongodb == nil {
		http.Error(w, "MongoDB not connected", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db := ts.mongodb.Database("rideshare_geo")
	collection := db.Collection("driver_locations")

	count, err := collection.CountDocuments(ctx, map[string]interface{}{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Query failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"service":               "mongodb",
		"status":                "healthy",
		"driver_location_count": count,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ts *TestService) testRedis(w http.ResponseWriter, r *http.Request) {
	if ts.redis == nil {
		http.Error(w, "Redis not connected", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test basic operations
	testKey := "test:health_check"
	err := ts.redis.Set(ctx, testKey, "test_value", time.Minute).Err()
	if err != nil {
		http.Error(w, fmt.Sprintf("Redis SET failed: %v", err), http.StatusInternalServerError)
		return
	}

	val, err := ts.redis.Get(ctx, testKey).Result()
	if err != nil {
		http.Error(w, fmt.Sprintf("Redis GET failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Cleanup
	ts.redis.Del(ctx, testKey)

	response := map[string]interface{}{
		"service":    "redis",
		"status":     "healthy",
		"test_value": val,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ts *TestService) testAll(w http.ResponseWriter, r *http.Request) {
	results := make(map[string]interface{})

	// Test all services
	if ts.postgres != nil {
		var count int
		if err := ts.postgres.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err == nil {
			results["postgresql"] = map[string]interface{}{
				"status":     "healthy",
				"user_count": count,
			}
		} else {
			results["postgresql"] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		}
	}

	if ts.mongodb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		db := ts.mongodb.Database("rideshare_geo")
		collection := db.Collection("driver_locations")

		if count, err := collection.CountDocuments(ctx, map[string]interface{}{}); err == nil {
			results["mongodb"] = map[string]interface{}{
				"status":                "healthy",
				"driver_location_count": count,
			}
		} else {
			results["mongodb"] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		}
	}

	if ts.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := ts.redis.Ping(ctx).Result(); err == nil {
			results["redis"] = map[string]interface{}{
				"status": "healthy",
			}
		} else {
			results["redis"] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		}
	}

	response := map[string]interface{}{
		"timestamp": time.Now(),
		"results":   results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
