package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rideshare-platform/services/payment-service/internal/types"
	"github.com/rideshare-platform/shared/logger"
)

// PaymentRepository defines the interface for payment data operations
type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment *types.Payment) error
	GetPayment(ctx context.Context, paymentID string) (*types.Payment, error)
	UpdatePaymentStatus(ctx context.Context, paymentID string, status types.PaymentStatus, processorResponse string) error
	GetPaymentsByTrip(ctx context.Context, tripID string) ([]*types.Payment, error)
	GetPaymentsByUser(ctx context.Context, userID string, limit, offset int) ([]*types.Payment, error)
	GetPaymentsByStatus(ctx context.Context, status types.PaymentStatus, limit, offset int) ([]*types.Payment, error)
}

// PaymentMethodRepository defines the interface for payment method operations
type PaymentMethodRepository interface {
	CreatePaymentMethod(ctx context.Context, method *types.PaymentMethodDetails) error
	GetPaymentMethod(ctx context.Context, methodID string) (*types.PaymentMethodDetails, error)
	GetUserPaymentMethods(ctx context.Context, userID string) ([]*types.PaymentMethodDetails, error)
	UpdatePaymentMethod(ctx context.Context, method *types.PaymentMethodDetails) error
	DeletePaymentMethod(ctx context.Context, methodID string) error
	SetDefaultPaymentMethod(ctx context.Context, userID, methodID string) error
}

// RefundRepository defines the interface for refund operations
type RefundRepository interface {
	CreateRefund(ctx context.Context, refund *types.RefundRequest) error
	GetRefund(ctx context.Context, refundID string) (*types.RefundRequest, error)
	GetRefundsByPayment(ctx context.Context, paymentID string) ([]*types.RefundRequest, error)
	UpdateRefundStatus(ctx context.Context, refundID string, status types.PaymentStatus) error
}

// PostgreSQLPaymentRepository implements PaymentRepository using PostgreSQL
type PostgreSQLPaymentRepository struct {
	db     *sql.DB
	logger logger.Logger
}

// NewPostgreSQLPaymentRepository creates a new PostgreSQL payment repository
func NewPostgreSQLPaymentRepository(db *sql.DB, logger logger.Logger) *PostgreSQLPaymentRepository {
	return &PostgreSQLPaymentRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PostgreSQLPaymentRepository) CreatePayment(ctx context.Context, payment *types.Payment) error {
	fraudScoresJSON, _ := json.Marshal(payment.FraudScores)
	metadataJSON, _ := json.Marshal(payment.Metadata)

	query := `
		INSERT INTO payments (
			id, trip_id, user_id, driver_id, amount, currency, payment_method,
			status, transaction_type, processor_response, fraud_risk,
			fraud_scores, metadata, failure_reason, processed_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err := r.db.ExecContext(ctx, query,
		payment.ID, payment.TripID, payment.UserID, payment.DriverID,
		payment.Amount, payment.Currency, payment.PaymentMethod,
		payment.Status, payment.TransactionType, payment.ProcessorResponse,
		payment.FraudRisk, fraudScoresJSON, metadataJSON,
		payment.FailureReason, payment.ProcessedAt,
		payment.CreatedAt, payment.UpdatedAt,
	)

	return err
}

func (r *PostgreSQLPaymentRepository) GetPayment(ctx context.Context, paymentID string) (*types.Payment, error) {
	query := `
		SELECT id, trip_id, user_id, driver_id, amount, currency, payment_method,
			   status, transaction_type, processor_response, fraud_risk,
			   fraud_scores, metadata, failure_reason, processed_at, created_at, updated_at
		FROM payments WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, paymentID)
	return r.scanPayment(row)
}

func (r *PostgreSQLPaymentRepository) UpdatePaymentStatus(ctx context.Context, paymentID string, status types.PaymentStatus, processorResponse string) error {
	query := `
		UPDATE payments 
		SET status = $1, processor_response = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query, status, processorResponse, time.Now(), paymentID)
	return err
}

func (r *PostgreSQLPaymentRepository) GetPaymentsByTrip(ctx context.Context, tripID string) ([]*types.Payment, error) {
	query := `
		SELECT id, trip_id, user_id, driver_id, amount, currency, payment_method,
			   status, transaction_type, processor_response, fraud_risk,
			   fraud_scores, metadata, failure_reason, processed_at, created_at, updated_at
		FROM payments WHERE trip_id = $1 ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPayments(rows)
}

func (r *PostgreSQLPaymentRepository) GetPaymentsByUser(ctx context.Context, userID string, limit, offset int) ([]*types.Payment, error) {
	query := `
		SELECT id, trip_id, user_id, driver_id, amount, currency, payment_method,
			   status, transaction_type, processor_response, fraud_risk,
			   fraud_scores, metadata, failure_reason, processed_at, created_at, updated_at
		FROM payments WHERE user_id = $1 
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPayments(rows)
}

func (r *PostgreSQLPaymentRepository) GetPaymentsByStatus(ctx context.Context, status types.PaymentStatus, limit, offset int) ([]*types.Payment, error) {
	query := `
		SELECT id, trip_id, user_id, driver_id, amount, currency, payment_method,
			   status, transaction_type, processor_response, fraud_risk,
			   fraud_scores, metadata, failure_reason, processed_at, created_at, updated_at
		FROM payments WHERE status = $1 
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPayments(rows)
}

func (r *PostgreSQLPaymentRepository) scanPayment(row *sql.Row) (*types.Payment, error) {
	var payment types.Payment
	var fraudScoresJSON, metadataJSON []byte

	err := row.Scan(
		&payment.ID, &payment.TripID, &payment.UserID, &payment.DriverID,
		&payment.Amount, &payment.Currency, &payment.PaymentMethod,
		&payment.Status, &payment.TransactionType, &payment.ProcessorResponse,
		&payment.FraudRisk, &fraudScoresJSON, &metadataJSON,
		&payment.FailureReason, &payment.ProcessedAt,
		&payment.CreatedAt, &payment.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	if len(fraudScoresJSON) > 0 {
		json.Unmarshal(fraudScoresJSON, &payment.FraudScores)
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &payment.Metadata)
	}

	return &payment, nil
}

func (r *PostgreSQLPaymentRepository) scanPayments(rows *sql.Rows) ([]*types.Payment, error) {
	var payments []*types.Payment

	for rows.Next() {
		var payment types.Payment
		var fraudScoresJSON, metadataJSON []byte

		err := rows.Scan(
			&payment.ID, &payment.TripID, &payment.UserID, &payment.DriverID,
			&payment.Amount, &payment.Currency, &payment.PaymentMethod,
			&payment.Status, &payment.TransactionType, &payment.ProcessorResponse,
			&payment.FraudRisk, &fraudScoresJSON, &metadataJSON,
			&payment.FailureReason, &payment.ProcessedAt,
			&payment.CreatedAt, &payment.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		// Unmarshal JSON fields
		if len(fraudScoresJSON) > 0 {
			json.Unmarshal(fraudScoresJSON, &payment.FraudScores)
		}
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &payment.Metadata)
		}

		payments = append(payments, &payment)
	}

	return payments, rows.Err()
}

// Mock implementations for testing and development

// MockPaymentRepository provides an in-memory implementation for testing
type MockPaymentRepository struct {
	payments map[string]*types.Payment
	mutex    sync.RWMutex
}

// NewMockPaymentRepository creates a new mock payment repository
func NewMockPaymentRepository() *MockPaymentRepository {
	return &MockPaymentRepository{
		payments: make(map[string]*types.Payment),
	}
}

func (m *MockPaymentRepository) CreatePayment(ctx context.Context, payment *types.Payment) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if payment.ID == "" {
		payment.ID = uuid.New().String()
	}

	now := time.Now()
	payment.CreatedAt = now
	payment.UpdatedAt = now

	m.payments[payment.ID] = payment
	return nil
}

func (m *MockPaymentRepository) GetPayment(ctx context.Context, paymentID string) (*types.Payment, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	payment, exists := m.payments[paymentID]
	if !exists {
		return nil, fmt.Errorf("payment not found: %s", paymentID)
	}

	return payment, nil
}

func (m *MockPaymentRepository) UpdatePaymentStatus(ctx context.Context, paymentID string, status types.PaymentStatus, processorResponse string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	payment, exists := m.payments[paymentID]
	if !exists {
		return fmt.Errorf("payment not found: %s", paymentID)
	}

	payment.Status = status
	payment.ProcessorResponse = processorResponse
	payment.UpdatedAt = time.Now()

	return nil
}

func (m *MockPaymentRepository) GetPaymentsByTrip(ctx context.Context, tripID string) ([]*types.Payment, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var payments []*types.Payment
	for _, payment := range m.payments {
		if payment.TripID == tripID {
			payments = append(payments, payment)
		}
	}

	return payments, nil
}

func (m *MockPaymentRepository) GetPaymentsByUser(ctx context.Context, userID string, limit, offset int) ([]*types.Payment, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var payments []*types.Payment
	count := 0
	for _, payment := range m.payments {
		if payment.UserID == userID {
			if count >= offset && len(payments) < limit {
				payments = append(payments, payment)
			}
			count++
		}
	}

	return payments, nil
}

func (m *MockPaymentRepository) GetPaymentsByStatus(ctx context.Context, status types.PaymentStatus, limit, offset int) ([]*types.Payment, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var payments []*types.Payment
	count := 0
	for _, payment := range m.payments {
		if payment.Status == status {
			if count >= offset && len(payments) < limit {
				payments = append(payments, payment)
			}
			count++
		}
	}

	return payments, nil
}

// MockPaymentMethodRepository provides an in-memory implementation for testing
type MockPaymentMethodRepository struct {
	methods map[string]*types.PaymentMethodDetails
	mutex   sync.RWMutex
}

// NewMockPaymentMethodRepository creates a new mock payment method repository
func NewMockPaymentMethodRepository() *MockPaymentMethodRepository {
	return &MockPaymentMethodRepository{
		methods: make(map[string]*types.PaymentMethodDetails),
	}
}

func (m *MockPaymentMethodRepository) CreatePaymentMethod(ctx context.Context, method *types.PaymentMethodDetails) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if method.ID == "" {
		method.ID = uuid.New().String()
	}

	now := time.Now()
	method.CreatedAt = now
	method.UpdatedAt = now

	m.methods[method.ID] = method
	return nil
}

func (m *MockPaymentMethodRepository) GetPaymentMethod(ctx context.Context, methodID string) (*types.PaymentMethodDetails, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	method, exists := m.methods[methodID]
	if !exists {
		return nil, fmt.Errorf("payment method not found: %s", methodID)
	}

	return method, nil
}

func (m *MockPaymentMethodRepository) GetUserPaymentMethods(ctx context.Context, userID string) ([]*types.PaymentMethodDetails, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var methods []*types.PaymentMethodDetails
	for _, method := range m.methods {
		if method.UserID == userID {
			methods = append(methods, method)
		}
	}

	return methods, nil
}

func (m *MockPaymentMethodRepository) UpdatePaymentMethod(ctx context.Context, method *types.PaymentMethodDetails) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	existing, exists := m.methods[method.ID]
	if !exists {
		return fmt.Errorf("payment method not found: %s", method.ID)
	}

	method.CreatedAt = existing.CreatedAt
	method.UpdatedAt = time.Now()
	m.methods[method.ID] = method

	return nil
}

func (m *MockPaymentMethodRepository) DeletePaymentMethod(ctx context.Context, methodID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.methods[methodID]; !exists {
		return fmt.Errorf("payment method not found: %s", methodID)
	}

	delete(m.methods, methodID)
	return nil
}

func (m *MockPaymentMethodRepository) SetDefaultPaymentMethod(ctx context.Context, userID, methodID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// First, unset all default methods for the user
	for _, method := range m.methods {
		if method.UserID == userID {
			method.IsDefault = false
		}
	}

	// Set the specified method as default
	method, exists := m.methods[methodID]
	if !exists {
		return fmt.Errorf("payment method not found: %s", methodID)
	}

	if method.UserID != userID {
		return fmt.Errorf("payment method does not belong to user: %s", userID)
	}

	method.IsDefault = true
	method.UpdatedAt = time.Now()

	return nil
}

// MockRefundRepository provides an in-memory implementation for testing
type MockRefundRepository struct {
	refunds map[string]*types.RefundRequest
	mutex   sync.RWMutex
}

// NewMockRefundRepository creates a new mock refund repository
func NewMockRefundRepository() *MockRefundRepository {
	return &MockRefundRepository{
		refunds: make(map[string]*types.RefundRequest),
	}
}

func (m *MockRefundRepository) CreateRefund(ctx context.Context, refund *types.RefundRequest) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if refund.ID == "" {
		refund.ID = uuid.New().String()
	}

	refund.CreatedAt = time.Now()
	m.refunds[refund.ID] = refund
	return nil
}

func (m *MockRefundRepository) GetRefund(ctx context.Context, refundID string) (*types.RefundRequest, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	refund, exists := m.refunds[refundID]
	if !exists {
		return nil, fmt.Errorf("refund not found: %s", refundID)
	}

	return refund, nil
}

func (m *MockRefundRepository) GetRefundsByPayment(ctx context.Context, paymentID string) ([]*types.RefundRequest, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var refunds []*types.RefundRequest
	for _, refund := range m.refunds {
		if refund.PaymentID == paymentID {
			refunds = append(refunds, refund)
		}
	}

	return refunds, nil
}

func (m *MockRefundRepository) UpdateRefundStatus(ctx context.Context, refundID string, status types.PaymentStatus) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	refund, exists := m.refunds[refundID]
	if !exists {
		return fmt.Errorf("refund not found: %s", refundID)
	}

	refund.Status = status
	if status == types.PaymentStatusCompleted || status == types.PaymentStatusFailed {
		now := time.Now()
		refund.ProcessedAt = &now
	}

	return nil
}
