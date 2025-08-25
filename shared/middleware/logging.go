package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rideshare-platform/shared/logger"
)

// LoggingMiddleware provides request logging middleware
type LoggingMiddleware struct {
	logger *logger.Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(log *logger.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: log,
	}
}

// RequestLogger logs HTTP requests and responses
func (l *LoggingMiddleware) RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID
		requestID := uuid.New().String()

		// Add request ID to context
		ctx := context.WithValue(c.Request.Context(), logger.RequestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		// Set request ID in response header
		c.Header("X-Request-ID", requestID)

		// Record start time
		start := time.Now()

		// Capture request body if needed (for debugging)
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Create response writer wrapper to capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Log request details
		l.logger.LogRequest(
			c.Request.Context(),
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
		)

		// Log additional details for errors
		if c.Writer.Status() >= 400 {
			l.logger.WithContext(c.Request.Context()).WithFields(logger.Fields{
				"method":        c.Request.Method,
				"path":          c.Request.URL.Path,
				"status_code":   c.Writer.Status(),
				"duration_ms":   duration.Milliseconds(),
				"request_body":  string(requestBody),
				"response_body": writer.body.String(),
				"user_agent":    c.Request.UserAgent(),
				"remote_addr":   c.ClientIP(),
				"query_params":  c.Request.URL.RawQuery,
			}).Error("HTTP request failed")
		}
	}
}

// CorrelationID adds correlation ID to requests
func (l *LoggingMiddleware) CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if correlation ID exists in header
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Add correlation ID to context
		ctx := context.WithValue(c.Request.Context(), logger.CorrelationIDKey, correlationID)
		c.Request = c.Request.WithContext(ctx)

		// Set correlation ID in response header
		c.Header("X-Correlation-ID", correlationID)

		c.Next()
	}
}

// Recovery handles panics and logs them
func (l *LoggingMiddleware) Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		l.logger.WithContext(c.Request.Context()).WithFields(logger.Fields{
			"panic_value": recovered,
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"user_agent":  c.Request.UserAgent(),
			"remote_addr": c.ClientIP(),
		}).Error("Panic recovered")

		c.AbortWithStatus(500)
	})
}

// responseWriter wraps gin.ResponseWriter to capture response body
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// SecurityHeaders adds security headers to responses
func (l *LoggingMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// CORS handles Cross-Origin Resource Sharing
func (l *LoggingMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Allow specific origins (configure as needed)
		allowedOrigins := []string{
			"http://frontend:3000",
			"http://frontend:3001",
			"https://rideshare-app.com",
		}

		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Correlation-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, X-Correlation-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RateLimiting provides basic rate limiting (simple in-memory implementation)
func (l *LoggingMiddleware) RateLimiting() gin.HandlerFunc {
	// This is a simple implementation - in production, use Redis-based rate limiting
	clients := make(map[string][]time.Time)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		// Clean old entries (older than 1 minute)
		if requests, exists := clients[clientIP]; exists {
			var validRequests []time.Time
			for _, reqTime := range requests {
				if now.Sub(reqTime) < time.Minute {
					validRequests = append(validRequests, reqTime)
				}
			}
			clients[clientIP] = validRequests
		}

		// Check rate limit (100 requests per minute)
		if len(clients[clientIP]) >= 100 {
			l.logger.WithContext(c.Request.Context()).WithFields(logger.Fields{
				"client_ip":     clientIP,
				"request_count": len(clients[clientIP]),
			}).Warn("Rate limit exceeded")

			c.JSON(429, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		// Add current request
		clients[clientIP] = append(clients[clientIP], now)

		c.Next()
	}
}

// RequestSize limits request body size
func (l *LoggingMiddleware) RequestSize(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			l.logger.WithContext(c.Request.Context()).WithFields(logger.Fields{
				"content_length": c.Request.ContentLength,
				"max_size":       maxSize,
			}).Warn("Request body too large")

			c.JSON(413, gin.H{"error": "Request body too large"})
			c.Abort()
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}

// Timeout adds request timeout
func (l *LoggingMiddleware) Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			c.Next()
			done <- struct{}{}
		}()

		select {
		case <-done:
			// Request completed normally
		case <-ctx.Done():
			l.logger.WithContext(c.Request.Context()).WithFields(logger.Fields{
				"timeout": timeout,
				"method":  c.Request.Method,
				"path":    c.Request.URL.Path,
			}).Warn("Request timeout")

			c.JSON(408, gin.H{"error": "Request timeout"})
			c.Abort()
		}
	}
}

// HealthCheck provides a simple health check endpoint
func (l *LoggingMiddleware) HealthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" {
			c.JSON(200, gin.H{
				"status":    "healthy",
				"timestamp": time.Now().UTC(),
				"service":   "rideshare-platform",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
