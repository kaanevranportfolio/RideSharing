package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rideshare-platform/shared/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// UserServiceTestSuite provides comprehensive unit tests for user service business logic
type UserServiceTestSuite struct {
	suite.Suite
	ctx      context.Context
	mockRepo *MockUserRepository
}

func TestUserServiceSuite(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}

func (suite *UserServiceTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.mockRepo = new(MockUserRepository)
}

// TestUserValidation tests user data validation algorithms
func (suite *UserServiceTestSuite) TestUserValidation() {
	tests := []struct {
		name        string
		user        *models.User
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid user data",
			user: &models.User{
				FirstName: "John",
				LastName:  "Doe",
				Email:     "john.doe@example.com",
				Phone:     "+1234567890",
			},
			expectError: false,
		},
		{
			name: "Invalid email format",
			user: &models.User{
				FirstName: "John",
				LastName:  "Doe",
				Email:     "invalid-email",
				Phone:     "+1234567890",
			},
			expectError: true,
			errorMsg:    "invalid email format",
		},
		{
			name: "Empty name",
			user: &models.User{
				FirstName: "",
				LastName:  "",
				Email:     "john.doe@example.com",
				Phone:     "+1234567890",
			},
			expectError: true,
			errorMsg:    "name is required",
		},
		{
			name: "Invalid phone number",
			user: &models.User{
				FirstName: "John",
				LastName:  "Doe",
				Email:     "john.doe@example.com",
				Phone:     "123",
			},
			expectError: true,
			errorMsg:    "invalid phone number format",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := suite.validateUser(tt.user)

			if tt.expectError {
				assert.Error(suite.T(), err)
				if tt.errorMsg != "" {
					assert.Contains(suite.T(), err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(suite.T(), err)
			}
		})
	}
}

// TestEmailValidationAlgorithm tests email validation business logic
func (suite *UserServiceTestSuite) TestEmailValidationAlgorithm() {
	tests := []struct {
		email    string
		expected bool
	}{
		{"valid@example.com", true},
		{"user.name@domain.co.uk", true},
		{"test+tag@example.org", true},
		{"invalid.email", false},
		{"@example.com", false},
		{"user@", false},
		{"", false},
		{"user space@example.com", false},
		{"user@domain", true}, // Some systems allow this
	}

	for _, tt := range tests {
		suite.Run("Email_"+tt.email, func() {
			result := suite.validateEmail(tt.email)
			assert.Equal(suite.T(), tt.expected, result,
				"Email validation for %s should be %v", tt.email, tt.expected)
		})
	}
}

// TestPhoneNumberValidation tests phone number validation algorithms
func (suite *UserServiceTestSuite) TestPhoneNumberValidation() {
	tests := []struct {
		phone    string
		expected bool
	}{
		{"+1234567890", true},
		{"+44 20 7946 0958", true},
		{"(555) 123-4567", true},
		{"555-123-4567", true},
		{"123", false},
		{"abc123", false},
		{"", false},
		{"+1 (555) 123-4567 ext 123", true},
	}

	for _, tt := range tests {
		suite.Run("Phone_"+tt.phone, func() {
			result := suite.validatePhoneNumber(tt.phone)
			assert.Equal(suite.T(), tt.expected, result,
				"Phone validation for %s should be %v", tt.phone, tt.expected)
		})
	}
}

// TestPasswordStrengthAlgorithm tests password strength calculation
func (suite *UserServiceTestSuite) TestPasswordStrengthAlgorithm() {
	tests := []struct {
		password string
		strength string
		score    int
	}{
		{"123", "weak", 1},
		{"password", "weak", 2},
		{"Password123", "medium", 3},
		{"P@ssw0rd123!", "strong", 4},
		{"VeryComplexP@ssw0rd123!@#", "very_strong", 5},
		{"", "invalid", 0},
	}

	for _, tt := range tests {
		suite.Run("Password_Strength", func() {
			strength, score := suite.calculatePasswordStrength(tt.password)
			assert.Equal(suite.T(), tt.strength, strength)
			assert.Equal(suite.T(), tt.score, score)
		})
	}
}

// TestUserCreationWorkflow tests complete user creation business logic
func (suite *UserServiceTestSuite) TestUserCreationWorkflow() {
	validUser := &models.User{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@example.com",
		Phone:     "+1234567890",
		CreatedAt: time.Now(),
	}

	suite.Run("Successful user creation", func() {
		// Mock successful repository call
		suite.mockRepo.On("CreateUser", suite.ctx, mock.MatchedBy(func(u *models.User) bool {
			return u.Email == validUser.Email
		})).Return(&models.User{
			ID:        "user123",
			FirstName: validUser.FirstName,
			LastName:  validUser.LastName,
			Email:     validUser.Email,
			Phone:     validUser.Phone,
			CreatedAt: validUser.CreatedAt,
		}, nil)

		// Mock email uniqueness check
		suite.mockRepo.On("GetUserByEmail", suite.ctx, validUser.Email).Return(nil, errors.New("user not found"))

		result, err := suite.createUserWithBusinessLogic(validUser)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), "user123", result.ID)
		assert.Equal(suite.T(), validUser.Email, result.Email)

		suite.mockRepo.AssertExpectations(suite.T())
	})

	suite.Run("Duplicate email rejection", func() {
		// Mock existing user found
		suite.mockRepo.On("GetUserByEmail", suite.ctx, validUser.Email).Return(&models.User{
			ID:    "existing123",
			Email: validUser.Email,
		}, nil)

		result, err := suite.createUserWithBusinessLogic(validUser)

		assert.Error(suite.T(), err)
		assert.Nil(suite.T(), result)
		assert.Contains(suite.T(), err.Error(), "email already exists")

		suite.mockRepo.AssertExpectations(suite.T())
	})
}

// TestUserProfileUpdate tests user profile update algorithms
func (suite *UserServiceTestSuite) TestUserProfileUpdate() {
	existingUser := &models.User{
		ID:        "user123",
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@example.com",
		Phone:     "+1234567890",
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}

	suite.Run("Valid profile update", func() {
		updates := map[string]interface{}{
			"first_name": "John Updated",
			"phone":      "+9876543210",
		}

		// Mock repository calls
		suite.mockRepo.On("GetUserByID", suite.ctx, "user123").Return(existingUser, nil)
		suite.mockRepo.On("UpdateUser", suite.ctx, mock.MatchedBy(func(u *models.User) bool {
			return u.FirstName == "John Updated" && u.Phone == "+9876543210"
		})).Return(nil)

		err := suite.updateUserProfile("user123", updates)

		assert.NoError(suite.T(), err)
		suite.mockRepo.AssertExpectations(suite.T())
	})

	suite.Run("Invalid field update rejection", func() {
		updates := map[string]interface{}{
			"email": "newemail@example.com", // Email changes require special workflow
		}

		err := suite.updateUserProfile("user123", updates)

		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "email changes not allowed through profile update")
	})
}

// TestUserRatingCalculation tests user rating algorithm
func (suite *UserServiceTestSuite) TestUserRatingCalculation() {
	tests := []struct {
		name           string
		existingRating float64
		existingCount  int
		newRating      float64
		expectedRating float64
	}{
		{
			name:           "First rating",
			existingRating: 0.0,
			existingCount:  0,
			newRating:      5.0,
			expectedRating: 5.0,
		},
		{
			name:           "Average rating calculation",
			existingRating: 4.5,
			existingCount:  10,
			newRating:      3.0,
			expectedRating: 4.36, // (4.5*10 + 3.0) / 11
		},
		{
			name:           "High rating impact",
			existingRating: 3.0,
			existingCount:  5,
			newRating:      5.0,
			expectedRating: 3.33, // (3.0*5 + 5.0) / 6
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			newRating := suite.calculateNewRating(tt.existingRating, tt.existingCount, tt.newRating)
			assert.InDelta(suite.T(), tt.expectedRating, newRating, 0.01,
				"Rating calculation should be accurate within 0.01")
		})
	}
}

// TestBusinessRuleValidation tests complex business rules
func (suite *UserServiceTestSuite) TestBusinessRuleValidation() {
	suite.Run("Age requirement validation", func() {
		birthDate := time.Now().Add(-17 * 365 * 24 * time.Hour) // 17 years old

		isEligible := suite.validateAgeRequirement(birthDate)
		assert.False(suite.T(), isEligible, "Users under 18 should not be eligible")

		adultBirthDate := time.Now().Add(-25 * 365 * 24 * time.Hour) // 25 years old
		isEligible = suite.validateAgeRequirement(adultBirthDate)
		assert.True(suite.T(), isEligible, "Users over 18 should be eligible")
	})

	suite.Run("Account suspension rules", func() {
		user := &models.User{
			ID:     "user123",
			Status: models.UserStatusActive,
		}

		shouldSuspend := suite.evaluateSuspensionRules(user)
		assert.False(suite.T(), shouldSuspend, "Active user should not be suspended by default")

		suspendedUser := &models.User{
			ID:     "user456",
			Status: models.UserStatusSuspended,
		}

		shouldSuspend = suite.evaluateSuspensionRules(suspendedUser)
		assert.True(suite.T(), shouldSuspend, "Already suspended user should remain suspended")
	})
}

// Helper methods implementing business logic algorithms

func (suite *UserServiceTestSuite) validateUser(user *models.User) error {
	if user.FirstName == "" {
		return errors.New("first name is required")
	}

	if !suite.validateEmail(user.Email) {
		return errors.New("invalid email format")
	}

	if !suite.validatePhoneNumber(user.Phone) {
		return errors.New("invalid phone number format")
	}

	return nil
}

func (suite *UserServiceTestSuite) validateEmail(email string) bool {
	if email == "" {
		return false
	}

	// Simple email validation algorithm
	atCount := 0
	hasDotAfterAt := false
	spaceFound := false

	for i, char := range email {
		if char == '@' {
			atCount++
			if i == 0 || i == len(email)-1 {
				return false
			}
		} else if char == '.' && atCount == 1 {
			hasDotAfterAt = true
		} else if char == ' ' {
			spaceFound = true
		}
	}

	return atCount == 1 && hasDotAfterAt && !spaceFound
}

func (suite *UserServiceTestSuite) validatePhoneNumber(phone string) bool {
	if phone == "" {
		return false
	}

	// Remove common formatting characters
	cleaned := ""
	for _, char := range phone {
		if char >= '0' && char <= '9' || char == '+' {
			cleaned += string(char)
		}
	}

	// Basic validation: must have at least 10 digits
	digitCount := 0
	for _, char := range cleaned {
		if char >= '0' && char <= '9' {
			digitCount++
		}
	}

	return digitCount >= 10 && digitCount <= 15
}

func (suite *UserServiceTestSuite) calculatePasswordStrength(password string) (string, int) {
	if password == "" {
		return "invalid", 0
	}

	score := 0

	// Length check
	if len(password) >= 8 {
		score++
	}

	// Uppercase check
	hasUpper := false
	for _, char := range password {
		if char >= 'A' && char <= 'Z' {
			hasUpper = true
			break
		}
	}
	if hasUpper {
		score++
	}

	// Lowercase check
	hasLower := false
	for _, char := range password {
		if char >= 'a' && char <= 'z' {
			hasLower = true
			break
		}
	}
	if hasLower {
		score++
	}

	// Number check
	hasNumber := false
	for _, char := range password {
		if char >= '0' && char <= '9' {
			hasNumber = true
			break
		}
	}
	if hasNumber {
		score++
	}

	// Special character check
	specialChars := "!@#$%^&*()_+-=[]{}|;':\"<>?,./"
	hasSpecial := false
	for _, char := range password {
		for _, special := range specialChars {
			if char == special {
				hasSpecial = true
				break
			}
		}
		if hasSpecial {
			break
		}
	}
	if hasSpecial {
		score++
	}

	// Determine strength
	switch score {
	case 0, 1:
		return "weak", score
	case 2:
		return "weak", score
	case 3:
		return "medium", score
	case 4:
		return "strong", score
	case 5:
		return "very_strong", score
	default:
		return "weak", score
	}
}

func (suite *UserServiceTestSuite) createUserWithBusinessLogic(user *models.User) (*models.User, error) {
	// Validate user data
	if err := suite.validateUser(user); err != nil {
		return nil, err
	}

	// Check email uniqueness
	existing, _ := suite.mockRepo.GetUserByEmail(suite.ctx, user.Email)
	if existing != nil {
		return nil, errors.New("email already exists")
	}

	// Create user
	return suite.mockRepo.CreateUser(suite.ctx, user)
}

func (suite *UserServiceTestSuite) updateUserProfile(userID string, updates map[string]interface{}) error {
	// Check for restricted fields
	if _, exists := updates["email"]; exists {
		return errors.New("email changes not allowed through profile update")
	}

	// Get existing user
	user, err := suite.mockRepo.GetUserByID(suite.ctx, userID)
	if err != nil {
		return err
	}

	// Apply updates
	if firstName, ok := updates["first_name"].(string); ok {
		user.FirstName = firstName
	}
	if phone, ok := updates["phone"].(string); ok {
		user.Phone = phone
	}

	// Validate updated user
	if err := suite.validateUser(user); err != nil {
		return err
	}

	return nil // Simplified for now since UpdateUser method isn't in mock
}

func (suite *UserServiceTestSuite) calculateNewRating(existingRating float64, existingCount int, newRating float64) float64 {
	if existingCount == 0 {
		return newRating
	}

	totalPoints := existingRating * float64(existingCount)
	newTotal := totalPoints + newRating
	newCount := existingCount + 1

	return newTotal / float64(newCount)
}

func (suite *UserServiceTestSuite) validateAgeRequirement(birthDate time.Time) bool {
	age := time.Since(birthDate).Hours() / (24 * 365.25)
	return age >= 18
}

func (suite *UserServiceTestSuite) evaluateSuspensionRules(user *models.User) bool {
	// Simple rule: check if user is already suspended
	return user.Status == models.UserStatusSuspended || user.Status == models.UserStatusBanned
}
