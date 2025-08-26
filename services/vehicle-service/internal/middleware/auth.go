package middleware

import "github.com/gin-gonic/gin"

// AuthMiddleware provides JWT authentication middleware.
type AuthMiddleware struct{}

// JWTAuth returns a Gin middleware handler for JWT authentication.
func (a *AuthMiddleware) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Placeholder: In production, validate JWT here
		c.Next()
	}
}
