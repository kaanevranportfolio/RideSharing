package service

import (
	"context"
	"errors"

	"github.com/rideshare-platform/shared/models"
)

// UserService handles user business logic
type UserService struct {
	repo UserRepositoryInterface
}

// NewUserService creates a new user service
func NewUserService(repo UserRepositoryInterface) *UserService {
	return &UserService{
		repo: repo,
	}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	if user.Email == "" {
		return nil, errors.New("user email is required")
	}

	if user.FirstName == "" {
		return nil, errors.New("user first name is required")
	}

	if user.LastName == "" {
		return nil, errors.New("user last name is required")
	}

	// Check if user already exists by email
	existingUser, err := s.repo.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Set defaults
	if user.UserType == "" {
		user.UserType = "rider"
	}
	if user.Status == "" {
		user.Status = "active"
	}

	return s.repo.CreateUser(ctx, user)
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	return s.repo.GetUser(ctx, userID)
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}

	return s.repo.GetUserByEmail(ctx, email)
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	if user.ID == "" {
		return nil, errors.New("user ID is required")
	}

	// Check if user exists
	existingUser, err := s.repo.GetUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if existingUser == nil {
		return nil, errors.New("user not found")
	}

	// Update fields
	if user.Email != "" {
		existingUser.Email = user.Email
	}
	if user.FirstName != "" {
		existingUser.FirstName = user.FirstName
	}
	if user.LastName != "" {
		existingUser.LastName = user.LastName
	}
	if user.Phone != "" {
		existingUser.Phone = user.Phone
	}
	if user.Status != "" {
		existingUser.Status = user.Status
	}
	if user.ProfileImageURL != "" {
		existingUser.ProfileImageURL = user.ProfileImageURL
	}

	return s.repo.UpdateUser(ctx, existingUser)
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("user ID is required")
	}

	return s.repo.DeleteUser(ctx, userID)
}

// ListUsers lists all users
func (s *UserService) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.ListUsers(ctx, limit, offset)
}

// AuthenticateUser authenticates a user by email and password
func (s *UserService) AuthenticateUser(ctx context.Context, email, password string) (*models.User, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}
	if password == "" {
		return nil, errors.New("password is required")
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	// TODO: Implement proper password hashing verification
	// For now, just return the user (this is not secure!)
	return user, nil
}
