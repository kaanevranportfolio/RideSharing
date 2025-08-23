package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/rideshare-platform/shared/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// ClientConfig holds gRPC client configuration
type ClientConfig struct {
	Address             string
	Timeout             time.Duration
	MaxRecvMsgSize      int
	MaxSendMsgSize      int
	KeepAliveTime       time.Duration
	KeepAliveTimeout    time.Duration
	PermitWithoutStream bool
	MaxRetryAttempts    int
	InitialBackoff      time.Duration
	MaxBackoff          time.Duration
	BackoffMultiplier   float64
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig(address string) *ClientConfig {
	return &ClientConfig{
		Address:             address,
		Timeout:             30 * time.Second,
		MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
		MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
		KeepAliveTime:       30 * time.Second,
		KeepAliveTimeout:    5 * time.Second,
		PermitWithoutStream: true,
		MaxRetryAttempts:    3,
		InitialBackoff:      100 * time.Millisecond,
		MaxBackoff:          30 * time.Second,
		BackoffMultiplier:   1.6,
	}
}

// Client wraps gRPC client connection with additional functionality
type Client struct {
	conn   *grpc.ClientConn
	config *ClientConfig
	logger *logger.Logger
}

// NewClient creates a new gRPC client
func NewClient(config *ClientConfig, log *logger.Logger) (*Client, error) {
	// Client options
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(config.MaxSendMsgSize),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                config.KeepAliveTime,
			Timeout:             config.KeepAliveTimeout,
			PermitWithoutStream: config.PermitWithoutStream,
		}),
		grpc.WithUnaryInterceptor(unaryClientInterceptor(log)),
		grpc.WithStreamInterceptor(streamClientInterceptor(log)),
	}

	// Establish connection
	conn, err := grpc.Dial(config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", config.Address, err)
	}

	log.WithFields(logger.Fields{
		"address": config.Address,
	}).Info("gRPC client connected")

	return &Client{
		conn:   conn,
		config: config,
		logger: log,
	}, nil
}

// GetConnection returns the underlying gRPC connection
func (c *Client) GetConnection() *grpc.ClientConn {
	return c.conn
}

// Close closes the client connection
func (c *Client) Close() error {
	c.logger.WithFields(logger.Fields{
		"address": c.config.Address,
	}).Info("Closing gRPC client connection")
	return c.conn.Close()
}

// Health checks the connection health
func (c *Client) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	state := c.conn.GetState()
	c.logger.WithFields(logger.Fields{
		"address": c.config.Address,
		"state":   state.String(),
	}).Debug("gRPC connection state")

	// Wait for connection to be ready
	if !c.conn.WaitForStateChange(ctx, state) {
		return fmt.Errorf("connection not ready: %s", state)
	}

	return nil
}

// unaryClientInterceptor provides logging and metrics for unary RPCs
func unaryClientInterceptor(log *logger.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()

		// Call the method
		err := invoker(ctx, method, req, reply, cc, opts...)

		// Calculate duration
		duration := time.Since(start)

		// Log the request
		log.LogGRPCRequest(ctx, method, duration, err)

		return err
	}
}

// streamClientInterceptor provides logging and metrics for streaming RPCs
func streamClientInterceptor(log *logger.Logger) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		start := time.Now()

		// Call the method
		stream, err := streamer(ctx, desc, cc, method, opts...)

		// Calculate duration
		duration := time.Since(start)

		// Log the request
		log.LogGRPCRequest(ctx, method, duration, err)

		return stream, err
	}
}

// ClientManager manages multiple gRPC clients
type ClientManager struct {
	clients map[string]*Client
	logger  *logger.Logger
}

// NewClientManager creates a new client manager
func NewClientManager(log *logger.Logger) *ClientManager {
	return &ClientManager{
		clients: make(map[string]*Client),
		logger:  log,
	}
}

// AddClient adds a client to the manager
func (cm *ClientManager) AddClient(name string, client *Client) {
	cm.clients[name] = client
	cm.logger.WithFields(logger.Fields{
		"client":  name,
		"address": client.config.Address,
	}).Info("Client added to manager")
}

// GetClient gets a client by name
func (cm *ClientManager) GetClient(name string) (*Client, bool) {
	client, exists := cm.clients[name]
	return client, exists
}

// CloseAll closes all clients
func (cm *ClientManager) CloseAll() error {
	for name, client := range cm.clients {
		cm.logger.WithFields(logger.Fields{
			"client": name,
		}).Info("Closing client")
		if err := client.Close(); err != nil {
			cm.logger.WithError(err).WithFields(logger.Fields{
				"client": name,
			}).Error("Failed to close client")
		}
	}
	return nil
}

// HealthCheckAll checks health of all clients
func (cm *ClientManager) HealthCheckAll(ctx context.Context) map[string]error {
	results := make(map[string]error)

	for name, client := range cm.clients {
		if err := client.Health(ctx); err != nil {
			results[name] = err
			cm.logger.WithError(err).WithFields(logger.Fields{
				"client": name,
			}).Warn("Client health check failed")
		} else {
			cm.logger.WithFields(logger.Fields{
				"client": name,
			}).Debug("Client health check passed")
		}
	}

	return results
}

// RetryableClient provides retry functionality for gRPC calls
type RetryableClient struct {
	client *Client
	config *ClientConfig
	logger *logger.Logger
}

// NewRetryableClient creates a new retryable client
func NewRetryableClient(client *Client, log *logger.Logger) *RetryableClient {
	return &RetryableClient{
		client: client,
		config: client.config,
		logger: log,
	}
}

// CallWithRetry executes a gRPC call with retry logic
func (rc *RetryableClient) CallWithRetry(ctx context.Context, call func() error) error {
	var lastErr error
	backoff := rc.config.InitialBackoff

	for attempt := 0; attempt < rc.config.MaxRetryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}

			// Increase backoff
			backoff = time.Duration(float64(backoff) * rc.config.BackoffMultiplier)
			if backoff > rc.config.MaxBackoff {
				backoff = rc.config.MaxBackoff
			}
		}

		// Execute the call
		err := call()
		if err == nil {
			if attempt > 0 {
				rc.logger.WithContext(ctx).WithFields(logger.Fields{
					"attempt": attempt + 1,
				}).Info("gRPC call succeeded after retry")
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			rc.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
				"attempt": attempt + 1,
			}).Warn("gRPC call failed with non-retryable error")
			return err
		}

		rc.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"attempt": attempt + 1,
			"backoff": backoff,
		}).Warn("gRPC call failed, retrying")
	}

	rc.logger.WithContext(ctx).WithError(lastErr).WithFields(logger.Fields{
		"max_attempts": rc.config.MaxRetryAttempts,
	}).Error("gRPC call failed after all retry attempts")

	return fmt.Errorf("call failed after %d attempts: %w", rc.config.MaxRetryAttempts, lastErr)
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	// Implement based on gRPC status codes
	// For now, assume all errors are retryable except context errors
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}
	return true
}

// LoadBalancer provides simple load balancing for multiple clients
type LoadBalancer struct {
	clients []*Client
	current int
	logger  *logger.Logger
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(clients []*Client, log *logger.Logger) *LoadBalancer {
	return &LoadBalancer{
		clients: clients,
		current: 0,
		logger:  log,
	}
}

// GetClient returns the next client using round-robin
func (lb *LoadBalancer) GetClient() *Client {
	if len(lb.clients) == 0 {
		return nil
	}

	client := lb.clients[lb.current]
	lb.current = (lb.current + 1) % len(lb.clients)

	lb.logger.WithFields(logger.Fields{
		"client_index":  lb.current,
		"total_clients": len(lb.clients),
	}).Debug("Selected client for load balancing")

	return client
}

// AddClient adds a client to the load balancer
func (lb *LoadBalancer) AddClient(client *Client) {
	lb.clients = append(lb.clients, client)
	lb.logger.WithFields(logger.Fields{
		"total_clients": len(lb.clients),
	}).Info("Client added to load balancer")
}

// RemoveClient removes a client from the load balancer
func (lb *LoadBalancer) RemoveClient(client *Client) {
	for i, c := range lb.clients {
		if c == client {
			lb.clients = append(lb.clients[:i], lb.clients[i+1:]...)
			if lb.current >= len(lb.clients) {
				lb.current = 0
			}
			lb.logger.WithFields(logger.Fields{
				"total_clients": len(lb.clients),
			}).Info("Client removed from load balancer")
			break
		}
	}
}

// CloseAll closes all clients in the load balancer
func (lb *LoadBalancer) CloseAll() error {
	for _, client := range lb.clients {
		if err := client.Close(); err != nil {
			lb.logger.WithError(err).Error("Failed to close client in load balancer")
		}
	}
	return nil
}

// CircuitBreaker provides circuit breaker functionality
type CircuitBreaker struct {
	client           *Client
	failureThreshold int
	resetTimeout     time.Duration
	failures         int
	lastFailureTime  time.Time
	state            string // "closed", "open", "half-open"
	logger           *logger.Logger
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(client *Client, failureThreshold int, resetTimeout time.Duration, log *logger.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		client:           client,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		failures:         0,
		state:            "closed",
		logger:           log,
	}
}

// Call executes a call through the circuit breaker
func (cb *CircuitBreaker) Call(ctx context.Context, call func() error) error {
	// Check circuit state
	if cb.state == "open" {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = "half-open"
			cb.logger.WithContext(ctx).Info("Circuit breaker state changed to half-open")
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	// Execute the call
	err := call()

	if err != nil {
		cb.failures++
		cb.lastFailureTime = time.Now()

		if cb.failures >= cb.failureThreshold {
			cb.state = "open"
			cb.logger.WithContext(ctx).WithFields(logger.Fields{
				"failures":  cb.failures,
				"threshold": cb.failureThreshold,
			}).Warn("Circuit breaker opened")
		}

		return err
	}

	// Success - reset circuit breaker
	if cb.state == "half-open" {
		cb.state = "closed"
		cb.logger.WithContext(ctx).Info("Circuit breaker state changed to closed")
	}
	cb.failures = 0

	return nil
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() string {
	return cb.state
}

// Reset resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.failures = 0
	cb.state = "closed"
	cb.logger.Logger.Info("Circuit breaker reset")
}
