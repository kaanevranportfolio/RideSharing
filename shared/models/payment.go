package models

import (
	"time"
)

// PaymentMethodType represents the type of payment method
type PaymentMethodType string

const (
	PaymentMethodCreditCard    PaymentMethodType = "credit_card"
	PaymentMethodDebitCard     PaymentMethodType = "debit_card"
	PaymentMethodDigitalWallet PaymentMethodType = "digital_wallet"
	PaymentMethodCash          PaymentMethodType = "cash"
	PaymentMethodBankTransfer  PaymentMethodType = "bank_transfer"
)

// PaymentStatus represents the current status of a payment
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusCompleted  PaymentStatus = "completed"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusCancelled  PaymentStatus = "cancelled"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

// PaymentMethod represents a payment method for a user
type PaymentMethod struct {
	ID                      string            `json:"id" db:"id"`
	UserID                  string            `json:"user_id" db:"user_id"`
	Type                    PaymentMethodType `json:"type" db:"type"`
	Provider                *string           `json:"provider" db:"provider"`
	ProviderPaymentMethodID *string           `json:"provider_payment_method_id" db:"provider_payment_method_id"`
	LastFour                string            `json:"last_four" db:"last_four"`
	Brand                   string            `json:"brand" db:"brand"`
	IsDefault               bool              `json:"is_default" db:"is_default"`
	ExpiresAt               *time.Time        `json:"expires_at" db:"expires_at"`
	BillingAddress          map[string]string `json:"billing_address" db:"billing_address"`
	CreatedAt               time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time         `json:"updated_at" db:"updated_at"`
}

// Payment represents a payment transaction
type Payment struct {
	ID                   string                 `json:"id" db:"id"`
	TripID               string                 `json:"trip_id" db:"trip_id"`
	UserID               string                 `json:"user_id" db:"user_id"`
	PaymentMethodID      *string                `json:"payment_method_id" db:"payment_method_id"`
	AmountCents          int64                  `json:"amount_cents" db:"amount_cents"`
	Currency             string                 `json:"currency" db:"currency"`
	Status               PaymentStatus          `json:"status" db:"status"`
	GatewayProvider      *string                `json:"gateway_provider" db:"gateway_provider"`
	GatewayTransactionID *string                `json:"gateway_transaction_id" db:"gateway_transaction_id"`
	GatewayResponse      map[string]interface{} `json:"gateway_response" db:"gateway_response"`
	FailureCode          *string                `json:"failure_code" db:"failure_code"`
	FailureReason        *string                `json:"failure_reason" db:"failure_reason"`
	RefundedAmountCents  int64                  `json:"refunded_amount_cents" db:"refunded_amount_cents"`
	RefundReason         *string                `json:"refund_reason" db:"refund_reason"`
	CreatedAt            time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at" db:"updated_at"`
	ProcessedAt          *time.Time             `json:"processed_at" db:"processed_at"`
	FailedAt             *time.Time             `json:"failed_at" db:"failed_at"`
}

// NewPaymentMethod creates a new payment method
func NewPaymentMethod(userID string, methodType PaymentMethodType, lastFour, brand string) *PaymentMethod {
	return &PaymentMethod{
		ID:             generateID(),
		UserID:         userID,
		Type:           methodType,
		LastFour:       lastFour,
		Brand:          brand,
		IsDefault:      false,
		BillingAddress: make(map[string]string),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// NewPayment creates a new payment
func NewPayment(tripID, userID string, amountCents int64, currency string) *Payment {
	return &Payment{
		ID:                  generateID(),
		TripID:              tripID,
		UserID:              userID,
		AmountCents:         amountCents,
		Currency:            currency,
		Status:              PaymentStatusPending,
		RefundedAmountCents: 0,
		GatewayResponse:     make(map[string]interface{}),
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
}

// IsActive returns true if the payment method is active
func (pm *PaymentMethod) IsActive() bool {
	if pm.ExpiresAt != nil && time.Now().After(*pm.ExpiresAt) {
		return false
	}
	return true
}

// SetAsDefault sets this payment method as the default for the user
func (pm *PaymentMethod) SetAsDefault() {
	pm.IsDefault = true
	pm.UpdatedAt = time.Now()
}

// UnsetAsDefault removes the default status from this payment method
func (pm *PaymentMethod) UnsetAsDefault() {
	pm.IsDefault = false
	pm.UpdatedAt = time.Now()
}

// SetProvider sets the payment provider information
func (pm *PaymentMethod) SetProvider(provider, providerPaymentMethodID string) {
	pm.Provider = &provider
	pm.ProviderPaymentMethodID = &providerPaymentMethodID
	pm.UpdatedAt = time.Now()
}

// SetBillingAddress sets the billing address
func (pm *PaymentMethod) SetBillingAddress(address map[string]string) {
	pm.BillingAddress = address
	pm.UpdatedAt = time.Now()
}

// SetExpiry sets the expiry date for the payment method
func (pm *PaymentMethod) SetExpiry(expiresAt time.Time) {
	pm.ExpiresAt = &expiresAt
	pm.UpdatedAt = time.Now()
}

// GetDisplayName returns a display name for the payment method
func (pm *PaymentMethod) GetDisplayName() string {
	switch pm.Type {
	case PaymentMethodCreditCard, PaymentMethodDebitCard:
		return pm.Brand + " ending in " + pm.LastFour
	case PaymentMethodDigitalWallet:
		return string(pm.Type) + " (" + pm.Brand + ")"
	case PaymentMethodCash:
		return "Cash"
	case PaymentMethodBankTransfer:
		return "Bank Transfer"
	default:
		return string(pm.Type)
	}
}

// IsPending returns true if the payment is pending
func (p *Payment) IsPending() bool {
	return p.Status == PaymentStatusPending
}

// IsProcessing returns true if the payment is processing
func (p *Payment) IsProcessing() bool {
	return p.Status == PaymentStatusProcessing
}

// IsCompleted returns true if the payment is completed
func (p *Payment) IsCompleted() bool {
	return p.Status == PaymentStatusCompleted
}

// IsFailed returns true if the payment failed
func (p *Payment) IsFailed() bool {
	return p.Status == PaymentStatusFailed
}

// IsCancelled returns true if the payment was cancelled
func (p *Payment) IsCancelled() bool {
	return p.Status == PaymentStatusCancelled
}

// IsRefunded returns true if the payment was refunded
func (p *Payment) IsRefunded() bool {
	return p.Status == PaymentStatusRefunded
}

// GetAmount returns the payment amount as Money
func (p *Payment) GetAmount() Money {
	return Money{
		Amount:   p.AmountCents,
		Currency: p.Currency,
	}
}

// GetRefundedAmount returns the refunded amount as Money
func (p *Payment) GetRefundedAmount() Money {
	return Money{
		Amount:   p.RefundedAmountCents,
		Currency: p.Currency,
	}
}

// GetNetAmount returns the net amount (amount - refunded) as Money
func (p *Payment) GetNetAmount() Money {
	return Money{
		Amount:   p.AmountCents - p.RefundedAmountCents,
		Currency: p.Currency,
	}
}

// UpdateStatus updates the payment status
func (p *Payment) UpdateStatus(status PaymentStatus) {
	p.Status = status
	p.UpdatedAt = time.Now()

	now := time.Now()
	switch status {
	case PaymentStatusCompleted:
		p.ProcessedAt = &now
	case PaymentStatusFailed:
		p.FailedAt = &now
	}
}

// SetPaymentMethod sets the payment method for this payment
func (p *Payment) SetPaymentMethod(paymentMethodID string) {
	p.PaymentMethodID = &paymentMethodID
	p.UpdatedAt = time.Now()
}

// SetGatewayInfo sets the payment gateway information
func (p *Payment) SetGatewayInfo(provider, transactionID string, response map[string]interface{}) {
	p.GatewayProvider = &provider
	p.GatewayTransactionID = &transactionID
	p.GatewayResponse = response
	p.UpdatedAt = time.Now()
}

// SetFailure sets the failure information
func (p *Payment) SetFailure(code, reason string) {
	p.FailureCode = &code
	p.FailureReason = &reason
	p.UpdateStatus(PaymentStatusFailed)
}

// AddRefund adds a refund amount to the payment
func (p *Payment) AddRefund(refundAmountCents int64, reason string) {
	p.RefundedAmountCents += refundAmountCents
	p.RefundReason = &reason

	// If fully refunded, update status
	if p.RefundedAmountCents >= p.AmountCents {
		p.UpdateStatus(PaymentStatusRefunded)
	}

	p.UpdatedAt = time.Now()
}

// CanBeRefunded returns true if the payment can be refunded
func (p *Payment) CanBeRefunded() bool {
	return p.IsCompleted() && p.RefundedAmountCents < p.AmountCents
}

// GetRefundableAmount returns the amount that can still be refunded
func (p *Payment) GetRefundableAmount() int64 {
	if !p.CanBeRefunded() {
		return 0
	}
	return p.AmountCents - p.RefundedAmountCents
}

// IsPartiallyRefunded returns true if the payment is partially refunded
func (p *Payment) IsPartiallyRefunded() bool {
	return p.RefundedAmountCents > 0 && p.RefundedAmountCents < p.AmountCents
}

// IsFullyRefunded returns true if the payment is fully refunded
func (p *Payment) IsFullyRefunded() bool {
	return p.RefundedAmountCents >= p.AmountCents
}

// GetProcessingDuration returns the duration it took to process the payment
func (p *Payment) GetProcessingDuration() *time.Duration {
	if p.ProcessedAt == nil {
		return nil
	}

	duration := p.ProcessedAt.Sub(p.CreatedAt)
	return &duration
}

// IsValidPaymentMethodType checks if a payment method type is valid
func IsValidPaymentMethodType(methodType string) bool {
	validTypes := []PaymentMethodType{
		PaymentMethodCreditCard,
		PaymentMethodDebitCard,
		PaymentMethodDigitalWallet,
		PaymentMethodCash,
		PaymentMethodBankTransfer,
	}

	for _, validType := range validTypes {
		if PaymentMethodType(methodType) == validType {
			return true
		}
	}
	return false
}

// IsValidPaymentStatus checks if a payment status is valid
func IsValidPaymentStatus(status string) bool {
	validStatuses := []PaymentStatus{
		PaymentStatusPending,
		PaymentStatusProcessing,
		PaymentStatusCompleted,
		PaymentStatusFailed,
		PaymentStatusCancelled,
		PaymentStatusRefunded,
	}

	for _, validStatus := range validStatuses {
		if PaymentStatus(status) == validStatus {
			return true
		}
	}
	return false
}

// GetPaymentMethodTypes returns all valid payment method types
func GetPaymentMethodTypes() []PaymentMethodType {
	return []PaymentMethodType{
		PaymentMethodCreditCard,
		PaymentMethodDebitCard,
		PaymentMethodDigitalWallet,
		PaymentMethodCash,
		PaymentMethodBankTransfer,
	}
}

// GetPaymentStatuses returns all valid payment statuses
func GetPaymentStatuses() []PaymentStatus {
	return []PaymentStatus{
		PaymentStatusPending,
		PaymentStatusProcessing,
		PaymentStatusCompleted,
		PaymentStatusFailed,
		PaymentStatusCancelled,
		PaymentStatusRefunded,
	}
}
