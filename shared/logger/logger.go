package logger

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger represents the application logger
type Logger struct {
	*logrus.Logger
}

// Fields represents log fields
type Fields = logrus.Fields

// ContextKey represents context keys for logger
type ContextKey string

const (
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey ContextKey = "correlation_id"
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
)

// NewLogger creates a new logger instance
func NewLogger(level string, environment string) *Logger {
	log := logrus.New()

	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	log.SetLevel(logLevel)

	// Set formatter based on environment
	if environment == "production" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "function",
			},
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
			ForceColors:     true,
		})
	}

	// Set output
	log.SetOutput(os.Stdout)

	// Add default fields
	log.WithFields(logrus.Fields{
		"service":     "rideshare-platform",
		"environment": environment,
		"version":     "1.0.0",
	})

	return &Logger{Logger: log}
}

// WithContext creates a logger with context values
func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	entry := l.Logger.WithFields(logrus.Fields{})

	// Add correlation ID if present
	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		entry = entry.WithField("correlation_id", correlationID)
	}

	// Add user ID if present
	if userID := ctx.Value(UserIDKey); userID != nil {
		entry = entry.WithField("user_id", userID)
	}

	// Add request ID if present
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		entry = entry.WithField("request_id", requestID)
	}

	return entry
}

// WithFields creates a logger with additional fields
func (l *Logger) WithFields(fields Fields) *logrus.Entry {
	return l.Logger.WithFields(logrus.Fields(fields))
}

// WithError creates a logger with error field
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// WithService creates a logger with service field
func (l *Logger) WithService(service string) *logrus.Entry {
	return l.Logger.WithField("service", service)
}

// WithComponent creates a logger with component field
func (l *Logger) WithComponent(component string) *logrus.Entry {
	return l.Logger.WithField("component", component)
}

// WithOperation creates a logger with operation field
func (l *Logger) WithOperation(operation string) *logrus.Entry {
	return l.Logger.WithField("operation", operation)
}

// WithDuration creates a logger with duration field
func (l *Logger) WithDuration(duration time.Duration) *logrus.Entry {
	return l.Logger.WithField("duration_ms", duration.Milliseconds())
}

// LogRequest logs an HTTP request
func (l *Logger) LogRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	l.WithContext(ctx).WithFields(logrus.Fields{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
		"type":        "http_request",
	}).Info("HTTP request processed")
}

// LogGRPCRequest logs a gRPC request
func (l *Logger) LogGRPCRequest(ctx context.Context, method string, duration time.Duration, err error) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"method":      method,
		"duration_ms": duration.Milliseconds(),
		"type":        "grpc_request",
	})

	if err != nil {
		entry.WithError(err).Error("gRPC request failed")
	} else {
		entry.Info("gRPC request processed")
	}
}

// LogDatabaseQuery logs a database query
func (l *Logger) LogDatabaseQuery(ctx context.Context, query string, duration time.Duration, err error) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"query":       query,
		"duration_ms": duration.Milliseconds(),
		"type":        "database_query",
	})

	if err != nil {
		entry.WithError(err).Error("Database query failed")
	} else {
		entry.Debug("Database query executed")
	}
}

// LogCacheOperation logs a cache operation
func (l *Logger) LogCacheOperation(ctx context.Context, operation, key string, hit bool, duration time.Duration) {
	l.WithContext(ctx).WithFields(logrus.Fields{
		"operation":   operation,
		"key":         key,
		"hit":         hit,
		"duration_ms": duration.Milliseconds(),
		"type":        "cache_operation",
	}).Debug("Cache operation")
}

// LogBusinessEvent logs a business event
func (l *Logger) LogBusinessEvent(ctx context.Context, event string, entityID string, fields Fields) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"event":     event,
		"entity_id": entityID,
		"type":      "business_event",
	})

	if fields != nil {
		entry = entry.WithFields(logrus.Fields(fields))
	}

	entry.Info("Business event")
}

// LogSecurityEvent logs a security event
func (l *Logger) LogSecurityEvent(ctx context.Context, event string, severity string, fields Fields) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"event":    event,
		"severity": severity,
		"type":     "security_event",
	})

	if fields != nil {
		entry = entry.WithFields(logrus.Fields(fields))
	}

	entry.Warn("Security event")
}

// LogError logs an error with context
func (l *Logger) LogError(ctx context.Context, err error, message string, fields Fields) {
	entry := l.WithContext(ctx).WithError(err)

	if fields != nil {
		entry = entry.WithFields(logrus.Fields(fields))
	}

	entry.Error(message)
}

// LogPanic logs a panic with context
func (l *Logger) LogPanic(ctx context.Context, panicValue interface{}, stack []byte) {
	l.WithContext(ctx).WithFields(logrus.Fields{
		"panic_value": panicValue,
		"stack_trace": string(stack),
		"type":        "panic",
	}).Fatal("Application panic")
}

// LogMetric logs a metric
func (l *Logger) LogMetric(ctx context.Context, name string, value float64, tags map[string]string) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"metric_name":  name,
		"metric_value": value,
		"type":         "metric",
	})

	if tags != nil {
		for key, val := range tags {
			entry = entry.WithField("tag_"+key, val)
		}
	}

	entry.Info("Metric recorded")
}

// LogAuditEvent logs an audit event
func (l *Logger) LogAuditEvent(ctx context.Context, action, resource string, fields Fields) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"action":   action,
		"resource": resource,
		"type":     "audit_event",
	})

	if fields != nil {
		entry = entry.WithFields(logrus.Fields(fields))
	}

	entry.Info("Audit event")
}

// LogPerformance logs performance metrics
func (l *Logger) LogPerformance(ctx context.Context, operation string, duration time.Duration, fields Fields) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"operation":   operation,
		"duration_ms": duration.Milliseconds(),
		"type":        "performance",
	})

	if fields != nil {
		entry = entry.WithFields(logrus.Fields(fields))
	}

	// Log as warning if operation takes too long
	if duration > 5*time.Second {
		entry.Warn("Slow operation detected")
	} else {
		entry.Debug("Performance metric")
	}
}

// StructuredLog provides structured logging methods
type StructuredLog struct {
	logger *Logger
	ctx    context.Context
	fields Fields
}

// NewStructuredLog creates a new structured log instance
func (l *Logger) NewStructuredLog(ctx context.Context) *StructuredLog {
	return &StructuredLog{
		logger: l,
		ctx:    ctx,
		fields: make(Fields),
	}
}

// WithField adds a field to the structured log
func (sl *StructuredLog) WithField(key string, value interface{}) *StructuredLog {
	sl.fields[key] = value
	return sl
}

// WithFields adds multiple fields to the structured log
func (sl *StructuredLog) WithFields(fields Fields) *StructuredLog {
	for key, value := range fields {
		sl.fields[key] = value
	}
	return sl
}

// WithError adds an error to the structured log
func (sl *StructuredLog) WithError(err error) *StructuredLog {
	sl.fields["error"] = err.Error()
	return sl
}

// Info logs an info message
func (sl *StructuredLog) Info(message string) {
	sl.logger.WithContext(sl.ctx).WithFields(logrus.Fields(sl.fields)).Info(message)
}

// Warn logs a warning message
func (sl *StructuredLog) Warn(message string) {
	sl.logger.WithContext(sl.ctx).WithFields(logrus.Fields(sl.fields)).Warn(message)
}

// Error logs an error message
func (sl *StructuredLog) Error(message string) {
	sl.logger.WithContext(sl.ctx).WithFields(logrus.Fields(sl.fields)).Error(message)
}

// Debug logs a debug message
func (sl *StructuredLog) Debug(message string) {
	sl.logger.WithContext(sl.ctx).WithFields(logrus.Fields(sl.fields)).Debug(message)
}

// Fatal logs a fatal message and exits
func (sl *StructuredLog) Fatal(message string) {
	sl.logger.WithContext(sl.ctx).WithFields(logrus.Fields(sl.fields)).Fatal(message)
}

// Global logger instance
var globalLogger *Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(level, environment string) {
	globalLogger = NewLogger(level, environment)
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		globalLogger = NewLogger("info", "development")
	}
	return globalLogger
}

// Convenience functions using global logger

// Info logs an info message using global logger
func Info(message string) {
	GetGlobalLogger().Logger.Info(message)
}

// Warn logs a warning message using global logger
func Warn(message string) {
	GetGlobalLogger().Logger.Warn(message)
}

// Error logs an error message using global logger
func Error(message string) {
	GetGlobalLogger().Logger.Error(message)
}

// Debug logs a debug message using global logger
func Debug(message string) {
	GetGlobalLogger().Logger.Debug(message)
}

// Fatal logs a fatal message and exits using global logger
func Fatal(message string) {
	GetGlobalLogger().Logger.Fatal(message)
}

// WithFields creates a logger with fields using global logger
func WithFields(fields Fields) *logrus.Entry {
	return GetGlobalLogger().WithFields(fields)
}

// WithContext creates a logger with context using global logger
func WithContext(ctx context.Context) *logrus.Entry {
	return GetGlobalLogger().WithContext(ctx)
}
