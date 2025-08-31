# 👤 USER SERVICE - DEEP DIVE

## 📋 Overview
The **User Service** is the foundation of the rideshare platform, managing all user-related operations including authentication, profile management, role-based access control, and driver verification. It serves as the single source of truth for user identity and permissions across the entire system.

---

## 🎯 Core Responsibilities

### **1. User Management & Authentication**
```go
type UserService struct {
    userRepo     UserRepository
    authService  AuthenticationService
    cryptoService CryptographyService
    emailService EmailService
    smsService   SMSService
    auditLogger  AuditLogger
}

type User struct {
    ID          string    `json:"id" db:"id"`
    Email       string    `json:"email" db:"email"`
    Phone       string    `json:"phone" db:"phone"`
    FirstName   string    `json:"first_name" db:"first_name"`
    LastName    string    `json:"last_name" db:"last_name"`
    Role        UserRole  `json:"role" db:"role"`
    Status      UserStatus `json:"status" db:"status"`
    ProfilePicture string `json:"profile_picture" db:"profile_picture"`
    DateOfBirth time.Time `json:"date_of_birth" db:"date_of_birth"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
    LastLogin   *time.Time `json:"last_login" db:"last_login"`
    
    // Verification status
    EmailVerified bool      `json:"email_verified" db:"email_verified"`
    PhoneVerified bool      `json:"phone_verified" db:"phone_verified"`
    
    // Security
    PasswordHash  string    `json:"-" db:"password_hash"`
    TwoFactorEnabled bool   `json:"two_factor_enabled" db:"two_factor_enabled"`
    TwoFactorSecret string  `json:"-" db:"two_factor_secret"`
    
    // Preferences
    Preferences UserPreferences `json:"preferences" db:"preferences"`
    
    // Metadata
    Metadata map[string]interface{} `json:"metadata" db:"metadata"`
}

type UserRole string

const (
    RoleRider  UserRole = "rider"
    RoleDriver UserRole = "driver"
    RoleAdmin  UserRole = "admin"
)

type UserStatus string

const (
    StatusActive    UserStatus = "active"
    StatusInactive  UserStatus = "inactive"
    StatusSuspended UserStatus = "suspended"
    StatusBanned    UserStatus = "banned"
)
```

### **2. Advanced Authentication System**
```go
type AuthenticationService struct {
    jwtSecret       []byte
    tokenExpiry     time.Duration
    refreshExpiry   time.Duration
    sessionStore    SessionStore
    passwordPolicy  PasswordPolicy
    rateLimiter     *RateLimiter
    auditLogger     AuditLogger
}

func (as *AuthenticationService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
    // 1. Validate input
    if err := as.validateRegistrationInput(req); err != nil {
        return nil, fmt.Errorf("validation failed: %v", err)
    }
    
    // 2. Check if user already exists
    existingUser, _ := as.userRepo.GetUserByEmail(ctx, req.Email)
    if existingUser != nil {
        return nil, fmt.Errorf("user already exists")
    }
    
    // 3. Hash password
    passwordHash, err := as.cryptoService.HashPassword(req.Password)
    if err != nil {
        return nil, fmt.Errorf("password hashing failed: %v", err)
    }
    
    // 4. Create user
    user := &User{
        ID:           generateUUID(),
        Email:        req.Email,
        Phone:        req.Phone,
        FirstName:    req.FirstName,
        LastName:     req.LastName,
        Role:         req.Role,
        Status:       StatusActive,
        PasswordHash: passwordHash,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
        Preferences:  DefaultUserPreferences(),
    }
    
    if err := as.userRepo.CreateUser(ctx, user); err != nil {
        return nil, fmt.Errorf("user creation failed: %v", err)
    }
    
    // 5. Send verification email
    verificationToken := as.generateVerificationToken(user.ID)
    if err := as.emailService.SendVerificationEmail(user.Email, verificationToken); err != nil {
        log.Printf("Failed to send verification email: %v", err)
        // Don't fail registration if email fails
    }
    
    // 6. Audit log
    as.auditLogger.Log(AuditEvent{
        UserID:    user.ID,
        Action:    "user_registered",
        Timestamp: time.Now(),
        Metadata:  map[string]interface{}{"email": user.Email, "role": user.Role},
    })
    
    return &RegisterResponse{
        UserID: user.ID,
        Message: "Registration successful. Please verify your email.",
    }, nil
}

func (as *AuthenticationService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
    // 1. Rate limiting
    if !as.rateLimiter.Allow(req.Email, "login_attempts") {
        return nil, fmt.Errorf("too many login attempts")
    }
    
    // 2. Get user
    user, err := as.userRepo.GetUserByEmail(ctx, req.Email)
    if err != nil {
        as.auditLogger.Log(AuditEvent{
            Action:    "login_failed_user_not_found",
            Metadata:  map[string]interface{}{"email": req.Email},
            Timestamp: time.Now(),
        })
        return nil, fmt.Errorf("invalid credentials")
    }
    
    // 3. Check user status
    if user.Status != StatusActive {
        return nil, fmt.Errorf("account is %s", user.Status)
    }
    
    // 4. Verify password
    if !as.cryptoService.VerifyPassword(req.Password, user.PasswordHash) {
        as.auditLogger.Log(AuditEvent{
            UserID:    user.ID,
            Action:    "login_failed_invalid_password",
            Timestamp: time.Now(),
        })
        return nil, fmt.Errorf("invalid credentials")
    }
    
    // 5. Check 2FA if enabled
    if user.TwoFactorEnabled {
        if req.TwoFactorCode == "" {
            return &LoginResponse{
                RequiresTwoFactor: true,
                TempToken:        as.generateTempToken(user.ID),
            }, nil
        }
        
        if !as.verifyTwoFactorCode(user.TwoFactorSecret, req.TwoFactorCode) {
            return nil, fmt.Errorf("invalid 2FA code")
        }
    }
    
    // 6. Generate tokens
    accessToken, refreshToken, err := as.generateTokenPair(user)
    if err != nil {
        return nil, fmt.Errorf("token generation failed: %v", err)
    }
    
    // 7. Create session
    session := &Session{
        ID:           generateUUID(),
        UserID:       user.ID,
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        CreatedAt:    time.Now(),
        ExpiresAt:    time.Now().Add(as.refreshExpiry),
        IsActive:     true,
        IPAddress:    getClientIP(ctx),
        UserAgent:    getUserAgent(ctx),
    }
    
    if err := as.sessionStore.CreateSession(ctx, session); err != nil {
        return nil, fmt.Errorf("session creation failed: %v", err)
    }
    
    // 8. Update last login
    user.LastLogin = &time.Time{}
    *user.LastLogin = time.Now()
    as.userRepo.UpdateUser(ctx, user)
    
    // 9. Audit log
    as.auditLogger.Log(AuditEvent{
        UserID:    user.ID,
        Action:    "login_successful",
        Timestamp: time.Now(),
        Metadata:  map[string]interface{}{"ip": session.IPAddress},
    })
    
    return &LoginResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        User:         user,
        ExpiresIn:    int(as.tokenExpiry.Seconds()),
    }, nil
}
```

### **3. Driver Management & Verification**
```go
type Driver struct {
    UserID          string    `json:"user_id" db:"user_id"`
    LicenseNumber   string    `json:"license_number" db:"license_number"`
    LicenseExpiry   time.Time `json:"license_expiry" db:"license_expiry"`
    LicenseClass    string    `json:"license_class" db:"license_class"`
    
    // Vehicle association
    PrimaryVehicleID *string  `json:"primary_vehicle_id" db:"primary_vehicle_id"`
    
    // Verification status
    BackgroundCheckStatus VerificationStatus `json:"background_check_status" db:"background_check_status"`
    DriverLicenseStatus   VerificationStatus `json:"driver_license_status" db:"driver_license_status"`
    InsuranceStatus       VerificationStatus `json:"insurance_status" db:"insurance_status"`
    
    // Performance metrics
    Rating              float64   `json:"rating" db:"rating"`
    TotalTrips          int       `json:"total_trips" db:"total_trips"`
    CompletionRate      float64   `json:"completion_rate" db:"completion_rate"`
    CancellationRate    float64   `json:"cancellation_rate" db:"cancellation_rate"`
    AverageRating       float64   `json:"average_rating" db:"average_rating"`
    
    // Current status
    IsOnline            bool      `json:"is_online" db:"is_online"`
    CurrentLocation     *Location `json:"current_location" db:"current_location"`
    LastLocationUpdate  time.Time `json:"last_location_update" db:"last_location_update"`
    
    // Financial
    TotalEarnings       float64   `json:"total_earnings" db:"total_earnings"`
    WeeklyEarnings      float64   `json:"weekly_earnings" db:"weekly_earnings"`
    
    // Documents
    Documents           []DriverDocument `json:"documents" db:"documents"`
    
    // Timestamps
    CreatedAt          time.Time `json:"created_at" db:"created_at"`
    UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
    VerifiedAt         *time.Time `json:"verified_at" db:"verified_at"`
}

type VerificationStatus string

const (
    VerificationPending   VerificationStatus = "pending"
    VerificationApproved  VerificationStatus = "approved"
    VerificationRejected  VerificationStatus = "rejected"
    VerificationExpired   VerificationStatus = "expired"
)

type DriverDocument struct {
    ID          string    `json:"id"`
    DriverID    string    `json:"driver_id"`
    Type        DocumentType `json:"type"`
    URL         string    `json:"url"`
    Status      VerificationStatus `json:"status"`
    UploadedAt  time.Time `json:"uploaded_at"`
    VerifiedAt  *time.Time `json:"verified_at"`
    VerifiedBy  *string   `json:"verified_by"`
    RejectionReason *string `json:"rejection_reason"`
}

type DocumentType string

const (
    DocumentDriverLicense     DocumentType = "driver_license"
    DocumentInsurance         DocumentType = "insurance"
    DocumentVehicleRegistration DocumentType = "vehicle_registration"
    DocumentBackgroundCheck   DocumentType = "background_check"
    DocumentProfilePhoto      DocumentType = "profile_photo"
)

func (us *UserService) BecomeDriver(ctx context.Context, req *BecomeDriverRequest) (*BecomeDriverResponse, error) {
    // 1. Get user
    user, err := us.userRepo.GetUserByID(ctx, req.UserID)
    if err != nil {
        return nil, fmt.Errorf("user not found: %v", err)
    }
    
    // 2. Check if already a driver
    existingDriver, _ := us.driverRepo.GetDriverByUserID(ctx, req.UserID)
    if existingDriver != nil {
        return nil, fmt.Errorf("user is already a driver")
    }
    
    // 3. Validate driver requirements
    if err := us.validateDriverRequirements(req); err != nil {
        return nil, fmt.Errorf("driver requirements not met: %v", err)
    }
    
    // 4. Create driver profile
    driver := &Driver{
        UserID:                req.UserID,
        LicenseNumber:         req.LicenseNumber,
        LicenseExpiry:         req.LicenseExpiry,
        LicenseClass:          req.LicenseClass,
        BackgroundCheckStatus: VerificationPending,
        DriverLicenseStatus:   VerificationPending,
        InsuranceStatus:       VerificationPending,
        Rating:                5.0, // Start with perfect rating
        CreatedAt:            time.Now(),
        UpdatedAt:            time.Now(),
    }
    
    if err := us.driverRepo.CreateDriver(ctx, driver); err != nil {
        return nil, fmt.Errorf("driver creation failed: %v", err)
    }
    
    // 5. Update user role
    user.Role = RoleDriver
    user.UpdatedAt = time.Now()
    if err := us.userRepo.UpdateUser(ctx, user); err != nil {
        return nil, fmt.Errorf("user role update failed: %v", err)
    }
    
    // 6. Initiate verification process
    go us.initiateDriverVerification(driver.UserID)
    
    // 7. Audit log
    us.auditLogger.Log(AuditEvent{
        UserID:    user.ID,
        Action:    "became_driver",
        Timestamp: time.Now(),
    })
    
    return &BecomeDriverResponse{
        DriverID: driver.UserID,
        Message:  "Driver application submitted. Verification process initiated.",
        Status:   "pending_verification",
    }, nil
}

func (us *UserService) VerifyDriverDocument(ctx context.Context, req *VerifyDocumentRequest) error {
    // 1. Get document
    document, err := us.documentRepo.GetDocument(ctx, req.DocumentID)
    if err != nil {
        return fmt.Errorf("document not found: %v", err)
    }
    
    // 2. Verify admin permissions
    admin, err := us.userRepo.GetUserByID(ctx, req.VerifierID)
    if err != nil || admin.Role != RoleAdmin {
        return fmt.Errorf("unauthorized: admin access required")
    }
    
    // 3. Update document status
    document.Status = req.Status
    document.VerifiedAt = &time.Time{}
    *document.VerifiedAt = time.Now()
    document.VerifiedBy = &req.VerifierID
    
    if req.Status == VerificationRejected {
        document.RejectionReason = &req.RejectionReason
    }
    
    if err := us.documentRepo.UpdateDocument(ctx, document); err != nil {
        return fmt.Errorf("document update failed: %v", err)
    }
    
    // 4. Check if all driver documents are verified
    driver, err := us.driverRepo.GetDriverByUserID(ctx, document.DriverID)
    if err != nil {
        return fmt.Errorf("driver not found: %v", err)
    }
    
    // Update specific verification status
    switch document.Type {
    case DocumentDriverLicense:
        driver.DriverLicenseStatus = req.Status
    case DocumentInsurance:
        driver.InsuranceStatus = req.Status
    case DocumentBackgroundCheck:
        driver.BackgroundCheckStatus = req.Status
    }
    
    // 5. Check if driver is fully verified
    if us.isDriverFullyVerified(driver) {
        driver.VerifiedAt = &time.Time{}
        *driver.VerifiedAt = time.Now()
        
        // Send notification
        user, _ := us.userRepo.GetUserByID(ctx, driver.UserID)
        us.emailService.SendDriverApprovalEmail(user.Email, user.FirstName)
    }
    
    driver.UpdatedAt = time.Now()
    if err := us.driverRepo.UpdateDriver(ctx, driver); err != nil {
        return fmt.Errorf("driver update failed: %v", err)
    }
    
    // 6. Audit log
    us.auditLogger.Log(AuditEvent{
        UserID:    req.VerifierID,
        Action:    "document_verified",
        Timestamp: time.Now(),
        Metadata: map[string]interface{}{
            "document_id":   req.DocumentID,
            "driver_id":     document.DriverID,
            "document_type": document.Type,
            "status":        req.Status,
        },
    })
    
    return nil
}

func (us *UserService) isDriverFullyVerified(driver *Driver) bool {
    return driver.DriverLicenseStatus == VerificationApproved &&
           driver.InsuranceStatus == VerificationApproved &&
           driver.BackgroundCheckStatus == VerificationApproved
}
```

### **4. Profile Management & Preferences**
```go
type UserPreferences struct {
    // Notification preferences
    EmailNotifications    bool `json:"email_notifications"`
    SMSNotifications      bool `json:"sms_notifications"`
    PushNotifications     bool `json:"push_notifications"`
    
    // Privacy settings
    ShareLocationHistory  bool `json:"share_location_history"`
    AllowDataAnalytics    bool `json:"allow_data_analytics"`
    MarketingOptIn        bool `json:"marketing_opt_in"`
    
    // Ride preferences (for riders)
    PreferredVehicleType  string `json:"preferred_vehicle_type"`
    MaxWaitTime          int    `json:"max_wait_time"`
    AutoAcceptMatches    bool   `json:"auto_accept_matches"`
    
    // Driver preferences (for drivers)
    AcceptSharedRides    bool   `json:"accept_shared_rides"`
    MaxTripDistance      int    `json:"max_trip_distance"`
    PreferredAreas       []string `json:"preferred_areas"`
    
    // Payment preferences
    DefaultPaymentMethod string `json:"default_payment_method"`
    AutoTipping          bool   `json:"auto_tipping"`
    DefaultTipPercentage int    `json:"default_tip_percentage"`
    
    // Accessibility
    RequireAccessibleVehicle bool   `json:"require_accessible_vehicle"`
    LanguagePreference      string `json:"language_preference"`
    
    // Safety preferences
    ShareTripDetails        bool `json:"share_trip_details"`
    EmergencyContacts       []string `json:"emergency_contacts"`
    RequireDriverPhoto      bool `json:"require_driver_photo"`
}

func (us *UserService) UpdateUserProfile(ctx context.Context, req *UpdateProfileRequest) (*User, error) {
    // 1. Get current user
    user, err := us.userRepo.GetUserByID(ctx, req.UserID)
    if err != nil {
        return nil, fmt.Errorf("user not found: %v", err)
    }
    
    // 2. Validate changes
    if err := us.validateProfileUpdate(req); err != nil {
        return nil, fmt.Errorf("validation failed: %v", err)
    }
    
    // 3. Track what changed for audit
    changes := us.trackProfileChanges(user, req)
    
    // 4. Apply updates
    if req.FirstName != nil {
        user.FirstName = *req.FirstName
    }
    if req.LastName != nil {
        user.LastName = *req.LastName
    }
    if req.Phone != nil && *req.Phone != user.Phone {
        user.Phone = *req.Phone
        user.PhoneVerified = false // Require re-verification
        
        // Send SMS verification
        verificationCode := us.generateSMSVerificationCode()
        us.smsService.SendVerificationSMS(*req.Phone, verificationCode)
        us.storeVerificationCode(req.UserID, "phone", verificationCode)
    }
    if req.ProfilePicture != nil {
        user.ProfilePicture = *req.ProfilePicture
    }
    if req.Preferences != nil {
        user.Preferences = *req.Preferences
    }
    
    user.UpdatedAt = time.Now()
    
    // 5. Save changes
    if err := us.userRepo.UpdateUser(ctx, user); err != nil {
        return nil, fmt.Errorf("profile update failed: %v", err)
    }
    
    // 6. Audit log
    us.auditLogger.Log(AuditEvent{
        UserID:    user.ID,
        Action:    "profile_updated",
        Timestamp: time.Now(),
        Metadata:  map[string]interface{}{"changes": changes},
    })
    
    return user, nil
}
```

### **5. Security & Audit System**
```go
type AuditLogger struct {
    db          *sql.DB
    eventQueue  chan AuditEvent
    workerCount int
}

type AuditEvent struct {
    ID          string                 `json:"id"`
    UserID      string                 `json:"user_id"`
    Action      string                 `json:"action"`
    Resource    string                 `json:"resource"`
    Timestamp   time.Time              `json:"timestamp"`
    IPAddress   string                 `json:"ip_address"`
    UserAgent   string                 `json:"user_agent"`
    Metadata    map[string]interface{} `json:"metadata"`
    Severity    AuditSeverity          `json:"severity"`
}

type AuditSeverity string

const (
    SeverityInfo     AuditSeverity = "info"
    SeverityWarning  AuditSeverity = "warning"
    SeverityError    AuditSeverity = "error"
    SeverityCritical AuditSeverity = "critical"
)

func (al *AuditLogger) Log(event AuditEvent) {
    event.ID = generateUUID()
    if event.Timestamp.IsZero() {
        event.Timestamp = time.Now()
    }
    if event.Severity == "" {
        event.Severity = SeverityInfo
    }
    
    // Queue event for async processing
    select {
    case al.eventQueue <- event:
    default:
        // Queue full, log synchronously
        al.logEventSync(event)
    }
}

func (al *AuditLogger) Start() {
    for i := 0; i < al.workerCount; i++ {
        go al.worker()
    }
}

func (al *AuditLogger) worker() {
    for event := range al.eventQueue {
        al.logEventSync(event)
    }
}

func (al *AuditLogger) logEventSync(event AuditEvent) {
    query := `
        INSERT INTO audit_events (
            id, user_id, action, resource, timestamp, 
            ip_address, user_agent, metadata, severity
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `
    
    metadataJSON, _ := json.Marshal(event.Metadata)
    
    _, err := al.db.Exec(query,
        event.ID, event.UserID, event.Action, event.Resource,
        event.Timestamp, event.IPAddress, event.UserAgent,
        metadataJSON, event.Severity,
    )
    
    if err != nil {
        log.Printf("Failed to log audit event: %v", err)
    }
    
    // Send alerts for critical events
    if event.Severity == SeverityCritical {
        al.sendCriticalAlert(event)
    }
}

// Password security
type PasswordPolicy struct {
    MinLength       int
    RequireUpper    bool
    RequireLower    bool
    RequireDigits   bool
    RequireSpecial  bool
    MaxAge          time.Duration
    PreventReuse    int
}

func (pp *PasswordPolicy) ValidatePassword(password string, userID string) error {
    if len(password) < pp.MinLength {
        return fmt.Errorf("password must be at least %d characters", pp.MinLength)
    }
    
    if pp.RequireUpper && !regexp.MustCompile(`[A-Z]`).MatchString(password) {
        return fmt.Errorf("password must contain uppercase letters")
    }
    
    if pp.RequireLower && !regexp.MustCompile(`[a-z]`).MatchString(password) {
        return fmt.Errorf("password must contain lowercase letters")
    }
    
    if pp.RequireDigits && !regexp.MustCompile(`[0-9]`).MatchString(password) {
        return fmt.Errorf("password must contain digits")
    }
    
    if pp.RequireSpecial && !regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(password) {
        return fmt.Errorf("password must contain special characters")
    }
    
    // Check against common passwords
    if pp.isCommonPassword(password) {
        return fmt.Errorf("password is too common")
    }
    
    // Check password history
    if pp.PreventReuse > 0 {
        recentPasswords, err := pp.getRecentPasswords(userID, pp.PreventReuse)
        if err == nil {
            for _, oldPassword := range recentPasswords {
                if bcrypt.CompareHashAndPassword([]byte(oldPassword), []byte(password)) == nil {
                    return fmt.Errorf("password was recently used")
                }
            }
        }
    }
    
    return nil
}
```

---

## 🔧 Technical Implementation Details

### **Database Schema**
```sql
-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'rider',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    profile_picture TEXT,
    date_of_birth DATE,
    password_hash VARCHAR(255) NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,
    phone_verified BOOLEAN DEFAULT FALSE,
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    two_factor_secret VARCHAR(255),
    preferences JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_login TIMESTAMP
);

-- Drivers table
CREATE TABLE drivers (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    license_number VARCHAR(50) UNIQUE NOT NULL,
    license_expiry DATE NOT NULL,
    license_class VARCHAR(10) NOT NULL,
    primary_vehicle_id UUID,
    background_check_status VARCHAR(20) DEFAULT 'pending',
    driver_license_status VARCHAR(20) DEFAULT 'pending',
    insurance_status VARCHAR(20) DEFAULT 'pending',
    rating DECIMAL(3,2) DEFAULT 5.00,
    total_trips INTEGER DEFAULT 0,
    completion_rate DECIMAL(5,2) DEFAULT 0.00,
    cancellation_rate DECIMAL(5,2) DEFAULT 0.00,
    average_rating DECIMAL(3,2) DEFAULT 5.00,
    is_online BOOLEAN DEFAULT FALSE,
    current_location POINT,
    last_location_update TIMESTAMP,
    total_earnings DECIMAL(10,2) DEFAULT 0.00,
    weekly_earnings DECIMAL(10,2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    verified_at TIMESTAMP
);

-- Driver documents table
CREATE TABLE driver_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL REFERENCES drivers(user_id),
    type VARCHAR(50) NOT NULL,
    url TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    uploaded_at TIMESTAMP DEFAULT NOW(),
    verified_at TIMESTAMP,
    verified_by UUID REFERENCES users(id),
    rejection_reason TEXT
);

-- Sessions table
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    access_token VARCHAR(512) NOT NULL,
    refresh_token VARCHAR(512) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    ip_address INET,
    user_agent TEXT
);

-- Audit events table
CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    resource VARCHAR(100),
    timestamp TIMESTAMP DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}',
    severity VARCHAR(20) DEFAULT 'info'
);

-- Indexes for performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_drivers_location ON drivers USING GIST(current_location);
CREATE INDEX idx_drivers_online ON drivers(is_online);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_active ON sessions(is_active, expires_at);
CREATE INDEX idx_audit_events_user_id ON audit_events(user_id);
CREATE INDEX idx_audit_events_action ON audit_events(action);
CREATE INDEX idx_audit_events_timestamp ON audit_events(timestamp);
```

### **gRPC Service Implementation**
```go
func (us *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    // Input validation
    if req.UserId == "" {
        return nil, status.Errorf(codes.InvalidArgument, "user ID is required")
    }
    
    // Get user from database
    user, err := us.userRepo.GetUserByID(ctx, req.UserId)
    if err != nil {
        if errors.Is(err, ErrUserNotFound) {
            return nil, status.Errorf(codes.NotFound, "user not found")
        }
        return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
    }
    
    // Convert to protobuf
    pbUser := &pb.User{
        Id:              user.ID,
        Email:           user.Email,
        Phone:           user.Phone,
        FirstName:       user.FirstName,
        LastName:        user.LastName,
        Role:            string(user.Role),
        Status:          string(user.Status),
        ProfilePicture:  user.ProfilePicture,
        EmailVerified:   user.EmailVerified,
        PhoneVerified:   user.PhoneVerified,
        CreatedAt:       timestamppb.New(user.CreatedAt),
        UpdatedAt:       timestamppb.New(user.UpdatedAt),
    }
    
    if user.LastLogin != nil {
        pbUser.LastLogin = timestamppb.New(*user.LastLogin)
    }
    
    return &pb.GetUserResponse{User: pbUser}, nil
}

func (us *UserService) UpdateDriverLocation(ctx context.Context, req *pb.UpdateDriverLocationRequest) (*pb.UpdateDriverLocationResponse, error) {
    // Validate input
    if req.DriverId == "" {
        return nil, status.Errorf(codes.InvalidArgument, "driver ID is required")
    }
    if req.Location == nil {
        return nil, status.Errorf(codes.InvalidArgument, "location is required")
    }
    
    // Get driver
    driver, err := us.driverRepo.GetDriverByUserID(ctx, req.DriverId)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "driver not found")
    }
    
    // Update location
    driver.CurrentLocation = &Location{
        Latitude:  req.Location.Latitude,
        Longitude: req.Location.Longitude,
    }
    driver.LastLocationUpdate = time.Now()
    driver.UpdatedAt = time.Now()
    
    if err := us.driverRepo.UpdateDriver(ctx, driver); err != nil {
        return nil, status.Errorf(codes.Internal, "failed to update location: %v", err)
    }
    
    return &pb.UpdateDriverLocationResponse{Success: true}, nil
}
```

---

## 📊 Performance Optimization

### **Caching Strategy**
```go
type UserCache struct {
    redis  *redis.Client
    prefix string
    ttl    time.Duration
}

func (uc *UserCache) GetUser(userID string) (*User, error) {
    key := fmt.Sprintf("%s:user:%s", uc.prefix, userID)
    
    data, err := uc.redis.Get(context.Background(), key).Result()
    if err != nil {
        if err == redis.Nil {
            return nil, ErrCacheMiss
        }
        return nil, err
    }
    
    var user User
    if err := json.Unmarshal([]byte(data), &user); err != nil {
        return nil, err
    }
    
    return &user, nil
}

func (uc *UserCache) SetUser(user *User) error {
    key := fmt.Sprintf("%s:user:%s", uc.prefix, user.ID)
    
    data, err := json.Marshal(user)
    if err != nil {
        return err
    }
    
    return uc.redis.Set(context.Background(), key, data, uc.ttl).Err()
}

func (uc *UserCache) InvalidateUser(userID string) error {
    key := fmt.Sprintf("%s:user:%s", uc.prefix, userID)
    return uc.redis.Del(context.Background(), key).Err()
}
```

The User Service is the cornerstone of the rideshare platform, providing secure authentication, comprehensive user management, and robust driver verification systems. It ensures data security, maintains audit trails, and provides the foundation for all user interactions across the platform.
