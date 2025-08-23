# Security Architecture

This document outlines the comprehensive security architecture for the rideshare platform, covering authentication, authorization, data protection, and security best practices.

## Security Principles

### Core Security Principles
- **Zero Trust Architecture**: Never trust, always verify
- **Defense in Depth**: Multiple layers of security controls
- **Principle of Least Privilege**: Minimal access rights
- **Security by Design**: Security integrated from the start
- **Data Minimization**: Collect and store only necessary data
- **Fail Secure**: System fails to a secure state

### Compliance Requirements
- **PCI DSS**: Payment card data protection
- **GDPR**: European data protection regulation
- **CCPA**: California consumer privacy act
- **SOC 2**: Security and availability controls
- **ISO 27001**: Information security management

## Authentication Architecture

### Multi-Factor Authentication (MFA)

#### JWT-Based Authentication

```go
// shared/auth/jwt.go
type JWTManager struct {
    secretKey     string
    tokenDuration time.Duration
    refreshDuration time.Duration
}

type Claims struct {
    UserID    string   `json:"user_id"`
    Email     string   `json:"email"`
    UserType  string   `json:"user_type"`
    Roles     []string `json:"roles"`
    SessionID string   `json:"session_id"`
    jwt.RegisteredClaims
}

func (manager *JWTManager) GenerateToken(user *User) (*TokenPair, error) {
    claims := &Claims{
        UserID:   user.ID,
        Email:    user.Email,
        UserType: user.Type,
        Roles:    user.Roles,
        SessionID: generateSessionID(),
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(manager.tokenDuration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "rideshare-platform",
            Subject:   user.ID,
            ID:        generateJTI(),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    accessToken, err := token.SignedString([]byte(manager.secretKey))
    if err != nil {
        return nil, err
    }

    refreshToken, err := manager.generateRefreshToken(user.ID)
    if err != nil {
        return nil, err
    }

    return &TokenPair{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresIn:    int(manager.tokenDuration.Seconds()),
    }, nil
}

func (manager *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(manager.secretKey), nil
    })

    if err != nil {
        return nil, err
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token")
    }

    // Check if token is blacklisted
    if manager.isTokenBlacklisted(claims.ID) {
        return nil, errors.New("token is blacklisted")
    }

    return claims, nil
}
```

#### OAuth 2.0 Integration

```go
// shared/auth/oauth.go
type OAuthProvider struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
    Scopes       []string
}

func (p *OAuthProvider) GetAuthURL(state string) string {
    return fmt.Sprintf(
        "https://accounts.google.com/o/oauth2/auth?client_id=%s&redirect_uri=%s&scope=%s&response_type=code&state=%s",
        p.ClientID,
        url.QueryEscape(p.RedirectURL),
        url.QueryEscape(strings.Join(p.Scopes, " ")),
        state,
    )
}

func (p *OAuthProvider) ExchangeCodeForToken(code string) (*OAuthToken, error) {
    data := url.Values{}
    data.Set("client_id", p.ClientID)
    data.Set("client_secret", p.ClientSecret)
    data.Set("code", code)
    data.Set("grant_type", "authorization_code")
    data.Set("redirect_uri", p.RedirectURL)

    resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var token OAuthToken
    if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
        return nil, err
    }

    return &token, nil
}
```

### Session Management

#### Redis-Based Session Store

```go
// shared/auth/session.go
type SessionManager struct {
    redis  *redis.Client
    prefix string
    ttl    time.Duration
}

type Session struct {
    ID        string    `json:"id"`
    UserID    string    `json:"user_id"`
    UserType  string    `json:"user_type"`
    IPAddress string    `json:"ip_address"`
    UserAgent string    `json:"user_agent"`
    CreatedAt time.Time `json:"created_at"`
    LastSeen  time.Time `json:"last_seen"`
    Active    bool      `json:"active"`
}

func (sm *SessionManager) CreateSession(userID, ipAddress, userAgent string) (*Session, error) {
    session := &Session{
        ID:        generateSessionID(),
        UserID:    userID,
        IPAddress: ipAddress,
        UserAgent: userAgent,
        CreatedAt: time.Now(),
        LastSeen:  time.Now(),
        Active:    true,
    }

    sessionData, err := json.Marshal(session)
    if err != nil {
        return nil, err
    }

    key := fmt.Sprintf("%s:session:%s", sm.prefix, session.ID)
    err = sm.redis.Set(context.Background(), key, sessionData, sm.ttl).Err()
    if err != nil {
        return nil, err
    }

    // Track user sessions
    userSessionsKey := fmt.Sprintf("%s:user_sessions:%s", sm.prefix, userID)
    sm.redis.SAdd(context.Background(), userSessionsKey, session.ID)
    sm.redis.Expire(context.Background(), userSessionsKey, sm.ttl)

    return session, nil
}

func (sm *SessionManager) ValidateSession(sessionID string) (*Session, error) {
    key := fmt.Sprintf("%s:session:%s", sm.prefix, sessionID)
    sessionData, err := sm.redis.Get(context.Background(), key).Result()
    if err != nil {
        if err == redis.Nil {
            return nil, errors.New("session not found")
        }
        return nil, err
    }

    var session Session
    if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
        return nil, err
    }

    if !session.Active {
        return nil, errors.New("session is inactive")
    }

    // Update last seen
    session.LastSeen = time.Now()
    updatedData, _ := json.Marshal(session)
    sm.redis.Set(context.Background(), key, updatedData, sm.ttl)

    return &session, nil
}

func (sm *SessionManager) RevokeSession(sessionID string) error {
    key := fmt.Sprintf("%s:session:%s", sm.prefix, sessionID)
    return sm.redis.Del(context.Background(), key).Err()
}

func (sm *SessionManager) RevokeAllUserSessions(userID string) error {
    userSessionsKey := fmt.Sprintf("%s:user_sessions:%s", sm.prefix, userID)
    sessionIDs, err := sm.redis.SMembers(context.Background(), userSessionsKey).Result()
    if err != nil {
        return err
    }

    for _, sessionID := range sessionIDs {
        sm.RevokeSession(sessionID)
    }

    return sm.redis.Del(context.Background(), userSessionsKey).Err()
}
```

## Authorization Architecture

### Role-Based Access Control (RBAC)

#### Role and Permission System

```go
// shared/auth/rbac.go
type Permission struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Resource string `json:"resource"`
    Action   string `json:"action"`
}

type Role struct {
    ID          string       `json:"id"`
    Name        string       `json:"name"`
    Description string       `json:"description"`
    Permissions []Permission `json:"permissions"`
}

type RBACManager struct {
    roles       map[string]*Role
    userRoles   map[string][]string
    permissions map[string]*Permission
}

func (rbac *RBACManager) HasPermission(userID, resource, action string) bool {
    userRoles, exists := rbac.userRoles[userID]
    if !exists {
        return false
    }

    for _, roleID := range userRoles {
        role, exists := rbac.roles[roleID]
        if !exists {
            continue
        }

        for _, permission := range role.Permissions {
            if permission.Resource == resource && permission.Action == action {
                return true
            }
            // Check wildcard permissions
            if permission.Resource == "*" || permission.Action == "*" {
                return true
            }
        }
    }

    return false
}

func (rbac *RBACManager) AssignRole(userID, roleID string) error {
    if _, exists := rbac.roles[roleID]; !exists {
        return errors.New("role does not exist")
    }

    if rbac.userRoles[userID] == nil {
        rbac.userRoles[userID] = []string{}
    }

    // Check if role already assigned
    for _, existingRole := range rbac.userRoles[userID] {
        if existingRole == roleID {
            return nil // Already assigned
        }
    }

    rbac.userRoles[userID] = append(rbac.userRoles[userID], roleID)
    return nil
}
```

#### GraphQL Authorization Middleware

```go
// services/api-gateway/internal/middleware/auth.go
func AuthMiddleware(rbac *RBACManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := jwtManager.ValidateToken(tokenString)
        if err != nil {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // Set user context
        c.Set("user_id", claims.UserID)
        c.Set("user_type", claims.UserType)
        c.Set("user_roles", claims.Roles)

        c.Next()
    }
}

func RequirePermission(resource, action string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, exists := c.Get("user_id")
        if !exists {
            c.JSON(403, gin.H{"error": "User not authenticated"})
            c.Abort()
            return
        }

        if !rbacManager.HasPermission(userID.(string), resource, action) {
            c.JSON(403, gin.H{"error": "Insufficient permissions"})
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### API Security

#### Rate Limiting

```go
// shared/middleware/ratelimit.go
type RateLimiter struct {
    redis  *redis.Client
    window time.Duration
}

func (rl *RateLimiter) Allow(key string, limit int) (bool, error) {
    ctx := context.Background()
    now := time.Now()
    windowStart := now.Truncate(rl.window)
    
    pipe := rl.redis.Pipeline()
    
    // Count requests in current window
    countKey := fmt.Sprintf("rate_limit:%s:%d", key, windowStart.Unix())
    pipe.Incr(ctx, countKey)
    pipe.Expire(ctx, countKey, rl.window)
    
    results, err := pipe.Exec(ctx)
    if err != nil {
        return false, err
    }
    
    count := results[0].(*redis.IntCmd).Val()
    return count <= int64(limit), nil
}

func RateLimitMiddleware(limiter *RateLimiter, limit int) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Use user ID if authenticated, otherwise IP address
        key := c.ClientIP()
        if userID, exists := c.Get("user_id"); exists {
            key = fmt.Sprintf("user:%s", userID)
        }
        
        allowed, err := limiter.Allow(key, limit)
        if err != nil {
            c.JSON(500, gin.H{"error": "Rate limiting error"})
            c.Abort()
            return
        }
        
        if !allowed {
            c.JSON(429, gin.H{"error": "Rate limit exceeded"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

#### Input Validation and Sanitization

```go
// shared/validation/validator.go
type Validator struct {
    validate *validator.Validate
}

func NewValidator() *Validator {
    v := validator.New()
    
    // Custom validators
    v.RegisterValidation("phone", validatePhone)
    v.RegisterValidation("coordinates", validateCoordinates)
    v.RegisterValidation("vehicle_type", validateVehicleType)
    
    return &Validator{validate: v}
}

func (v *Validator) ValidateStruct(s interface{}) error {
    return v.validate.Struct(s)
}

func validatePhone(fl validator.FieldLevel) bool {
    phone := fl.Field().String()
    phoneRegex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
    return phoneRegex.MatchString(phone)
}

func validateCoordinates(fl validator.FieldLevel) bool {
    coord := fl.Field().Float()
    fieldName := fl.FieldName()
    
    if fieldName == "Latitude" {
        return coord >= -90 && coord <= 90
    }
    if fieldName == "Longitude" {
        return coord >= -180 && coord <= 180
    }
    
    return false
}

// Input sanitization
func SanitizeInput(input string) string {
    // Remove potentially dangerous characters
    input = html.EscapeString(input)
    input = strings.TrimSpace(input)
    
    // Remove SQL injection patterns
    sqlPatterns := []string{
        `(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute)`,
        `(?i)(script|javascript|vbscript|onload|onerror|onclick)`,
        `[<>\"'%;()&+]`,
    }
    
    for _, pattern := range sqlPatterns {
        re := regexp.MustCompile(pattern)
        input = re.ReplaceAllString(input, "")
    }
    
    return input
}
```

## Data Protection

### Encryption at Rest

#### Database Encryption

```sql
-- PostgreSQL encryption setup
-- Enable transparent data encryption
ALTER SYSTEM SET ssl = on;
ALTER SYSTEM SET ssl_cert_file = '/path/to/server.crt';
ALTER SYSTEM SET ssl_key_file = '/path/to/server.key';

-- Encrypt sensitive columns
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Encrypted user data
CREATE TABLE users_encrypted (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email_encrypted BYTEA NOT NULL,
    phone_encrypted BYTEA NOT NULL,
    -- Use application-level encryption for PII
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Encryption functions
CREATE OR REPLACE FUNCTION encrypt_pii(data TEXT, key TEXT)
RETURNS BYTEA AS $$
BEGIN
    RETURN pgp_sym_encrypt(data, key);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION decrypt_pii(encrypted_data BYTEA, key TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN pgp_sym_decrypt(encrypted_data, key);
END;
$$ LANGUAGE plpgsql;
```

#### Application-Level Encryption

```go
// shared/crypto/encryption.go
type EncryptionManager struct {
    key []byte
}

func NewEncryptionManager(key string) *EncryptionManager {
    return &EncryptionManager{
        key: []byte(key),
    }
}

func (em *EncryptionManager) Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(em.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (em *EncryptionManager) Decrypt(ciphertext string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }

    block, err := aes.NewCipher(em.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", errors.New("ciphertext too short")
    }

    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}

// PII encryption for sensitive data
func (em *EncryptionManager) EncryptPII(data *PIIData) (*EncryptedPIIData, error) {
    encryptedEmail, err := em.Encrypt(data.Email)
    if err != nil {
        return nil, err
    }

    encryptedPhone, err := em.Encrypt(data.Phone)
    if err != nil {
        return nil, err
    }

    return &EncryptedPIIData{
        ID:            data.ID,
        EmailHash:     hashEmail(data.Email), // For indexing
        EncryptedEmail: encryptedEmail,
        EncryptedPhone: encryptedPhone,
        CreatedAt:     data.CreatedAt,
    }, nil
}
```

### Encryption in Transit

#### TLS Configuration

```go
// shared/tls/config.go
func GetTLSConfig() *tls.Config {
    return &tls.Config{
        MinVersion:               tls.VersionTLS12,
        CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
        PreferServerCipherSuites: true,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
    }
}

// gRPC TLS setup
func NewTLSCredentials(certFile, keyFile string) (credentials.TransportCredentials, error) {
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, err
    }

    config := &tls.Config{
        Certificates: []tls.Certificate{cert},
        ClientAuth:   tls.RequireAndVerifyClientCert,
    }

    return credentials.NewTLS(config), nil
}
```

## Security Monitoring and Incident Response

### Security Event Logging

```go
// shared/security/audit.go
type SecurityEvent struct {
    ID        string    `json:"id"`
    Type      string    `json:"type"`
    UserID    string    `json:"user_id,omitempty"`
    IPAddress string    `json:"ip_address"`
    UserAgent string    `json:"user_agent"`
    Resource  string    `json:"resource"`
    Action    string    `json:"action"`
    Result    string    `json:"result"` // success, failure, blocked
    Details   map[string]interface{} `json:"details"`
    Timestamp time.Time `json:"timestamp"`
    Severity  string    `json:"severity"` // low, medium, high, critical
}

type SecurityLogger struct {
    logger *logrus.Logger
    redis  *redis.Client
}

func (sl *SecurityLogger) LogSecurityEvent(event *SecurityEvent) {
    event.ID = uuid.New().String()
    event.Timestamp = time.Now()

    // Log to structured logger
    sl.logger.WithFields(logrus.Fields{
        "event_id":   event.ID,
        "event_type": event.Type,
        "user_id":    event.UserID,
        "ip_address": event.IPAddress,
        "resource":   event.Resource,
        "action":     event.Action,
        "result":     event.Result,
        "severity":   event.Severity,
        "details":    event.Details,
    }).Info("Security event")

    // Store in Redis for real-time monitoring
    eventData, _ := json.Marshal(event)
    sl.redis.LPush(context.Background(), "security_events", eventData)
    sl.redis.LTrim(context.Background(), "security_events", 0, 10000) // Keep last 10k events

    // Trigger alerts for high severity events
    if event.Severity == "high" || event.Severity == "critical" {
        sl.triggerAlert(event)
    }
}

func (sl *SecurityLogger) triggerAlert(event *SecurityEvent) {
    // Send to monitoring system
    alert := map[string]interface{}{
        "alert_type": "security_incident",
        "severity":   event.Severity,
        "event":      event,
        "timestamp":  time.Now(),
    }

    alertData, _ := json.Marshal(alert)
    sl.redis.Publish(context.Background(), "security_alerts", alertData)
}
```

### Intrusion Detection

```go
// shared/security/ids.go
type IntrusionDetectionSystem struct {
    redis     *redis.Client
    logger    *SecurityLogger
    rules     []DetectionRule
    whitelist map[string]bool
}

type DetectionRule struct {
    Name        string
    Pattern     string
    Threshold   int
    TimeWindow  time.Duration
    Severity    string
    Action      string // log, block, alert
}

func (ids *IntrusionDetectionSystem) AnalyzeRequest(req *http.Request, userID string) {
    ipAddress := getClientIP(req)
    
    // Check for suspicious patterns
    for _, rule := range ids.rules {
        if ids.matchesRule(req, rule) {
            ids.handleRuleMatch(rule, ipAddress, userID, req)
        }
    }
    
    // Rate-based detection
    ids.checkRateLimits(ipAddress, userID)
    
    // Behavioral analysis
    ids.analyzeBehavior(userID, req)
}

func (ids *IntrusionDetectionSystem) matchesRule(req *http.Request, rule DetectionRule) bool {
    // Check URL patterns
    if matched, _ := regexp.MatchString(rule.Pattern, req.URL.Path); matched {
        return true
    }
    
    // Check headers
    for _, header := range req.Header {
        for _, value := range header {
            if matched, _ := regexp.MatchString(rule.Pattern, value); matched {
                return true
            }
        }
    }
    
    return false
}

func (ids *IntrusionDetectionSystem) handleRuleMatch(rule DetectionRule, ipAddress, userID string, req *http.Request) {
    event := &SecurityEvent{
        Type:      "intrusion_attempt",
        UserID:    userID,
        IPAddress: ipAddress,
        UserAgent: req.UserAgent(),
        Resource:  req.URL.Path,
        Action:    req.Method,
        Result:    "blocked",
        Severity:  rule.Severity,
        Details: map[string]interface{}{
            "rule_name": rule.Name,
            "pattern":   rule.Pattern,
        },
    }
    
    ids.logger.LogSecurityEvent(event)
    
    if rule.Action == "block" {
        ids.blockIP(ipAddress, rule.TimeWindow)
    }
}

func (ids *IntrusionDetectionSystem) blockIP(ipAddress string, duration time.Duration) {
    key := fmt.Sprintf("blocked_ip:%s", ipAddress)
    ids.redis.Set(context.Background(), key, "blocked", duration)
}
```

## Vulnerability Management

### Security Scanning

```yaml
# .github/workflows/security-scan.yml
name: Security Scan

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]
  schedule:
    - cron: '0 2 * * *' # Daily at 2 AM

jobs:
  dependency-scan:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Run Snyk to check for vulnerabilities
      uses: snyk/actions/golang@master
      env:
        SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
      with:
        args: --severity-threshold=high
    
    - name: Run Nancy for Go dependencies
      run: |
        go list -json -m all | nancy sleuth

  static-analysis:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Run Gosec Security Scanner
      uses: securecodewarrior/github-action-gosec@master
      with:
        args: '-fmt sarif -out gosec.sarif ./...'
    
    - name: Upload SARIF file
      uses: github/codeql-action/upload-sarif@v2
      with:
        sarif_file: gosec.sarif

  container-scan:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Build Docker images
      run: |
        docker build -t rideshare/user-service:latest services/user-service/
        docker build -t rideshare/api-gateway:latest services/api-gateway/
    
    - name: Run Trivy vulnerability scanner
      uses: aquasecurity/trivy-action@master
      with:
        image-ref: 'rideshare/user-service:latest'
        format: 'sarif'
        output: 'trivy-results.sarif'
    
    - name: Upload Trivy scan results
      uses: github/codeql-action/upload-sarif@v2
      with:
        sarif_file: 'trivy-results.sarif'
```

### Security Headers

```go
// shared/middleware/security.go
func SecurityHeadersMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Prevent clickjacking
        c.Header("X-Frame-Options", "DENY")
        
        // Prevent MIME type sniffing
        c.Header("X-Content-Type-Options", "nosniff")
        
        // XSS protection
        c.Header("X-XSS-Protection", "1; mode=block")
        
        // Strict transport security
        c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        
        // Content security policy
        c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
        
        // Referrer policy
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        
        // Permissions policy
        c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
        
        c.Next()
    }
}

func CORSMiddleware() gin.HandlerFunc {
    return cors.New(cors.Config{
        AllowOrigins:     []string{"https://app.rideshare.com", "https://admin.rideshare.com"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    })
}
```

## Incident Response Plan

### Security Incident Classification

#### Severity Levels
- **Critical**: Data breach, system compromise, service unavailable
- **High**: Unauthorized access attempt, privilege escalation
- **Medium**: Suspicious activity, policy violation
- **Low**: Failed login attempts, minor security events

#### Response Procedures

```go
// shared/security/incident.go
type IncidentResponse struct {
    logger    *SecurityLogger
    notifier  *AlertNotifier
    escalator *EscalationManager
}

func (ir *IncidentResponse) HandleIncident(incident *SecurityIncident) {
    // Log the incident
    ir.logger.LogSecurityEvent(&SecurityEvent{
        Type:     "security_incident",
        Severity: incident.Severity,
        Details:  incident.Details,
    })
    
    // Immediate response based on severity
    switch incident.Severity {
    case "critical":
        ir.handleCriticalIncident(incident)
    case "high":
        ir.handleHighIncident(incident)
    case "medium":
        ir.handleMediumIncident(incident)
    case "low":
        ir.handleLowIncident(incident)
    }
    
    // Escalate if needed
    ir.escalator.EscalateIfNeeded(incident)
}

func (ir *IncidentResponse) handleCriticalIncident(incident *SecurityIncident) {
    // Immediate actions
    ir.notifier.SendImmediateAlert(incident)
    ir.activateIncidentResponse(incident)
    
    // Containment
    if incident.Type == "data_breach" {
        ir.containDataBreach(incident)
    }
    
    // Communication
    ir.notifyStakeholders(incident)
}
```

This comprehensive security architecture provides multiple layers of protection,