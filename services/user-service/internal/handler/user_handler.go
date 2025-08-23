package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rideshare-platform/services/user-service/internal/service"
	"github.com/rideshare-platform/shared/models"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// RegisterRoutes registers user routes
func (h *UserHandler) RegisterRoutes(router *gin.Engine) {
	// Health check endpoint
	router.GET("/health", h.healthCheck)

	users := router.Group("/api/v1/users")
	{
		users.POST("/", h.CreateUser)
		users.GET("/:id", h.GetUser)
		users.PUT("/:id", h.UpdateUser)
		users.DELETE("/:id", h.DeleteUser)
		users.GET("/", h.ListUsers)
		users.POST("/auth", h.AuthenticateUser)
	}
}

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Email     string          `json:"email" binding:"required"`
	Phone     string          `json:"phone" binding:"required"`
	FirstName string          `json:"first_name" binding:"required"`
	LastName  string          `json:"last_name" binding:"required"`
	UserType  models.UserType `json:"user_type" binding:"required"`
	Password  string          `json:"password" binding:"required"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Email     string            `json:"email"`
	Phone     string            `json:"phone"`
	FirstName string            `json:"first_name"`
	LastName  string            `json:"last_name"`
	UserType  models.UserType   `json:"user_type"`
	Status    models.UserStatus `json:"status"`
}

// AuthRequest represents the authentication request
type AuthRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Create user model
	user := models.NewUser(req.Email, req.Phone, req.FirstName, req.LastName, req.UserType)

	// Create user
	createdUser, err := h.userService.CreateUser(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to create user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, createdUser)
}

// GetUser retrieves a user by ID
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "User not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser updates an existing user
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Create user model with update data
	user := &models.User{
		ID:        userID,
		Email:     req.Email,
		Phone:     req.Phone,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		UserType:  req.UserType,
		Status:    req.Status,
	}

	updatedUser, err := h.userService.UpdateUser(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to update user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}

// DeleteUser deletes a user by ID
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	err := h.userService.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Failed to delete user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}

// ListUsers returns all users
func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.userService.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list users",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}

// AuthenticateUser authenticates a user with email and password
func (h *UserHandler) AuthenticateUser(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	user, err := h.userService.AuthenticateUser(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Authentication failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Authentication successful",
		"user":    user,
	})
}

// healthCheck returns the health status of the service
func (h *UserHandler) healthCheck(c *gin.Context) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "user-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
	}

	c.JSON(http.StatusOK, response)
}
