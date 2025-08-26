//go:build integration
// +build integration

package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/rideshare-platform/services/user-service/internal/repository"
	"github.com/rideshare-platform/shared/models"
)

func getTestDB() (*sql.DB, error) {
	// Use test database configuration
	postgresHost := getEnv("TEST_POSTGRES_HOST", "localhost")
	postgresPort := getEnv("TEST_POSTGRES_PORT", "5433")
	postgresUser := getEnv("TEST_POSTGRES_USER", "postgres")
	postgresPassword := getEnv("TEST_POSTGRES_PASSWORD", "testpass_change_me")
	postgresDB := getEnv("TEST_POSTGRES_DB", "rideshare_test")

	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		postgresUser, postgresPassword, postgresHost, postgresPort, postgresDB)

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping test database: %w", err)
	}

	return db, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupTestTable(t *testing.T, db *sql.DB) {
	t.Helper()

	// Clean up any existing test data and ensure table exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(255) PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			phone VARCHAR(20),
			password_hash VARCHAR(255),
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			user_type VARCHAR(20) DEFAULT 'rider',
			status VARCHAR(20) DEFAULT 'active',
			profile_image_url TEXT,
			email_verified BOOLEAN DEFAULT FALSE,
			phone_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		DELETE FROM users WHERE email LIKE '%example.com' OR email LIKE '%real@%';
	`)
	if err != nil {
		t.Fatalf("Failed to setup users table: %v", err)
	}
}

func TestUserService_RealIntegration_CreateUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, err := getTestDB()
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	setupTestTable(t, db)

	// Setup repository and service with REAL implementations
	userRepo := repository.NewUserRepository(db)
	userService := NewUserService(userRepo)

	tests := []struct {
		name        string
		user        *models.User
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful_user_creation_with_real_db",
			user: &models.User{
				Email:     "real.integration@example.com",
				Phone:     "+1234567890", // Add unique phone
				FirstName: "Real",
				LastName:  "Integration",
				UserType:  "rider",
			},
			expectError: false,
		},
		{
			name: "email_validation_with_real_service",
			user: &models.User{
				FirstName: "Test",
				LastName:  "User",
			},
			expectError: true,
			errorMsg:    "user email is required",
		},
		{
			name: "duplicate_email_with_real_db",
			user: &models.User{
				Email:     "duplicate.real@example.com",
				FirstName: "Duplicate",
				LastName:  "User",
			},
			expectError: true,
			errorMsg:    "user with this email already exists",
		},
	}

	// Pre-create a user for duplicate email test using REAL service
	_, err = userService.CreateUser(context.Background(), &models.User{
		Email:     "duplicate.real@example.com",
		Phone:     "+9876543210", // Unique phone
		FirstName: "Existing",
		LastName:  "User",
	})
	if err != nil {
		t.Fatalf("Failed to pre-create user for duplicate test: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Call the REAL user service
			result, err := userService.CreateUser(ctx, tt.user)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
				t.Logf("✓ Real service validation works: %v", err)
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if result == nil {
					t.Errorf("Expected user result but got nil")
				} else {
					t.Logf("✓ Real service created user: ID=%s, Email=%s", result.ID, result.Email)

					// Verify the user was actually created in the REAL database
					retrievedUser, err := userService.GetUser(ctx, result.ID)
					if err != nil {
						t.Errorf("Failed to retrieve created user from real DB: %v", err)
					} else if retrievedUser == nil {
						t.Errorf("User was not found in real database after creation")
					} else if retrievedUser.Email != tt.user.Email {
						t.Errorf("Real DB user email mismatch: expected %s, got %s", tt.user.Email, retrievedUser.Email)
					} else {
						t.Logf("✓ Real database persistence verified: %s", retrievedUser.Email)
					}
				}
			}
		})
	}
}

func TestUserService_RealIntegration_GetUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, err := getTestDB()
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	setupTestTable(t, db)

	// Setup repository and service with REAL implementations
	userRepo := repository.NewUserRepository(db)
	userService := NewUserService(userRepo)

	// Create a test user first using REAL service
	ctx := context.Background()
	createdUser, err := userService.CreateUser(ctx, &models.User{
		Email:     "gettest.real@example.com",
		Phone:     "+5555555555", // Unique phone
		FirstName: "GetTest",
		LastName:  "Real",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name        string
		userID      string
		expectError bool
		expectNil   bool
		errorMsg    string
	}{
		{
			name:        "empty_user_id_real_validation",
			userID:      "",
			expectError: true,
			errorMsg:    "user ID is required",
		},
		{
			name:        "non_existent_user_real_db",
			userID:      "550e8400-e29b-41d4-a716-446655440000", // Valid UUID format
			expectError: false,
			expectNil:   true,
		},
		{
			name:        "successful_user_retrieval_real_db",
			userID:      createdUser.ID,
			expectError: false,
			expectNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Call the REAL user service
			result, err := userService.GetUser(ctx, tt.userID)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
				t.Logf("✓ Real service validation works: %v", err)
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if tt.expectNil && result != nil {
					t.Errorf("Expected nil result but got: %+v", result)
				} else if !tt.expectNil && result == nil {
					t.Errorf("Expected user result but got nil")
				} else if !tt.expectNil && result.ID != tt.userID {
					t.Errorf("Expected user ID %s, got %s", tt.userID, result.ID)
				} else if !tt.expectNil {
					t.Logf("✓ Real database retrieval successful: ID=%s, Email=%s", result.ID, result.Email)
				} else {
					t.Logf("✓ Real database correctly returned nil for non-existent user")
				}
			}
		})
	}
}
