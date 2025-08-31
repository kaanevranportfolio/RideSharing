package alerting

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rideshare-platform/shared/logger"
)

// AlertManager manages platform alerts and notifications
type AlertManager struct {
	redis    *redis.Client
	logger   *logger.Logger
	channels map[string]NotificationChannel
	rules    []*AlertRule
}

// AlertRule defines conditions that trigger alerts
type AlertRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Conditions  []AlertCondition       `json:"conditions"`
	Severity    AlertSeverity          `json:"severity"`
	Actions     []AlertAction          `json:"actions"`
	Cooldown    time.Duration          `json:"cooldown"`
	Enabled     bool                   `json:"enabled"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// AlertCondition defines a condition that must be met for an alert
type AlertCondition struct {
	Metric    string        `json:"metric"`
	Operator  string        `json:"operator"` // gt, lt, eq, gte, lte
	Threshold interface{}   `json:"threshold"`
	Duration  time.Duration `json:"duration"` // How long condition must persist
}

// AlertAction defines what to do when an alert fires
type AlertAction struct {
	Type    string                 `json:"type"` // email, slack, webhook, sms
	Target  string                 `json:"target"`
	Config  map[string]interface{} `json:"config"`
	Enabled bool                   `json:"enabled"`
}

// Alert represents an active or resolved alert
type Alert struct {
	ID          string                 `json:"id"`
	RuleID      string                 `json:"rule_id"`
	Severity    AlertSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Service     string                 `json:"service"`
	Metadata    map[string]interface{} `json:"metadata"`
	Status      AlertStatus            `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	AckedAt     *time.Time             `json:"acked_at,omitempty"`
	AckedBy     string                 `json:"acked_by,omitempty"`
}

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityWarning  AlertSeverity = "warning"
	SeverityInfo     AlertSeverity = "info"
)

// AlertStatus defines alert status
type AlertStatus string

const (
	StatusActive   AlertStatus = "active"
	StatusAcked    AlertStatus = "acknowledged"
	StatusResolved AlertStatus = "resolved"
)

// NotificationChannel interface for different notification methods
type NotificationChannel interface {
	Send(ctx context.Context, alert *Alert) error
	GetType() string
}

// MetricValue represents a metric value at a point in time
type MetricValue struct {
	Name      string            `json:"name"`
	Value     interface{}       `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// NewAlertManager creates a new alert manager
func NewAlertManager(redis *redis.Client, logger *logger.Logger) *AlertManager {
	am := &AlertManager{
		redis:    redis,
		logger:   logger,
		channels: make(map[string]NotificationChannel),
		rules:    []*AlertRule{},
	}

	// Initialize default alert rules
	am.initializeDefaultRules()

	// Initialize notification channels
	am.initializeChannels()

	return am
}

// initializeDefaultRules sets up default alerting rules
func (am *AlertManager) initializeDefaultRules() {
	defaultRules := []*AlertRule{
		{
			ID:          "high_error_rate",
			Name:        "High Error Rate",
			Description: "Error rate exceeds acceptable threshold",
			Conditions: []AlertCondition{
				{
					Metric:    "error_rate",
					Operator:  "gt",
					Threshold: 0.05, // 5%
					Duration:  5 * time.Minute,
				},
			},
			Severity: SeverityWarning,
			Actions: []AlertAction{
				{Type: "email", Target: "ops@rideshare.com", Enabled: true},
				{Type: "slack", Target: "#alerts", Enabled: true},
			},
			Cooldown: 15 * time.Minute,
			Enabled:  true,
		},
		{
			ID:          "service_down",
			Name:        "Service Unavailable",
			Description: "Service availability below critical threshold",
			Conditions: []AlertCondition{
				{
					Metric:    "availability",
					Operator:  "lt",
					Threshold: 0.90, // 90%
					Duration:  2 * time.Minute,
				},
			},
			Severity: SeverityCritical,
			Actions: []AlertAction{
				{Type: "email", Target: "oncall@rideshare.com", Enabled: true},
				{Type: "slack", Target: "#critical", Enabled: true},
				{Type: "sms", Target: "+1234567890", Enabled: true},
			},
			Cooldown: 5 * time.Minute,
			Enabled:  true,
		},
		{
			ID:          "high_response_time",
			Name:        "High Response Time",
			Description: "API response time exceeds acceptable threshold",
			Conditions: []AlertCondition{
				{
					Metric:    "response_time_p95",
					Operator:  "gt",
					Threshold: 2000.0, // 2 seconds
					Duration:  10 * time.Minute,
				},
			},
			Severity: SeverityWarning,
			Actions: []AlertAction{
				{Type: "slack", Target: "#performance", Enabled: true},
			},
			Cooldown: 20 * time.Minute,
			Enabled:  true,
		},
		{
			ID:          "no_available_drivers",
			Name:        "No Available Drivers",
			Description: "No drivers available for matching in major area",
			Conditions: []AlertCondition{
				{
					Metric:    "available_drivers",
					Operator:  "eq",
					Threshold: 0,
					Duration:  5 * time.Minute,
				},
			},
			Severity: SeverityWarning,
			Actions: []AlertAction{
				{Type: "email", Target: "operations@rideshare.com", Enabled: true},
				{Type: "slack", Target: "#operations", Enabled: true},
			},
			Cooldown: 10 * time.Minute,
			Enabled:  true,
		},
		{
			ID:          "payment_failure_spike",
			Name:        "Payment Failure Spike",
			Description: "Payment failure rate unusually high",
			Conditions: []AlertCondition{
				{
					Metric:    "payment_failure_rate",
					Operator:  "gt",
					Threshold: 0.10, // 10%
					Duration:  5 * time.Minute,
				},
			},
			Severity: SeverityWarning,
			Actions: []AlertAction{
				{Type: "email", Target: "finance@rideshare.com", Enabled: true},
				{Type: "slack", Target: "#payments", Enabled: true},
			},
			Cooldown: 15 * time.Minute,
			Enabled:  true,
		},
		{
			ID:          "database_connection_issues",
			Name:        "Database Connection Issues",
			Description: "High database query latency or connection errors",
			Conditions: []AlertCondition{
				{
					Metric:    "db_query_latency_p95",
					Operator:  "gt",
					Threshold: 5000.0, // 5 seconds
					Duration:  3 * time.Minute,
				},
			},
			Severity: SeverityCritical,
			Actions: []AlertAction{
				{Type: "email", Target: "dba@rideshare.com", Enabled: true},
				{Type: "slack", Target: "#database", Enabled: true},
			},
			Cooldown: 10 * time.Minute,
			Enabled:  true,
		},
	}

	am.rules = defaultRules
}

// initializeChannels sets up notification channels
func (am *AlertManager) initializeChannels() {
	// Initialize email channel
	am.channels["email"] = &EmailChannel{
		SMTPHost: "smtp.rideshare.com",
		SMTPPort: 587,
		Username: "alerts@rideshare.com",
		Password: "smtp_password", // In production, use secure config
		From:     "RideShare Alerts <alerts@rideshare.com>",
	}

	// Initialize Slack channel
	am.channels["slack"] = &SlackChannel{
		WebhookURL:     "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK",
		DefaultChannel: "#alerts",
	}

	// Initialize webhook channel
	am.channels["webhook"] = &WebhookChannel{
		DefaultURL: "https://api.rideshare.com/webhooks/alerts",
		Timeout:    10 * time.Second,
	}
}

// EvaluateMetrics evaluates incoming metrics against alert rules
func (am *AlertManager) EvaluateMetrics(ctx context.Context, metrics []*MetricValue) error {
	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		// Check if rule conditions are met
		conditionsMet := am.evaluateRuleConditions(ctx, rule, metrics)

		if conditionsMet {
			// Check if this alert is already active and within cooldown
			if am.isInCooldown(ctx, rule.ID) {
				continue
			}

			// Create and fire alert
			alert := &Alert{
				ID:          fmt.Sprintf("%s_%d", rule.ID, time.Now().Unix()),
				RuleID:      rule.ID,
				Severity:    rule.Severity,
				Title:       rule.Name,
				Description: rule.Description,
				Status:      StatusActive,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Metadata:    rule.Metadata,
			}

			if err := am.fireAlert(ctx, alert, rule.Actions); err != nil {
				am.logger.WithError(err).Error("Failed to fire alert", "alert_id", alert.ID)
			}
		}
	}

	return nil
}

// evaluateRuleConditions checks if all conditions for a rule are met
func (am *AlertManager) evaluateRuleConditions(ctx context.Context, rule *AlertRule, metrics []*MetricValue) bool {
	for _, condition := range rule.Conditions {
		if !am.evaluateCondition(ctx, condition, metrics) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single condition against metrics
func (am *AlertManager) evaluateCondition(ctx context.Context, condition AlertCondition, metrics []*MetricValue) bool {
	// Find the metric we're looking for
	var metricValue *MetricValue
	for _, metric := range metrics {
		if metric.Name == condition.Metric {
			metricValue = metric
			break
		}
	}

	if metricValue == nil {
		return false
	}

	// Compare the metric value against the threshold
	return am.compareValues(metricValue.Value, condition.Operator, condition.Threshold)
}

// compareValues compares two values using the specified operator
func (am *AlertManager) compareValues(value interface{}, operator string, threshold interface{}) bool {
	// Convert values to float64 for comparison
	v1, ok1 := toFloat64(value)
	v2, ok2 := toFloat64(threshold)

	if !ok1 || !ok2 {
		return false
	}

	switch operator {
	case "gt":
		return v1 > v2
	case "gte":
		return v1 >= v2
	case "lt":
		return v1 < v2
	case "lte":
		return v1 <= v2
	case "eq":
		return v1 == v2
	default:
		return false
	}
}

// isInCooldown checks if an alert rule is in cooldown period
func (am *AlertManager) isInCooldown(ctx context.Context, ruleID string) bool {
	if am.redis == nil {
		return false
	}

	key := fmt.Sprintf("alert_cooldown:%s", ruleID)
	exists, err := am.redis.Exists(ctx, key).Result()
	return err == nil && exists > 0
}

// fireAlert creates and sends an alert
func (am *AlertManager) fireAlert(ctx context.Context, alert *Alert, actions []AlertAction) error {
	// Store alert in Redis
	if am.redis != nil {
		alertData, _ := json.Marshal(alert)
		alertKey := fmt.Sprintf("alert:%s", alert.ID)
		am.redis.SetEx(ctx, alertKey, alertData, 24*time.Hour)

		// Add to active alerts list
		am.redis.ZAdd(ctx, "active_alerts", redis.Z{
			Score:  float64(alert.CreatedAt.Unix()),
			Member: alert.ID,
		})

		// Set cooldown for this rule
		cooldownKey := fmt.Sprintf("alert_cooldown:%s", alert.RuleID)
		rule := am.findRule(alert.RuleID)
		if rule != nil {
			am.redis.SetEx(ctx, cooldownKey, "1", rule.Cooldown)
		}
	}

	// Send notifications
	for _, action := range actions {
		if !action.Enabled {
			continue
		}

		if channel, exists := am.channels[action.Type]; exists {
			if err := channel.Send(ctx, alert); err != nil {
				am.logger.WithError(err).Error("Failed to send alert notification",
					"alert_id", alert.ID, "channel", action.Type)
			} else {
				am.logger.WithFields(logger.Fields{
					"alert_id": alert.ID,
					"channel":  action.Type,
					"target":   action.Target,
				}).Info("Alert notification sent")
			}
		}
	}

	am.logger.WithFields(logger.Fields{
		"alert_id": alert.ID,
		"rule_id":  alert.RuleID,
		"severity": alert.Severity,
		"title":    alert.Title,
	}).Warn("Alert fired")

	return nil
}

// ResolveAlert marks an alert as resolved
func (am *AlertManager) ResolveAlert(ctx context.Context, alertID string) error {
	if am.redis == nil {
		return fmt.Errorf("redis client not available")
	}

	// Get alert
	alertKey := fmt.Sprintf("alert:%s", alertID)
	alertData, err := am.redis.Get(ctx, alertKey).Result()
	if err != nil {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	var alert Alert
	if err := json.Unmarshal([]byte(alertData), &alert); err != nil {
		return fmt.Errorf("failed to unmarshal alert: %w", err)
	}

	// Update alert status
	now := time.Now()
	alert.Status = StatusResolved
	alert.ResolvedAt = &now
	alert.UpdatedAt = now

	// Save updated alert
	updatedData, _ := json.Marshal(alert)
	am.redis.SetEx(ctx, alertKey, updatedData, 24*time.Hour)

	// Remove from active alerts
	am.redis.ZRem(ctx, "active_alerts", alertID)

	// Add to resolved alerts
	am.redis.ZAdd(ctx, "resolved_alerts", redis.Z{
		Score:  float64(now.Unix()),
		Member: alertID,
	})

	am.logger.WithField("alert_id", alertID).Info("Alert resolved")
	return nil
}

// AcknowledgeAlert marks an alert as acknowledged
func (am *AlertManager) AcknowledgeAlert(ctx context.Context, alertID, ackedBy string) error {
	if am.redis == nil {
		return fmt.Errorf("redis client not available")
	}

	// Get alert
	alertKey := fmt.Sprintf("alert:%s", alertID)
	alertData, err := am.redis.Get(ctx, alertKey).Result()
	if err != nil {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	var alert Alert
	if err := json.Unmarshal([]byte(alertData), &alert); err != nil {
		return fmt.Errorf("failed to unmarshal alert: %w", err)
	}

	// Update alert status
	now := time.Now()
	alert.Status = StatusAcked
	alert.AckedAt = &now
	alert.AckedBy = ackedBy
	alert.UpdatedAt = now

	// Save updated alert
	updatedData, _ := json.Marshal(alert)
	am.redis.SetEx(ctx, alertKey, updatedData, 24*time.Hour)

	am.logger.WithFields(logger.Fields{
		"alert_id": alertID,
		"acked_by": ackedBy,
	}).Info("Alert acknowledged")

	return nil
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts(ctx context.Context) ([]*Alert, error) {
	if am.redis == nil {
		return []*Alert{}, nil
	}

	// Get active alert IDs
	alertIDs, err := am.redis.ZRevRange(ctx, "active_alerts", 0, -1).Result()
	if err != nil {
		return nil, err
	}

	var alerts []*Alert
	for _, alertID := range alertIDs {
		alertKey := fmt.Sprintf("alert:%s", alertID)
		alertData, err := am.redis.Get(ctx, alertKey).Result()
		if err != nil {
			continue
		}

		var alert Alert
		if err := json.Unmarshal([]byte(alertData), &alert); err != nil {
			continue
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// GetAlertHistory returns alert history
func (am *AlertManager) GetAlertHistory(ctx context.Context, hours int) ([]*Alert, error) {
	if am.redis == nil {
		return []*Alert{}, nil
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	// Get recent alert IDs from both active and resolved
	activeIDs, _ := am.redis.ZRangeByScore(ctx, "active_alerts", &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", since.Unix()),
		Max: "+inf",
	}).Result()

	resolvedIDs, _ := am.redis.ZRangeByScore(ctx, "resolved_alerts", &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", since.Unix()),
		Max: "+inf",
	}).Result()

	// Combine and deduplicate
	allIDs := append(activeIDs, resolvedIDs...)
	idSet := make(map[string]bool)
	for _, id := range allIDs {
		idSet[id] = true
	}

	var alerts []*Alert
	for alertID := range idSet {
		alertKey := fmt.Sprintf("alert:%s", alertID)
		alertData, err := am.redis.Get(ctx, alertKey).Result()
		if err != nil {
			continue
		}

		var alert Alert
		if err := json.Unmarshal([]byte(alertData), &alert); err != nil {
			continue
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// AddRule adds a new alert rule
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.rules = append(am.rules, rule)
}

// findRule finds a rule by ID
func (am *AlertManager) findRule(ruleID string) *AlertRule {
	for _, rule := range am.rules {
		if rule.ID == ruleID {
			return rule
		}
	}
	return nil
}

// Helper function to convert interface{} to float64
func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

// Notification channel implementations

// EmailChannel sends alerts via email
type EmailChannel struct {
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
}

func (ec *EmailChannel) Send(ctx context.Context, alert *Alert) error {
	// Mock email sending implementation
	subject := fmt.Sprintf("[%s] %s", strings.ToUpper(string(alert.Severity)), alert.Title)
	body := fmt.Sprintf("Alert: %s\n\nDescription: %s\n\nService: %s\n\nCreated: %s",
		alert.Title, alert.Description, alert.Service, alert.CreatedAt.Format(time.RFC3339))

	fmt.Printf("EMAIL ALERT:\nTo: ops@rideshare.com\nSubject: %s\nBody: %s\n", subject, body)
	return nil
}

func (ec *EmailChannel) GetType() string {
	return "email"
}

// SlackChannel sends alerts to Slack
type SlackChannel struct {
	WebhookURL     string
	DefaultChannel string
}

func (sc *SlackChannel) Send(ctx context.Context, alert *Alert) error {
	// Mock Slack notification implementation
	message := fmt.Sprintf("🚨 *%s Alert*: %s\n_%s_\nService: %s\nTime: %s",
		strings.ToUpper(string(alert.Severity)), alert.Title, alert.Description,
		alert.Service, alert.CreatedAt.Format(time.RFC3339))

	fmt.Printf("SLACK ALERT:\nChannel: %s\nMessage: %s\n", sc.DefaultChannel, message)
	return nil
}

func (sc *SlackChannel) GetType() string {
	return "slack"
}

// WebhookChannel sends alerts via HTTP webhook
type WebhookChannel struct {
	DefaultURL string
	Timeout    time.Duration
}

func (wc *WebhookChannel) Send(ctx context.Context, alert *Alert) error {
	// Mock webhook implementation
	alertJSON, _ := json.Marshal(alert)
	fmt.Printf("WEBHOOK ALERT:\nURL: %s\nPayload: %s\n", wc.DefaultURL, string(alertJSON))
	return nil
}

func (wc *WebhookChannel) GetType() string {
	return "webhook"
}
