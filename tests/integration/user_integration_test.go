//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/rideshare-platform/services/user-service/internal/repository"
	"github.com/rideshare-platform/services/user-service/internal/service"
	"github.com/rideshare-platform/shared/models"
	"github.com/rideshare-platform/tests/testutils"
)

func TestUserService_Integration_CreateUser(t *testing.T) {
	testutils.SkipIfShort(t)

	config := testutils.DefaultTestConfig()
	db := testutils.SetupTestDB(t, config)
	defer testutils.CleanupTestDB(t, db)

	// Create the users table for testing
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Setup repository and service
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)

	tests := []struct {
		name        string
		user        *models.User
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful_user_creation",
			user: &models.User{
				Email:     "integration.test@example.com",
				FirstName: "Integration",
				LastName:  "Test",
				UserType:  "rider",
			},
			expectError: false,
		},
		{
			name: "email_validation_error",
			user: &models.User{
				FirstName: "Test",
				LastName:  "User",
			},
			expectError: true,
			errorMsg:    "user email is required",
		},
		{
			name: "duplicate_email_error",
			user: &models.User{
				Email:     "duplicate@example.com",
				FirstName: "Duplicate",
				LastName:  "User",
			},
			expectError: true,
			errorMsg:    "user with this email already exists",
		},
	}

	// Pre-create a user for duplicate email test
	_, err = userService.CreateUser(context.Background(), &models.User{
		Email:     "duplicate@example.com",
		FirstName: "Existing",
		LastName:  "User",
	})
	if err != nil {
		t.Fatalf("Failed to pre-create user for duplicate test: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := userService.CreateUser(ctx, tt.user)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if result == nil {
					t.Errorf("Expected user result but got nil")
				} else {
					// Verify the user was actually created in the database
					retrievedUser, err := userService.GetUser(ctx, result.ID)
					if err != nil {
						t.Errorf("Failed to retrieve created user: %v", err)
					} else if retrievedUser == nil {
						t.Errorf("User was not found in database after creation")
					} else if retrievedUser.Email != tt.user.Email {
						t.Errorf("Created user email mismatch: expected %s, got %s", tt.user.Email, retrievedUser.Email)
					}
				}
			}
		})
	}
}

func TestUserService_Integration_GetUser(t *testing.T) {
	testutils.SkipIfShort(t)

	config := testutils.DefaultTestConfig()
	db := testutils.SetupTestDB(t, config)
	defer testutils.CleanupTestDB(t, db)

	// Create the users table for testing
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Setup repository and service
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)

	// Create a test user first
	ctx := context.Background()
	createdUser, err := userService.CreateUser(ctx, &models.User{
		Email:     "gettest@example.com",
		FirstName: "Get",
		LastName:  "Test",
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
			name:        "empty_user_id",
			userID:      "",
			expectError: true,
			errorMsg:    "user ID is required",
		},
		{
			name:        "non_existent_user",
			userID:      "non-existent-id",
			expectError: false,
			expectNil:   true,
		},
		{
			name:        "successful_user_retrieval",
			userID:      createdUser.ID,
			expectError: false,
			expectNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := userService.GetUser(ctx, tt.userID)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if tt.expectNil && result != nil {
					t.Errorf("Expected nil result but got: %+v", result)
				} else if !tt.expectNil && result == nil {
					t.Errorf("Expected user result but got nil")
				} else if !tt.expectNil && result.ID != tt.userID {
					t.Errorf("Expected user ID %s, got %s", tt.userID, result.ID)
				}
			}
		})
	}
}

func TestUserService_Integration_GetUserByEmail(t *testing.T) {
	testutils.SkipIfShort(t)

	config := testutils.DefaultTestConfig()
	db := testutils.SetupTestDB(t, config)
	defer testutils.CleanupTestDB(t, db)

	// Create the users table for testing
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Setup repository and service
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)

	// Create a test user first
	ctx := context.Background()
	testEmail := "emailtest@example.com"
	createdUser, err := userService.CreateUser(ctx, &models.User{
		Email:     testEmail,
		FirstName: "Email",
		LastName:  "Test",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name        string
		email       string
		expectError bool
		expectNil   bool
	}{
		{
			name:        "non_existent_email",
			email:       "nonexistent@example.com",
			expectError: false,
			expectNil:   true,
		},
		{
			name:        "successful_email_lookup",
			email:       testEmail,
			expectError: false,
			expectNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := userService.GetUserByEmail(ctx, tt.email)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if tt.expectNil && result != nil {
					t.Errorf("Expected nil result but got: %+v", result)
				} else if !tt.expectNil && result == nil {
					t.Errorf("Expected user result but got nil")
				} else if !tt.expectNil && result.Email != tt.email {
					t.Errorf("Expected user email %s, got %s", tt.email, result.Email)
				} else if !tt.expectNil && result.ID != createdUser.ID {
					t.Errorf("Expected user ID %s, got %s", createdUser.ID, result.ID)
				}
			}
		})
	}
}
