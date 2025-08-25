package metrics

import (
	"time"
)

// BasicMetrics provides simple metrics collection without prometheus
type BasicMetrics struct {
	RequestCount int64
	StartTime    time.Time
}

// NewBasicMetrics creates a new basic metrics collector
func NewBasicMetrics() *BasicMetrics {
	return &BasicMetrics{
		RequestCount: 0,
		StartTime:    time.Now(),
	}
}

// IncrementRequests increments the request counter
func (m *BasicMetrics) IncrementRequests() {
	m.RequestCount++
}

// GetUptime returns the uptime duration
func (m *BasicMetrics) GetUptime() time.Duration {
	return time.Since(m.StartTime)
}

// GetStats returns basic statistics
func (m *BasicMetrics) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"request_count": m.RequestCount,
		"uptime":        m.GetUptime().String(),
		"start_time":    m.StartTime,
	}
}
