package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/rideshare-platform/services/payment-service/internal/types"
)

// Mock payment processors for different payment methods

// MockCardProcessor simulates credit/debit card processing
type MockCardProcessor struct{}

func NewMockCardProcessor() *MockCardProcessor {
	return &MockCardProcessor{}
}

func (p *MockCardProcessor) ProcessPayment(ctx context.Context, payment *types.Payment) (*ProcessorResponse, error) {
	// Simulate processing delay
	time.Sleep(time.Millisecond * 200)

	// Simulate random failures (10% failure rate)
	rand.Seed(time.Now().UnixNano())
	if rand.Float64() < 0.1 {
		return &ProcessorResponse{
			Success:         false,
			TransactionID:   uuid.New().String(),
			ProcessorID:     "card_processor_v1",
			ResponseCode:    "DECLINED",
			ResponseMessage: "Card declined by issuer",
			ProcessingFee:   0,
		}, nil
	}

	// Simulate successful payment
	return &ProcessorResponse{
		Success:           true,
		TransactionID:     uuid.New().String(),
		ProcessorID:       "card_processor_v1",
		ResponseCode:      "APPROVED",
		ResponseMessage:   "Payment approved",
		ProcessingFee:     payment.Amount * 0.029, // 2.9% processing fee
		AuthorizationCode: fmt.Sprintf("AUTH_%d", rand.Int31()),
	}, nil
}

func (p *MockCardProcessor) ProcessRefund(ctx context.Context, payment *types.Payment, amount float64) (*ProcessorResponse, error) {
	// Simulate processing delay
	time.Sleep(time.Millisecond * 300)

	// Simulate random failures (5% failure rate for refunds)
	rand.Seed(time.Now().UnixNano())
	if rand.Float64() < 0.05 {
		return &ProcessorResponse{
			Success:         false,
			TransactionID:   uuid.New().String(),
			ProcessorID:     "card_processor_v1",
			ResponseCode:    "REFUND_FAILED",
			ResponseMessage: "Refund could not be processed",
			ProcessingFee:   0,
		}, nil
	}

	return &ProcessorResponse{
		Success:         true,
		TransactionID:   uuid.New().String(),
		ProcessorID:     "card_processor_v1",
		ResponseCode:    "REFUND_APPROVED",
		ResponseMessage: "Refund processed successfully",
		ProcessingFee:   0, // No fee for refunds
	}, nil
}

func (p *MockCardProcessor) VerifyPaymentMethod(ctx context.Context, method *types.PaymentMethodDetails) error {
	// Simulate card verification
	time.Sleep(time.Millisecond * 100)

	// Basic validation
	if cardNumber, ok := method.Details["card_number"].(string); ok {
		if len(cardNumber) < 13 || len(cardNumber) > 19 {
			return fmt.Errorf("invalid card number length")
		}
	} else {
		return fmt.Errorf("card number is required")
	}

	if cvv, ok := method.Details["cvv"].(string); ok {
		if len(cvv) < 3 || len(cvv) > 4 {
			return fmt.Errorf("invalid CVV")
		}
	} else {
		return fmt.Errorf("CVV is required")
	}

	// Simulate random verification failures (2% failure rate)
	rand.Seed(time.Now().UnixNano())
	if rand.Float64() < 0.02 {
		return fmt.Errorf("card verification failed")
	}

	return nil
}

// MockWalletProcessor simulates digital wallet processing (PayPal, Apple Pay, etc.)
type MockWalletProcessor struct{}

func NewMockWalletProcessor() *MockWalletProcessor {
	return &MockWalletProcessor{}
}

func (p *MockWalletProcessor) ProcessPayment(ctx context.Context, payment *types.Payment) (*ProcessorResponse, error) {
	// Simulate processing delay
	time.Sleep(time.Millisecond * 150)

	// Digital wallets typically have lower failure rates (5%)
	rand.Seed(time.Now().UnixNano())
	if rand.Float64() < 0.05 {
		return &ProcessorResponse{
			Success:         false,
			TransactionID:   uuid.New().String(),
			ProcessorID:     "wallet_processor_v2",
			ResponseCode:    "INSUFFICIENT_FUNDS",
			ResponseMessage: "Insufficient balance in wallet",
			ProcessingFee:   0,
		}, nil
	}

	return &ProcessorResponse{
		Success:         true,
		TransactionID:   uuid.New().String(),
		ProcessorID:     "wallet_processor_v2",
		ResponseCode:    "SUCCESS",
		ResponseMessage: "Wallet payment successful",
		ProcessingFee:   payment.Amount * 0.025, // 2.5% processing fee
	}, nil
}

func (p *MockWalletProcessor) ProcessRefund(ctx context.Context, payment *types.Payment, amount float64) (*ProcessorResponse, error) {
	time.Sleep(time.Millisecond * 200)

	// Very low failure rate for wallet refunds (1%)
	rand.Seed(time.Now().UnixNano())
	if rand.Float64() < 0.01 {
		return &ProcessorResponse{
			Success:         false,
			TransactionID:   uuid.New().String(),
			ProcessorID:     "wallet_processor_v2",
			ResponseCode:    "REFUND_FAILED",
			ResponseMessage: "Wallet refund failed",
			ProcessingFee:   0,
		}, nil
	}

	return &ProcessorResponse{
		Success:         true,
		TransactionID:   uuid.New().String(),
		ProcessorID:     "wallet_processor_v2",
		ResponseCode:    "REFUND_SUCCESS",
		ResponseMessage: "Wallet refund completed",
		ProcessingFee:   0,
	}, nil
}

func (p *MockWalletProcessor) VerifyPaymentMethod(ctx context.Context, method *types.PaymentMethodDetails) error {
	time.Sleep(time.Millisecond * 50)

	// Verify email for wallet
	if email, ok := method.Details["email"].(string); ok {
		if !contains(email, "@") {
			return fmt.Errorf("invalid email format")
		}
	} else {
		return fmt.Errorf("email is required for wallet")
	}

	return nil
}

// MockBankProcessor simulates bank transfer processing
type MockBankProcessor struct{}

func NewMockBankProcessor() *MockBankProcessor {
	return &MockBankProcessor{}
}

func (p *MockBankProcessor) ProcessPayment(ctx context.Context, payment *types.Payment) (*ProcessorResponse, error) {
	// Bank transfers take longer to process
	time.Sleep(time.Millisecond * 500)

	// Higher failure rate for bank transfers (15%)
	rand.Seed(time.Now().UnixNano())
	if rand.Float64() < 0.15 {
		return &ProcessorResponse{
			Success:         false,
			TransactionID:   uuid.New().String(),
			ProcessorID:     "bank_processor_v1",
			ResponseCode:    "ACCOUNT_BLOCKED",
			ResponseMessage: "Bank account is blocked or insufficient funds",
			ProcessingFee:   0,
		}, nil
	}

	return &ProcessorResponse{
		Success:         true,
		TransactionID:   uuid.New().String(),
		ProcessorID:     "bank_processor_v1",
		ResponseCode:    "TRANSFER_INITIATED",
		ResponseMessage: "Bank transfer initiated successfully",
		ProcessingFee:   payment.Amount * 0.01, // 1% processing fee
	}, nil
}

func (p *MockBankProcessor) ProcessRefund(ctx context.Context, payment *types.Payment, amount float64) (*ProcessorResponse, error) {
	time.Sleep(time.Millisecond * 600)

	// Moderate failure rate for bank refunds (8%)
	rand.Seed(time.Now().UnixNano())
	if rand.Float64() < 0.08 {
		return &ProcessorResponse{
			Success:         false,
			TransactionID:   uuid.New().String(),
			ProcessorID:     "bank_processor_v1",
			ResponseCode:    "REFUND_BLOCKED",
			ResponseMessage: "Bank refund could not be initiated",
			ProcessingFee:   0,
		}, nil
	}

	return &ProcessorResponse{
		Success:         true,
		TransactionID:   uuid.New().String(),
		ProcessorID:     "bank_processor_v1",
		ResponseCode:    "REFUND_INITIATED",
		ResponseMessage: "Bank refund initiated",
		ProcessingFee:   0,
	}, nil
}

func (p *MockBankProcessor) VerifyPaymentMethod(ctx context.Context, method *types.PaymentMethodDetails) error {
	time.Sleep(time.Millisecond * 200)

	// Verify account number
	if accountNumber, ok := method.Details["account_number"].(string); ok {
		if len(accountNumber) < 8 || len(accountNumber) > 17 {
			return fmt.Errorf("invalid account number length")
		}
	} else {
		return fmt.Errorf("account number is required")
	}

	// Verify routing number
	if routingNumber, ok := method.Details["routing_number"].(string); ok {
		if len(routingNumber) != 9 {
			return fmt.Errorf("routing number must be 9 digits")
		}
	} else {
		return fmt.Errorf("routing number is required")
	}

	return nil
}

// MockCashProcessor simulates cash payment handling
type MockCashProcessor struct{}

func NewMockCashProcessor() *MockCashProcessor {
	return &MockCashProcessor{}
}

func (p *MockCashProcessor) ProcessPayment(ctx context.Context, payment *types.Payment) (*ProcessorResponse, error) {
	// Cash payments are instant but require driver confirmation
	time.Sleep(time.Millisecond * 50)

	// Very low failure rate for cash (2%)
	rand.Seed(time.Now().UnixNano())
	if rand.Float64() < 0.02 {
		return &ProcessorResponse{
			Success:         false,
			TransactionID:   uuid.New().String(),
			ProcessorID:     "cash_processor_v1",
			ResponseCode:    "CASH_NOT_RECEIVED",
			ResponseMessage: "Driver did not confirm cash receipt",
			ProcessingFee:   0,
		}, nil
	}

	return &ProcessorResponse{
		Success:         true,
		TransactionID:   uuid.New().String(),
		ProcessorID:     "cash_processor_v1",
		ResponseCode:    "CASH_RECEIVED",
		ResponseMessage: "Cash payment confirmed by driver",
		ProcessingFee:   0, // No processing fee for cash
	}, nil
}

func (p *MockCashProcessor) ProcessRefund(ctx context.Context, payment *types.Payment, amount float64) (*ProcessorResponse, error) {
	// Cash refunds require manual handling
	time.Sleep(time.Millisecond * 100)

	return &ProcessorResponse{
		Success:         true,
		TransactionID:   uuid.New().String(),
		ProcessorID:     "cash_processor_v1",
		ResponseCode:    "MANUAL_REFUND",
		ResponseMessage: "Cash refund to be handled manually by driver",
		ProcessingFee:   0,
	}, nil
}

func (p *MockCashProcessor) VerifyPaymentMethod(ctx context.Context, method *types.PaymentMethodDetails) error {
	// Cash doesn't require verification
	return nil
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || contains(s[1:], substr) || (len(s) > 0 && s[0:len(substr)] == substr))
}
