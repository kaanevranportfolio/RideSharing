package service

import (
	"context"
	"errors"
	"time"

	"github.com/rideshare-platform/services/user-service/internal/config"
	"github.com/rideshare-platform/shared/models"
)

// UserService handles user business logic
type UserService struct {
	config *config.Config
	users  map[string]*models.User // In-memory storage for demo
}

// NewUserService creates a new user service
func NewUserService(config *config.Config) *UserService {
	return &UserService{
		config: config,
		users:  make(map[string]*models.User),
	}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	if user.ID == "" {
		return nil, errors.New("user ID is required")
	}

	if user.Email == "" {
		return nil, errors.New("user email is required")
	}

	// Check if user already exists
	if _, exists := s.users[user.ID]; exists {
		return nil, errors.New("user already exists")
	}

	// Set timestamps
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Store user
	s.users[user.ID] = user

	return user, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	user, exists := s.users[userID]
	if !exists {
		return nil, errors.New("user not found")
	}

	return user, nil
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	if user.ID == "" {
		return nil, errors.New("user ID is required")
	}

	existingUser, exists := s.users[user.ID]
	if !exists {
		return nil, errors.New("user not found")
	}

	// Update fields
	if user.FirstName != "" {
		existingUser.FirstName = user.FirstName
	}
	if user.LastName != "" {
		existingUser.LastName = user.LastName
	}
	if user.Email != "" {
		existingUser.Email = user.Email
	}
	if user.Phone != "" {
		existingUser.Phone = user.Phone
	}
	if user.UserType != "" {
		existingUser.UserType = user.UserType
	}
	if user.Status != "" {
		existingUser.Status = user.Status
	}

	// Update timestamp
	existingUser.UpdatedAt = time.Now()

	return existingUser, nil
}

// DeleteUser deletes a user by ID
func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("user ID is required")
	}

	if _, exists := s.users[userID]; !exists {
		return errors.New("user not found")
	}

	delete(s.users, userID)
	return nil
}

// ListUsers returns all users
func (s *UserService) ListUsers(ctx context.Context) ([]*models.User, error) {
	users := make([]*models.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users, nil
}

// AuthenticateUser authenticates a user with email and password
func (s *UserService) AuthenticateUser(ctx context.Context, email, password string) (*models.User, error) {
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	// Find user by email
	for _, user := range s.users {
		if user.Email == email {
			// In a real implementation, you would hash and compare passwords
			// For demo purposes, we'll just return the user
			return user, nil
		}
	}

	return nil, errors.New("invalid credentials")
}
