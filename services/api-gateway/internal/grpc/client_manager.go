package grpc

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	geopb "github.com/rideshare-platform/shared/proto/geo"
	matchingpb "github.com/rideshare-platform/shared/proto/matching"
	paymentpb "github.com/rideshare-platform/shared/proto/payment"
	pricingpb "github.com/rideshare-platform/shared/proto/pricing"
	trippb "github.com/rideshare-platform/shared/proto/trip"
	userpb "github.com/rideshare-platform/shared/proto/user"
)

// ServiceConfig holds configuration for individual services
type ServiceConfig struct {
	Address        string
	MaxRetries     int
	TimeoutSeconds int
	EnableTLS      bool
}

// ClientManager manages gRPC connections to all microservices
type ClientManager struct {
	// Service clients
	GeoClient      geopb.GeospatialServiceClient
	UserClient     userpb.UserServiceClient
	TripClient     trippb.TripServiceClient
	MatchingClient matchingpb.MatchingServiceClient
	PricingClient  pricingpb.PricingServiceClient
	PaymentClient  paymentpb.PaymentServiceClient

	// Connection management
	connections map[string]*grpc.ClientConn
	mutex       sync.RWMutex
	config      map[string]ServiceConfig
}

// NewClientManager creates a new gRPC client manager
func NewClientManager() *ClientManager {
	return &ClientManager{
		connections: make(map[string]*grpc.ClientConn),
		config: map[string]ServiceConfig{
			"geo": {
				Address:        "localhost:9083",
				MaxRetries:     3,
				TimeoutSeconds: 30,
				EnableTLS:      false,
			},
			"user": {
				Address:        "localhost:9084",
				MaxRetries:     3,
				TimeoutSeconds: 30,
				EnableTLS:      false,
			},
			"trip": {
				Address:        "localhost:9086",
				MaxRetries:     3,
				TimeoutSeconds: 30,
				EnableTLS:      false,
			},
			"matching": {
				Address:        "localhost:9085",
				MaxRetries:     3,
				TimeoutSeconds: 30,
				EnableTLS:      false,
			},
			"pricing": {
				Address:        "localhost:9087",
				MaxRetries:     3,
				TimeoutSeconds: 30,
				EnableTLS:      false,
			},
			"payment": {
				Address:        "localhost:9088",
				MaxRetries:     3,
				TimeoutSeconds: 30,
				EnableTLS:      false,
			},
		},
	}
}

// Initialize establishes connections to all services
func (cm *ClientManager) Initialize() error {
	log.Println("Initializing gRPC client connections...")

	for serviceName, config := range cm.config {
		if err := cm.connectService(serviceName, config); err != nil {
			log.Printf("Failed to connect to %s service: %v", serviceName, err)
			// Continue with other services for graceful degradation
		}
	}

	return nil
}

// connectService establishes a connection to a specific service
func (cm *ClientManager) connectService(serviceName string, config ServiceConfig) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Configure keepalive and connection parameters
	kacp := keepalive.ClientParameters{
		Time:                10 * time.Second, // send pings every 10 seconds if there is no activity
		Timeout:             time.Second,      // wait 1 second for ping ack before considering the connection dead
		PermitWithoutStream: true,             // send pings even without active streams
	}

	// Create connection options
	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(kacp),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Establish connection
	conn, err := grpc.Dial(config.Address, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", serviceName, err)
	}

	// Store connection
	cm.connections[serviceName] = conn

	// Initialize service clients
	switch serviceName {
	case "geo":
		cm.GeoClient = geopb.NewGeospatialServiceClient(conn)
		log.Printf("✅ Connected to Geo Service at %s", config.Address)
	case "user":
		cm.UserClient = userpb.NewUserServiceClient(conn)
		log.Printf("✅ Connected to User Service at %s", config.Address)
	case "trip":
		cm.TripClient = trippb.NewTripServiceClient(conn)
		log.Printf("✅ Connected to Trip Service at %s", config.Address)
	case "matching":
		cm.MatchingClient = matchingpb.NewMatchingServiceClient(conn)
		log.Printf("✅ Connected to Matching Service at %s", config.Address)
	case "pricing":
		cm.PricingClient = pricingpb.NewPricingServiceClient(conn)
		log.Printf("✅ Connected to Pricing Service at %s", config.Address)
	case "payment":
		cm.PaymentClient = paymentpb.NewPaymentServiceClient(conn)
		log.Printf("✅ Connected to Payment Service at %s", config.Address)
	}

	return nil
}

// HealthCheck checks the health of all connected services
func (cm *ClientManager) HealthCheck(ctx context.Context) map[string]bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	health := make(map[string]bool)

	for serviceName, conn := range cm.connections {
		health[serviceName] = cm.checkConnection(ctx, conn)
	}

	return health
}

// checkConnection verifies if a connection is healthy
func (cm *ClientManager) checkConnection(ctx context.Context, conn *grpc.ClientConn) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	state := conn.GetState()
	return state.String() == "READY" || state.String() == "IDLE"
}

// Close gracefully closes all connections
func (cm *ClientManager) Close() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	log.Println("Closing gRPC client connections...")

	for serviceName, conn := range cm.connections {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection to %s: %v", serviceName, err)
		} else {
			log.Printf("✅ Closed connection to %s", serviceName)
		}
	}

	return nil
}

// GetServiceConfig returns the configuration for a specific service
func (cm *ClientManager) GetServiceConfig(serviceName string) (ServiceConfig, bool) {
	config, exists := cm.config[serviceName]
	return config, exists
}

// UpdateServiceConfig updates the configuration for a specific service
func (cm *ClientManager) UpdateServiceConfig(serviceName string, config ServiceConfig) {
	cm.config[serviceName] = config
}

// Reconnect attempts to reconnect to a specific service
func (cm *ClientManager) Reconnect(serviceName string) error {
	config, exists := cm.config[serviceName]
	if !exists {
		return fmt.Errorf("service %s not configured", serviceName)
	}

	// Close existing connection if it exists
	cm.mutex.Lock()
	if conn, exists := cm.connections[serviceName]; exists {
		conn.Close()
		delete(cm.connections, serviceName)
	}
	cm.mutex.Unlock()

	// Reconnect
	return cm.connectService(serviceName, config)
}

// GetConnectionStatus returns the status of all connections
func (cm *ClientManager) GetConnectionStatus() map[string]string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	status := make(map[string]string)

	for serviceName, conn := range cm.connections {
		status[serviceName] = conn.GetState().String()
	}

	return status
}

// WithTimeout returns a context with the configured timeout for a service
func (cm *ClientManager) WithTimeout(ctx context.Context, serviceName string) (context.Context, context.CancelFunc) {
	config, exists := cm.config[serviceName]
	if !exists {
		// Default timeout
		return context.WithTimeout(ctx, 30*time.Second)
	}

	return context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds)*time.Second)
}
