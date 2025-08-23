package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/rideshare-platform/shared/logger"
)

// AuthClaims represents JWT claims
type AuthClaims struct {
	UserID   string `json:"user_id"`
	UserType string `json:"user_type"` // "rider" or "driver"
	Email    string `json:"email"`
	jwt.StandardClaims
}

// AuthMiddleware provides authentication middleware
type AuthMiddleware struct {
	jwtSecret []byte
	logger    *logger.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtSecret string, log *logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: []byte(jwtSecret),
		logger:    log,
	}
}

// JWTAuth validates JWT tokens
func (a *AuthMiddleware) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			a.logger.WithContext(c.Request.Context()).Warn("Missing authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			a.logger.WithContext(c.Request.Context()).Warn("Invalid authorization header format")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return a.jwtSecret, nil
		})

		if err != nil {
			a.logger.WithContext(c.Request.Context()).WithError(err).Warn("Invalid JWT token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*AuthClaims)
		if !ok || !token.Valid {
			a.logger.WithContext(c.Request.Context()).Warn("Invalid JWT claims")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Check token expiration
		if claims.ExpiresAt < time.Now().Unix() {
			a.logger.WithContext(c.Request.Context()).Warn("JWT token expired")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			c.Abort()
			return
		}

		// Add user info to context
		ctx := context.WithValue(c.Request.Context(), logger.UserIDKey, claims.UserID)
		c.Request = c.Request.WithContext(ctx)

		// Set user info in Gin context
		c.Set("user_id", claims.UserID)
		c.Set("user_type", claims.UserType)
		c.Set("email", claims.Email)

		a.logger.WithContext(c.Request.Context()).WithFields(logger.Fields{
			"user_id":   claims.UserID,
			"user_type": claims.UserType,
		}).Debug("User authenticated successfully")

		c.Next()
	}
}

// RequireUserType ensures the user has the required type
func (a *AuthMiddleware) RequireUserType(userType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUserType, exists := c.Get("user_type")
		if !exists {
			a.logger.WithContext(c.Request.Context()).Error("User type not found in context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication error"})
			c.Abort()
			return
		}

		if currentUserType != userType {
			a.logger.WithContext(c.Request.Context()).WithFields(logger.Fields{
				"required_type": userType,
				"actual_type":   currentUserType,
			}).Warn("Insufficient permissions")
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth validates JWT tokens but doesn't require them
func (a *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := tokenParts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return a.jwtSecret, nil
		})

		if err != nil {
			c.Next()
			return
		}

		claims, ok := token.Claims.(*AuthClaims)
		if !ok || !token.Valid {
			c.Next()
			return
		}

		// Check token expiration
		if claims.ExpiresAt < time.Now().Unix() {
			c.Next()
			return
		}

		// Add user info to context
		ctx := context.WithValue(c.Request.Context(), logger.UserIDKey, claims.UserID)
		c.Request = c.Request.WithContext(ctx)

		// Set user info in Gin context
		c.Set("user_id", claims.UserID)
		c.Set("user_type", claims.UserType)
		c.Set("email", claims.Email)

		c.Next()
	}
}

// GenerateToken generates a JWT token for a user
func (a *AuthMiddleware) GenerateToken(userID, userType, email string, expirationHours int) (string, error) {
	claims := &AuthClaims{
		UserID:   userID,
		UserType: userType,
		Email:    email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * time.Duration(expirationHours)).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "rideshare-platform",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns claims
func (a *AuthMiddleware) ValidateToken(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*AuthClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check token expiration
	if claims.ExpiresAt < time.Now().Unix() {
		return nil, fmt.Errorf("token expired")
	}

	return claims, nil
}

// RefreshToken generates a new token with extended expiration
func (a *AuthMiddleware) RefreshToken(tokenString string, expirationHours int) (string, error) {
	claims, err := a.ValidateToken(tokenString)
	if err != nil {
		return "", fmt.Errorf("invalid token for refresh: %w", err)
	}

	// Generate new token with extended expiration
	return a.GenerateToken(claims.UserID, claims.UserType, claims.Email, expirationHours)
}

// GetUserFromContext extracts user information from Gin context
func GetUserFromContext(c *gin.Context) (userID, userType, email string, ok bool) {
	userIDVal, exists1 := c.Get("user_id")
	userTypeVal, exists2 := c.Get("user_type")
	emailVal, exists3 := c.Get("email")

	if !exists1 || !exists2 || !exists3 {
		return "", "", "", false
	}

	userID, ok1 := userIDVal.(string)
	userType, ok2 := userTypeVal.(string)
	email, ok3 := emailVal.(string)

	return userID, userType, email, ok1 && ok2 && ok3
}

// IsDriver checks if the current user is a driver
func IsDriver(c *gin.Context) bool {
	userType, exists := c.Get("user_type")
	if !exists {
		return false
	}
	return userType == "driver"
}

// IsRider checks if the current user is a rider
func IsRider(c *gin.Context) bool {
	userType, exists := c.Get("user_type")
	if !exists {
		return false
	}
	return userType == "rider"
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	id, ok := userID.(string)
	return id, ok
}
