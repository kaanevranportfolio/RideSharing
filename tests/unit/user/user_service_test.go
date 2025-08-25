package user_test

import (
	"context"
	"testing"
	"time"

	"github.com/rideshare-platform/shared/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of the user repository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// UserService interface for testing
type UserService interface {
	CreateUser(ctx context.Context, user *models.User) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	AuthenticateUser(ctx context.Context, email, password string) (*models.User, error)
}

// MockUserService for testing business logic
type MockUserService struct {
	repo *MockUserRepository
}

func NewMockUserService(repo *MockUserRepository) *MockUserService {
	return &MockUserService{repo: repo}
}

func (s *MockUserService) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	// Validate user input
	if user.Email == "" {
		return nil, assert.AnError
	}
	if user.UserType != models.UserTypeRider && user.UserType != models.UserTypeDriver {
		return nil, assert.AnError
	}

	// Set default values
	user.ID = "generated-id"
	user.Status = models.UserStatusActive
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	return s.repo.CreateUser(ctx, user)
}

func (s *MockUserService) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	if id == "" {
		return nil, assert.AnError
	}
	return s.repo.GetUserByID(ctx, id)
}

func (s *MockUserService) AuthenticateUser(ctx context.Context, email, password string) (*models.User, error) {
	if email == "" || password == "" {
		return nil, assert.AnError
	}
	return s.repo.GetUserByEmail(ctx, email)
}

func TestUserService_CreateUser(t *testing.T) {
	tests := []struct {
		name          string
		user          *models.User
		setupMock     func(*MockUserRepository)
		expectedError bool
		errorContains string
	}{
		{
			name: "successful user creation",
			user: &models.User{
				Email:     "test@example.com",
				Phone:     "+1234567890",
				FirstName: "Test",
				LastName:  "User",
				UserType:  models.UserTypeRider,
			},
			setupMock: func(m *MockUserRepository) {
				expectedUser := &models.User{
					ID:        "user123",
					Email:     "test@example.com",
					Phone:     "+1234567890",
					FirstName: "Test",
					LastName:  "User",
					UserType:  models.UserTypeRider,
					Status:    models.UserStatusActive,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(expectedUser, nil)
			},
			expectedError: false,
		},
		{
			name: "empty email error",
			user: &models.User{
				Phone:     "+1234567890",
				FirstName: "Test",
				LastName:  "User",
				UserType:  models.UserTypeRider,
			},
			setupMock: func(m *MockUserRepository) {
				// No mock setup needed as validation should fail before repository call
			},
			expectedError: true,
		},
		{
			name: "invalid user type",
			user: &models.User{
				Email:     "test@example.com",
				Phone:     "+1234567890",
				FirstName: "Test",
				LastName:  "User",
				UserType:  "invalid_type",
			},
			setupMock: func(m *MockUserRepository) {
				// No mock setup needed as validation should fail before repository call
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockUserRepository)
			tt.setupMock(mockRepo)

			userService := NewMockUserService(mockRepo)
			ctx := context.Background()

			// Execute
			result, err := userService.CreateUser(ctx, tt.user)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.user.Email, result.Email)
				assert.Equal(t, tt.user.FirstName, result.FirstName)
				assert.Equal(t, tt.user.LastName, result.LastName)
				assert.Equal(t, tt.user.UserType, result.UserType)
				assert.NotEmpty(t, result.ID)
			}

			// Verify all expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetUserByID(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		setupMock     func(*MockUserRepository)
		expectedError bool
	}{
		{
			name:   "successful user retrieval",
			userID: "user123",
			setupMock: func(m *MockUserRepository) {
				expectedUser := &models.User{
					ID:        "user123",
					Email:     "test@example.com",
					FirstName: "Test",
					LastName:  "User",
					UserType:  models.UserTypeRider,
					Status:    models.UserStatusActive,
				}
				m.On("GetUserByID", mock.Anything, "user123").Return(expectedUser, nil)
			},
			expectedError: false,
		},
		{
			name:   "user not found",
			userID: "nonexistent",
			setupMock: func(m *MockUserRepository) {
				m.On("GetUserByID", mock.Anything, "nonexistent").Return(nil, assert.AnError)
			},
			expectedError: true,
		},
		{
			name:   "empty user ID",
			userID: "",
			setupMock: func(m *MockUserRepository) {
				// No mock setup needed as validation should fail before repository call
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockUserRepository)
			tt.setupMock(mockRepo)

			userService := NewMockUserService(mockRepo)
			ctx := context.Background()

			// Execute
			result, err := userService.GetUserByID(ctx, tt.userID)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.userID, result.ID)
			}

			// Verify all expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkUserService_CreateUser(b *testing.B) {
	mockRepo := new(MockUserRepository)
	user := &models.User{
		Email:     "bench@example.com",
		Phone:     "+1234567890",
		FirstName: "Bench",
		LastName:  "User",
		UserType:  models.UserTypeRider,
	}

	expectedUser := &models.User{
		ID:        "user123",
		Email:     "bench@example.com",
		Phone:     "+1234567890",
		FirstName: "Bench",
		LastName:  "User",
		UserType:  models.UserTypeRider,
		Status:    models.UserStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(expectedUser, nil)

	userService := NewMockUserService(mockRepo)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userService.CreateUser(ctx, user)
	}
}
