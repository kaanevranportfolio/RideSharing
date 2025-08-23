package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GenerateID generates a new UUID string
func GenerateID() string {
	return uuid.New().String()
}

// GenerateShortID generates a shorter ID (8 characters)
func GenerateShortID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GenerateSessionID generates a session ID
func GenerateSessionID() string {
	return GenerateID()
}

// GenerateJTI generates a JWT ID
func GenerateJTI() string {
	return GenerateShortID()
}

// GenerateCorrelationID generates a correlation ID for request tracing
func GenerateCorrelationID() string {
	return GenerateID()
}

// GenerateOrderID generates an order ID with timestamp prefix
func GenerateOrderID() string {
	timestamp := time.Now().Unix()
	shortID := GenerateShortID()
	return fmt.Sprintf("%d-%s", timestamp, shortID)
}

// GenerateTransactionID generates a transaction ID
func GenerateTransactionID() string {
	return GenerateOrderID()
}

// IsValidUUID checks if a string is a valid UUID
func IsValidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}