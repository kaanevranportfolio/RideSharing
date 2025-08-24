package types

import (
	"time"
)

// PaymentMethod represents different payment options
type PaymentMethod string

const (
	PaymentMethodCreditCard    PaymentMethod = "credit_card"
	PaymentMethodDebitCard     PaymentMethod = "debit_card"
	PaymentMethodDigitalWallet PaymentMethod = "digital_wallet"
	PaymentMethodBankTransfer  PaymentMethod = "bank_transfer"
	PaymentMethodCash          PaymentMethod = "cash"
)

// PaymentStatus represents the current state of a payment
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusCompleted  PaymentStatus = "completed"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusRefunded   PaymentStatus = "refunded"
	PaymentStatusCancelled  PaymentStatus = "cancelled"
	PaymentStatusChargeback PaymentStatus = "chargeback"
)

// TransactionType defines the type of financial transaction
type TransactionType string

const (
	TransactionTypePayment       TransactionType = "payment"
	TransactionTypeRefund        TransactionType = "refund"
	TransactionTypeChargeback    TransactionType = "chargeback"
	TransactionTypeAuthorization TransactionType = "authorization"
	TransactionTypeCapture       TransactionType = "capture"
)

// FraudRiskLevel indicates the fraud detection assessment
type FraudRiskLevel string

const (
	FraudRiskLow    FraudRiskLevel = "low"
	FraudRiskMedium FraudRiskLevel = "medium"
	FraudRiskHigh   FraudRiskLevel = "high"
)

// Payment represents a payment transaction
type Payment struct {
	ID                string                 `json:"id" db:"id"`
	TripID            string                 `json:"trip_id" db:"trip_id"`
	UserID            string                 `json:"user_id" db:"user_id"`
	DriverID          string                 `json:"driver_id" db:"driver_id"`
	Amount            float64                `json:"amount" db:"amount"`
	Currency          string                 `json:"currency" db:"currency"`
	PaymentMethod     PaymentMethod          `json:"payment_method" db:"payment_method"`
	Status            PaymentStatus          `json:"status" db:"status"`
	TransactionType   TransactionType        `json:"transaction_type" db:"transaction_type"`
	ProcessorResponse string                 `json:"processor_response" db:"processor_response"`
	FraudRisk         FraudRiskLevel         `json:"fraud_risk" db:"fraud_risk"`
	FraudScores       map[string]float64     `json:"fraud_scores" db:"fraud_scores"`
	Metadata          map[string]interface{} `json:"metadata" db:"metadata"`
	FailureReason     string                 `json:"failure_reason,omitempty" db:"failure_reason"`
	ProcessedAt       *time.Time             `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

// PaymentMethod detail structure for different payment types
type PaymentMethodDetails struct {
	ID             string                 `json:"id" db:"id"`
	UserID         string                 `json:"user_id" db:"user_id"`
	Type           PaymentMethod          `json:"type" db:"type"`
	IsDefault      bool                   `json:"is_default" db:"is_default"`
	Fingerprint    string                 `json:"fingerprint" db:"fingerprint"`
	ExpiryDate     *time.Time             `json:"expiry_date,omitempty" db:"expiry_date"`
	LastFourDigits string                 `json:"last_four_digits,omitempty" db:"last_four_digits"`
	BankName       string                 `json:"bank_name,omitempty" db:"bank_name"`
	WalletProvider string                 `json:"wallet_provider,omitempty" db:"wallet_provider"`
	Details        map[string]interface{} `json:"details" db:"details"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// RefundRequest represents a refund transaction
type RefundRequest struct {
	ID          string        `json:"id" db:"id"`
	PaymentID   string        `json:"payment_id" db:"payment_id"`
	Amount      float64       `json:"amount" db:"amount"`
	Reason      string        `json:"reason" db:"reason"`
	RequestedBy string        `json:"requested_by" db:"requested_by"`
	Status      PaymentStatus `json:"status" db:"status"`
	ProcessedAt *time.Time    `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
}

// FraudDetectionResult contains fraud analysis results
type FraudDetectionResult struct {
	TransactionID  string             `json:"transaction_id"`
	RiskLevel      FraudRiskLevel     `json:"risk_level"`
	RiskScore      float64            `json:"risk_score"`
	Reasons        []string           `json:"reasons"`
	Scores         map[string]float64 `json:"scores"`
	RequiresReview bool               `json:"requires_review"`
}

// PaymentEvent represents a payment-related event
type PaymentEvent struct {
	ID        string                 `json:"id"`
	PaymentID string                 `json:"payment_id"`
	EventType string                 `json:"event_type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// Request/Response DTOs

// ProcessPaymentRequest represents a payment processing request
type ProcessPaymentRequest struct {
	TripID          string                 `json:"trip_id" validate:"required"`
	UserID          string                 `json:"user_id" validate:"required"`
	DriverID        string                 `json:"driver_id" validate:"required"`
	Amount          float64                `json:"amount" validate:"required,gt=0"`
	Currency        string                 `json:"currency" validate:"required"`
	PaymentMethodID string                 `json:"payment_method_id" validate:"required"`
	Description     string                 `json:"description"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// RefundPaymentRequest represents a refund request
type RefundPaymentRequest struct {
	PaymentID   string  `json:"payment_id" validate:"required"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Reason      string  `json:"reason" validate:"required"`
	RequestedBy string  `json:"requested_by" validate:"required"`
}

// AddPaymentMethodRequest represents adding a new payment method
type AddPaymentMethodRequest struct {
	UserID    string                 `json:"user_id" validate:"required"`
	Type      PaymentMethod          `json:"type" validate:"required"`
	Details   map[string]interface{} `json:"details" validate:"required"`
	IsDefault bool                   `json:"is_default"`
}

// PaymentResponse represents the response from payment operations
type PaymentResponse struct {
	Payment *Payment `json:"payment"`
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Errors  []string `json:"errors,omitempty"`
}

// PaymentMethodResponse represents the response for payment method operations
type PaymentMethodResponse struct {
	PaymentMethod *PaymentMethodDetails `json:"payment_method"`
	Success       bool                  `json:"success"`
	Message       string                `json:"message"`
	Errors        []string              `json:"errors,omitempty"`
}
