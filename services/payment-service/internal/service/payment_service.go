package service

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rideshare-platform/services/payment-service/internal/repository"
	"github.com/rideshare-platform/services/payment-service/internal/types"
	"github.com/rideshare-platform/shared/logger"
)

// PaymentProcessor interface for different payment processors
type PaymentProcessor interface {
	ProcessPayment(ctx context.Context, payment *types.Payment) (*ProcessorResponse, error)
	ProcessRefund(ctx context.Context, payment *types.Payment, amount float64) (*ProcessorResponse, error)
	VerifyPaymentMethod(ctx context.Context, method *types.PaymentMethodDetails) error
}

// ProcessorResponse represents the response from a payment processor
type ProcessorResponse struct {
	Success           bool    `json:"success"`
	TransactionID     string  `json:"transaction_id"`
	ProcessorID       string  `json:"processor_id"`
	ResponseCode      string  `json:"response_code"`
	ResponseMessage   string  `json:"response_message"`
	ProcessingFee     float64 `json:"processing_fee"`
	AuthorizationCode string  `json:"authorization_code,omitempty"`
}

// FraudDetectionService handles fraud detection logic
type FraudDetectionService interface {
	AnalyzeTransaction(ctx context.Context, payment *types.Payment) (*types.FraudDetectionResult, error)
}

// PaymentService handles all payment-related operations
type PaymentService struct {
	paymentRepo       repository.PaymentRepository
	paymentMethodRepo repository.PaymentMethodRepository
	refundRepo        repository.RefundRepository
	fraudService      FraudDetectionService
	processors        map[types.PaymentMethod]PaymentProcessor
	logger            logger.Logger
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	paymentRepo repository.PaymentRepository,
	paymentMethodRepo repository.PaymentMethodRepository,
	refundRepo repository.RefundRepository,
	fraudService FraudDetectionService,
	logger logger.Logger,
) *PaymentService {
	service := &PaymentService{
		paymentRepo:       paymentRepo,
		paymentMethodRepo: paymentMethodRepo,
		refundRepo:        refundRepo,
		fraudService:      fraudService,
		processors:        make(map[types.PaymentMethod]PaymentProcessor),
		logger:            logger,
	}

	// Initialize mock processors
	service.processors[types.PaymentMethodCreditCard] = NewMockCardProcessor()
	service.processors[types.PaymentMethodDebitCard] = NewMockCardProcessor()
	service.processors[types.PaymentMethodDigitalWallet] = NewMockWalletProcessor()
	service.processors[types.PaymentMethodBankTransfer] = NewMockBankProcessor()
	service.processors[types.PaymentMethodCash] = NewMockCashProcessor()

	return service
}

// ProcessPayment processes a payment transaction
func (s *PaymentService) ProcessPayment(ctx context.Context, req *types.ProcessPaymentRequest) (*types.PaymentResponse, error) {
	// Get payment method details
	paymentMethod, err := s.paymentMethodRepo.GetPaymentMethod(ctx, req.PaymentMethodID)
	if err != nil {
		return &types.PaymentResponse{
			Success: false,
			Message: "Payment method not found",
			Errors:  []string{err.Error()},
		}, nil
	}

	// Create payment record
	payment := &types.Payment{
		ID:              uuid.New().String(),
		TripID:          req.TripID,
		UserID:          req.UserID,
		DriverID:        req.DriverID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		PaymentMethod:   paymentMethod.Type,
		Status:          types.PaymentStatusPending,
		TransactionType: types.TransactionTypePayment,
		Metadata:        req.Metadata,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Run fraud detection
	if s.fraudService != nil {
		fraudResult, err := s.fraudService.AnalyzeTransaction(ctx, payment)
		if err != nil {
			s.logger.Error("Fraud detection failed", "error", err, "payment_id", payment.ID)
		} else {
			payment.FraudRisk = fraudResult.RiskLevel
			payment.FraudScores = fraudResult.Scores

			// Block high-risk transactions
			if fraudResult.RiskLevel == types.FraudRiskHigh {
				payment.Status = types.PaymentStatusFailed
				payment.FailureReason = "Transaction blocked due to high fraud risk"

				s.paymentRepo.CreatePayment(ctx, payment)

				return &types.PaymentResponse{
					Payment: payment,
					Success: false,
					Message: "Payment blocked due to security concerns",
					Errors:  []string{"High fraud risk detected"},
				}, nil
			}
		}
	}

	// Save initial payment record
	if err := s.paymentRepo.CreatePayment(ctx, payment); err != nil {
		return &types.PaymentResponse{
			Success: false,
			Message: "Failed to create payment record",
			Errors:  []string{err.Error()},
		}, nil
	}

	// Get appropriate processor
	processor, exists := s.processors[paymentMethod.Type]
	if !exists {
		payment.Status = types.PaymentStatusFailed
		payment.FailureReason = "Unsupported payment method"
		s.paymentRepo.UpdatePaymentStatus(ctx, payment.ID, payment.Status, payment.FailureReason)

		return &types.PaymentResponse{
			Payment: payment,
			Success: false,
			Message: "Unsupported payment method",
		}, nil
	}

	// Update status to processing
	payment.Status = types.PaymentStatusProcessing
	s.paymentRepo.UpdatePaymentStatus(ctx, payment.ID, payment.Status, "Processing payment")

	// Process payment
	processorResp, err := processor.ProcessPayment(ctx, payment)
	if err != nil {
		payment.Status = types.PaymentStatusFailed
		payment.FailureReason = err.Error()
		s.paymentRepo.UpdatePaymentStatus(ctx, payment.ID, payment.Status, payment.FailureReason)

		return &types.PaymentResponse{
			Payment: payment,
			Success: false,
			Message: "Payment processing failed",
			Errors:  []string{err.Error()},
		}, nil
	}

	// Update payment with processor response
	if processorResp.Success {
		payment.Status = types.PaymentStatusCompleted
		now := time.Now()
		payment.ProcessedAt = &now
	} else {
		payment.Status = types.PaymentStatusFailed
		payment.FailureReason = processorResp.ResponseMessage
	}

	payment.ProcessorResponse = fmt.Sprintf("Code: %s, Message: %s, TxnID: %s",
		processorResp.ResponseCode, processorResp.ResponseMessage, processorResp.TransactionID)

	s.paymentRepo.UpdatePaymentStatus(ctx, payment.ID, payment.Status, payment.ProcessorResponse)

	return &types.PaymentResponse{
		Payment: payment,
		Success: processorResp.Success,
		Message: "Payment processed successfully",
	}, nil
}

// ProcessRefund processes a refund request
func (s *PaymentService) ProcessRefund(ctx context.Context, req *types.RefundPaymentRequest) (*types.PaymentResponse, error) {
	// Get original payment
	payment, err := s.paymentRepo.GetPayment(ctx, req.PaymentID)
	if err != nil {
		return &types.PaymentResponse{
			Success: false,
			Message: "Payment not found",
			Errors:  []string{err.Error()},
		}, nil
	}

	// Validate refund amount
	if req.Amount > payment.Amount {
		return &types.PaymentResponse{
			Success: false,
			Message: "Refund amount cannot exceed payment amount",
		}, nil
	}

	// Check if payment can be refunded
	if payment.Status != types.PaymentStatusCompleted {
		return &types.PaymentResponse{
			Success: false,
			Message: "Only completed payments can be refunded",
		}, nil
	}

	// Create refund record
	refund := &types.RefundRequest{
		ID:          uuid.New().String(),
		PaymentID:   req.PaymentID,
		Amount:      req.Amount,
		Reason:      req.Reason,
		RequestedBy: req.RequestedBy,
		Status:      types.PaymentStatusPending,
		CreatedAt:   time.Now(),
	}

	if err := s.refundRepo.CreateRefund(ctx, refund); err != nil {
		return &types.PaymentResponse{
			Success: false,
			Message: "Failed to create refund record",
			Errors:  []string{err.Error()},
		}, nil
	}

	// Get processor for refund
	processor, exists := s.processors[payment.PaymentMethod]
	if !exists {
		s.refundRepo.UpdateRefundStatus(ctx, refund.ID, types.PaymentStatusFailed)
		return &types.PaymentResponse{
			Success: false,
			Message: "Refund processor not available",
		}, nil
	}

	// Process refund
	processorResp, err := processor.ProcessRefund(ctx, payment, req.Amount)
	if err != nil {
		s.refundRepo.UpdateRefundStatus(ctx, refund.ID, types.PaymentStatusFailed)
		return &types.PaymentResponse{
			Success: false,
			Message: "Refund processing failed",
			Errors:  []string{err.Error()},
		}, nil
	}

	// Update refund status
	if processorResp.Success {
		s.refundRepo.UpdateRefundStatus(ctx, refund.ID, types.PaymentStatusCompleted)
		// Note: In real implementation, we might update payment status to partially/fully refunded
	} else {
		s.refundRepo.UpdateRefundStatus(ctx, refund.ID, types.PaymentStatusFailed)
	}

	return &types.PaymentResponse{
		Success: processorResp.Success,
		Message: "Refund processed",
	}, nil
}

// AddPaymentMethod adds a new payment method for a user
func (s *PaymentService) AddPaymentMethod(ctx context.Context, req *types.AddPaymentMethodRequest) (*types.PaymentMethodResponse, error) {
	// Create payment method
	method := &types.PaymentMethodDetails{
		ID:        uuid.New().String(),
		UserID:    req.UserID,
		Type:      req.Type,
		IsDefault: req.IsDefault,
		Details:   req.Details,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Generate fingerprint for duplicate detection
	method.Fingerprint = s.generateFingerprint(method)

	// Extract relevant details based on payment type
	switch req.Type {
	case types.PaymentMethodCreditCard, types.PaymentMethodDebitCard:
		if cardNumber, ok := req.Details["card_number"].(string); ok {
			method.LastFourDigits = cardNumber[len(cardNumber)-4:]
		}
		if bankName, ok := req.Details["bank_name"].(string); ok {
			method.BankName = bankName
		}
	case types.PaymentMethodDigitalWallet:
		if provider, ok := req.Details["provider"].(string); ok {
			method.WalletProvider = provider
		}
	case types.PaymentMethodBankTransfer:
		if bankName, ok := req.Details["bank_name"].(string); ok {
			method.BankName = bankName
		}
	}

	// Verify payment method with processor
	if processor, exists := s.processors[req.Type]; exists {
		if err := processor.VerifyPaymentMethod(ctx, method); err != nil {
			return &types.PaymentMethodResponse{
				Success: false,
				Message: "Payment method verification failed",
				Errors:  []string{err.Error()},
			}, nil
		}
	}

	// Save payment method
	if err := s.paymentMethodRepo.CreatePaymentMethod(ctx, method); err != nil {
		return &types.PaymentMethodResponse{
			Success: false,
			Message: "Failed to save payment method",
			Errors:  []string{err.Error()},
		}, nil
	}

	// Set as default if requested
	if req.IsDefault {
		s.paymentMethodRepo.SetDefaultPaymentMethod(ctx, req.UserID, method.ID)
	}

	return &types.PaymentMethodResponse{
		PaymentMethod: method,
		Success:       true,
		Message:       "Payment method added successfully",
	}, nil
}

// GetUserPaymentMethods retrieves all payment methods for a user
func (s *PaymentService) GetUserPaymentMethods(ctx context.Context, userID string) ([]*types.PaymentMethodDetails, error) {
	return s.paymentMethodRepo.GetUserPaymentMethods(ctx, userID)
}

// GetPayment retrieves a payment by ID
func (s *PaymentService) GetPayment(ctx context.Context, paymentID string) (*types.Payment, error) {
	return s.paymentRepo.GetPayment(ctx, paymentID)
}

// GetUserPayments retrieves payments for a user with pagination
func (s *PaymentService) GetUserPayments(ctx context.Context, userID string, limit, offset int) ([]*types.Payment, error) {
	return s.paymentRepo.GetPaymentsByUser(ctx, userID, limit, offset)
}

// GetTripPayments retrieves all payments for a trip
func (s *PaymentService) GetTripPayments(ctx context.Context, tripID string) ([]*types.Payment, error) {
	return s.paymentRepo.GetPaymentsByTrip(ctx, tripID)
}

// generateFingerprint creates a unique fingerprint for duplicate detection
func (s *PaymentService) generateFingerprint(method *types.PaymentMethodDetails) string {
	var parts []string
	parts = append(parts, string(method.Type))
	parts = append(parts, method.UserID)

	switch method.Type {
	case types.PaymentMethodCreditCard, types.PaymentMethodDebitCard:
		if cardNumber, ok := method.Details["card_number"].(string); ok {
			// Use last 4 digits and length for fingerprint
			parts = append(parts, fmt.Sprintf("%d_%s", len(cardNumber), cardNumber[len(cardNumber)-4:]))
		}
	case types.PaymentMethodDigitalWallet:
		if email, ok := method.Details["email"].(string); ok {
			parts = append(parts, email)
		}
	}

	return strings.Join(parts, "_")
}

// SimpleFraudDetectionService provides basic fraud detection
type SimpleFraudDetectionService struct {
	logger logger.Logger
}

// NewSimpleFraudDetectionService creates a new simple fraud detection service
func NewSimpleFraudDetectionService(logger logger.Logger) *SimpleFraudDetectionService {
	return &SimpleFraudDetectionService{logger: logger}
}

// AnalyzeTransaction analyzes a transaction for fraud indicators
func (s *SimpleFraudDetectionService) AnalyzeTransaction(ctx context.Context, payment *types.Payment) (*types.FraudDetectionResult, error) {
	result := &types.FraudDetectionResult{
		TransactionID: payment.ID,
		Scores:        make(map[string]float64),
		Reasons:       []string{},
	}

	// Amount-based scoring
	amountScore := s.analyzeAmount(payment.Amount)
	result.Scores["amount"] = amountScore

	// Time-based scoring (suspicious if late night)
	timeScore := s.analyzeTime()
	result.Scores["time"] = timeScore

	// Velocity scoring (simplified - in real implementation, check recent transactions)
	velocityScore := s.analyzeVelocity(payment.UserID)
	result.Scores["velocity"] = velocityScore

	// Location scoring (mock - would use IP geolocation in real implementation)
	locationScore := s.analyzeLocation()
	result.Scores["location"] = locationScore

	// Calculate overall risk score
	weights := map[string]float64{
		"amount":   0.3,
		"time":     0.2,
		"velocity": 0.3,
		"location": 0.2,
	}

	var totalScore float64
	for factor, score := range result.Scores {
		totalScore += score * weights[factor]
	}

	result.RiskScore = totalScore

	// Determine risk level
	switch {
	case totalScore >= 0.8:
		result.RiskLevel = types.FraudRiskHigh
		result.RequiresReview = true
	case totalScore >= 0.5:
		result.RiskLevel = types.FraudRiskMedium
		result.RequiresReview = true
	default:
		result.RiskLevel = types.FraudRiskLow
	}

	// Add reasons for high scores
	if amountScore > 0.7 {
		result.Reasons = append(result.Reasons, "Unusually high transaction amount")
	}
	if timeScore > 0.7 {
		result.Reasons = append(result.Reasons, "Transaction during suspicious hours")
	}
	if velocityScore > 0.7 {
		result.Reasons = append(result.Reasons, "High transaction frequency")
	}
	if locationScore > 0.7 {
		result.Reasons = append(result.Reasons, "Suspicious location")
	}

	return result, nil
}

func (s *SimpleFraudDetectionService) analyzeAmount(amount float64) float64 {
	// Higher amounts are more suspicious
	switch {
	case amount > 1000:
		return 0.9
	case amount > 500:
		return 0.7
	case amount > 100:
		return 0.4
	default:
		return 0.1
	}
}

func (s *SimpleFraudDetectionService) analyzeTime() float64 {
	hour := time.Now().Hour()
	// Late night transactions are more suspicious
	if hour >= 2 && hour <= 5 {
		return 0.8
	}
	if hour >= 22 || hour <= 1 {
		return 0.5
	}
	return 0.1
}

func (s *SimpleFraudDetectionService) analyzeVelocity(userID string) float64 {
	// Mock implementation - in real system, check recent transaction count
	// For demo, randomly assign velocity scores
	rand.Seed(time.Now().UnixNano())
	return rand.Float64() * 0.6 // Max 0.6 for velocity
}

func (s *SimpleFraudDetectionService) analyzeLocation() float64 {
	// Mock implementation - in real system, use IP geolocation
	rand.Seed(time.Now().UnixNano())
	return rand.Float64() * 0.5 // Max 0.5 for location
}
