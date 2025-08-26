package service

import (
	"context"
	"errors"
	"testing"

	"github.com/rideshare-platform/shared/models"
)

// MockUserRepository implements the UserRepositoryInterface for testing
type MockUserRepository struct {
	users       map[string]*models.User
	emailIndex  map[string]*models.User
	shouldError bool
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:      make(map[string]*models.User),
		emailIndex: make(map[string]*models.User),
	}
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	if m.shouldError {
		return nil, errors.New("database error")
	}

	// Generate ID if not set
	if user.ID == "" {
		user.ID = "user-123"
	}

	m.users[user.ID] = user
	m.emailIndex[user.Email] = user
	return user, nil
}

func (m *MockUserRepository) GetUser(ctx context.Context, userID string) (*models.User, error) {
	if m.shouldError {
		return nil, errors.New("database error")
	}

	user, exists := m.users[userID]
	if !exists {
		return nil, nil
	}
	return user, nil
}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.shouldError {
		return nil, errors.New("database error")
	}

	user, exists := m.emailIndex[email]
	if !exists {
		return nil, nil
	}
	return user, nil
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	if m.shouldError {
		return nil, errors.New("database error")
	}

	m.users[user.ID] = user
	m.emailIndex[user.Email] = user
	return user, nil
}

func (m *MockUserRepository) DeleteUser(ctx context.Context, userID string) error {
	if m.shouldError {
		return errors.New("database error")
	}

	user, exists := m.users[userID]
	if exists {
		delete(m.users, userID)
		delete(m.emailIndex, user.Email)
	}
	return nil
}

func (m *MockUserRepository) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	if m.shouldError {
		return nil, errors.New("database error")
	}

	var users []*models.User
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func (m *MockUserRepository) SetShouldError(shouldError bool) {
	m.shouldError = shouldError
}

func TestUserService_CreateUser(t *testing.T) {
	tests := []struct {
		name        string
		user        *models.User
		expectError bool
		errorMsg    string
		setupMock   func(*MockUserRepository)
	}{
		{
			name: "successful_user_creation",
			user: &models.User{
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "Doe",
				UserType:  "rider",
			},
			expectError: false,
			setupMock:   func(m *MockUserRepository) {},
		},
		{
			name: "empty_email_error",
			user: &models.User{
				FirstName: "John",
				LastName:  "Doe",
			},
			expectError: true,
			errorMsg:    "user email is required",
			setupMock:   func(m *MockUserRepository) {},
		},
		{
			name: "empty_first_name_error",
			user: &models.User{
				Email:    "test@example.com",
				LastName: "Doe",
			},
			expectError: true,
			errorMsg:    "user first name is required",
			setupMock:   func(m *MockUserRepository) {},
		},
		{
			name: "empty_last_name_error",
			user: &models.User{
				Email:     "test@example.com",
				FirstName: "John",
			},
			expectError: true,
			errorMsg:    "user last name is required",
			setupMock:   func(m *MockUserRepository) {},
		},
		{
			name: "duplicate_email_error",
			user: &models.User{
				Email:     "existing@example.com",
				FirstName: "Jane",
				LastName:  "Doe",
			},
			expectError: true,
			errorMsg:    "user with this email already exists",
			setupMock: func(m *MockUserRepository) {
				// Pre-populate with existing user
				m.emailIndex["existing@example.com"] = &models.User{
					ID:    "existing-user",
					Email: "existing@example.com",
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := NewMockUserRepository()
			tt.setupMock(mockRepo)

			service := NewUserService(mockRepo)

			result, err := service.CreateUser(context.Background(), tt.user)

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
					// Verify default values are set
					if result.UserType == "" {
						t.Errorf("Expected UserType to be set to default 'rider'")
					}
					if result.Status == "" {
						t.Errorf("Expected Status to be set to default 'active'")
					}
				}
			}
		})
	}
}

func TestUserService_GetUser(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		expectError bool
		errorMsg    string
		expectNil   bool
		setupMock   func(*MockUserRepository)
	}{
		{
			name:        "empty_user_ID",
			userID:      "",
			expectError: true,
			errorMsg:    "user ID is required",
			setupMock:   func(m *MockUserRepository) {},
		},
		{
			name:        "user_not_found",
			userID:      "non-existent",
			expectError: false,
			expectNil:   true,
			setupMock:   func(m *MockUserRepository) {},
		},
		{
			name:        "successful_user_retrieval",
			userID:      "test-123",
			expectError: false,
			expectNil:   false,
			setupMock: func(m *MockUserRepository) {
				m.users["test-123"] = &models.User{
					ID:        "test-123",
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := NewMockUserRepository()
			tt.setupMock(mockRepo)

			service := NewUserService(mockRepo)

			result, err := service.GetUser(context.Background(), tt.userID)

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
				}
			}
		})
	}
}
