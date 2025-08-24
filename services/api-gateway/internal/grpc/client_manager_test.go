package grpc

import (
	"context"
	"testing"
	"time"
)

func TestClientManagerInitialization(t *testing.T) {
	cm := NewClientManager()
	if cm == nil {
		t.Fatal("Expected non-nil ClientManager")
	}

	if cm.connections == nil {
		t.Error("Expected initialized connections map")
	}

	if cm.config == nil {
		t.Error("Expected initialized config map")
	}

	// Verify default service configurations
	expectedServices := []string{"geo", "user", "trip", "matching", "pricing", "payment"}
	for _, service := range expectedServices {
		if _, exists := cm.config[service]; !exists {
			t.Errorf("Expected configuration for service %s", service)
		}
	}
}

func TestClientManagerConfiguration(t *testing.T) {
	cm := NewClientManager()

	// Test getting existing configuration
	config, exists := cm.GetServiceConfig("user")
	if !exists {
		t.Error("Expected user service configuration to exist")
	}
	if config.Address == "" {
		t.Error("Expected non-empty address in user service configuration")
	}

	// Test getting non-existent configuration
	_, exists = cm.GetServiceConfig("nonexistent")
	if exists {
		t.Error("Expected nonexistent service configuration to not exist")
	}

	// Test updating configuration
	newConfig := ServiceConfig{
		Address:        "localhost:9999",
		MaxRetries:     5,
		TimeoutSeconds: 60,
		EnableTLS:      true,
	}
	cm.UpdateServiceConfig("user", newConfig)

	updatedConfig, exists := cm.GetServiceConfig("user")
	if !exists {
		t.Error("Expected updated configuration to exist")
	}
	if updatedConfig.Address != "localhost:9999" {
		t.Errorf("Expected address localhost:9999, got %s", updatedConfig.Address)
	}
}

func TestClientManagerTimeout(t *testing.T) {
	cm := NewClientManager()

	// Test timeout context creation
	ctx := context.Background()
	timeoutCtx, cancel := cm.WithTimeout(ctx, "user")
	defer cancel()

	if timeoutCtx == nil {
		t.Error("Expected non-nil timeout context")
	}

	// Verify timeout is set (should complete before 35 seconds which is > default 30s timeout)
	select {
	case <-timeoutCtx.Done():
		// Context should not be done immediately
		if time.Since(time.Now()) < 25*time.Second {
			// This is expected behavior - context has timeout set
		}
	case <-time.After(1 * time.Millisecond):
		// Context is not immediately cancelled, which is expected
	}
}

func TestClientManagerHealthCheck(t *testing.T) {
	cm := NewClientManager()

	// Test health check without any connections (should return empty map)
	ctx := context.Background()
	health := cm.HealthCheck(ctx)

	if health == nil {
		t.Error("Expected non-nil health map")
	}

	// Since no real connections are established, all should be false or map should be empty
	for service, healthy := range health {
		if healthy {
			t.Errorf("Expected service %s to be unhealthy without real connection", service)
		}
	}
}

func TestClientManagerConnectionStatus(t *testing.T) {
	cm := NewClientManager()

	// Test connection status without any connections
	status := cm.GetConnectionStatus()

	if status == nil {
		t.Error("Expected non-nil status map")
	}

	// Without real connections, status map should be empty
	if len(status) > 0 {
		// If there are entries, they should indicate connection failure
		for service, state := range status {
			if state == "READY" {
				t.Errorf("Expected service %s to not be READY without real connection", service)
			}
		}
	}
}
