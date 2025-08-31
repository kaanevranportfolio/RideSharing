# 🚙 VEHICLE SERVICE - DEEP DIVE

## 📋 Overview
The **Vehicle Service** manages all vehicle-related operations in the rideshare platform, including vehicle registration, verification, availability tracking, and maintenance scheduling. It ensures that only properly registered and verified vehicles are used for rides, maintaining safety and regulatory compliance.

---

## 🎯 Core Responsibilities

### **1. Vehicle Registration & Management**
```go
type VehicleService struct {
    vehicleRepo     VehicleRepository
    driverRepo      DriverRepository
    documentService DocumentService
    insuranceService InsuranceService
    auditLogger     AuditLogger
    metrics         *VehicleMetrics
}

type Vehicle struct {
    ID              string        `json:"id" db:"id"`
    OwnerID         string        `json:"owner_id" db:"owner_id"` // Driver's user ID
    
    // Basic vehicle information
    Make            string        `json:"make" db:"make"`
    Model           string        `json:"model" db:"model"`
    Year            int           `json:"year" db:"year"`
    Color           string        `json:"color" db:"color"`
    LicensePlate    string        `json:"license_plate" db:"license_plate"`
    VIN             string        `json:"vin" db:"vin"`
    
    // Vehicle classification
    VehicleType     VehicleType   `json:"vehicle_type" db:"vehicle_type"`
    SeatingCapacity int           `json:"seating_capacity" db:"seating_capacity"`
    Category        VehicleCategory `json:"category" db:"category"`
    
    // Status and availability
    Status          VehicleStatus `json:"status" db:"status"`
    IsAvailable     bool          `json:"is_available" db:"is_available"`
    CurrentLocation *Location     `json:"current_location" db:"current_location"`
    
    // Verification and compliance
    RegistrationStatus     VerificationStatus `json:"registration_status" db:"registration_status"`
    InsuranceStatus        VerificationStatus `json:"insurance_status" db:"insurance_status"`
    InspectionStatus       VerificationStatus `json:"inspection_status" db:"inspection_status"`
    
    // Insurance information
    InsuranceProvider      string        `json:"insurance_provider" db:"insurance_provider"`
    InsurancePolicyNumber  string        `json:"insurance_policy_number" db:"insurance_policy_number"`
    InsuranceExpiry        time.Time     `json:"insurance_expiry" db:"insurance_expiry"`
    
    // Registration information
    RegistrationNumber     string        `json:"registration_number" db:"registration_number"`
    RegistrationExpiry     time.Time     `json:"registration_expiry" db:"registration_expiry"`
    RegistrationState      string        `json:"registration_state" db:"registration_state"`
    
    // Technical specifications
    FuelType               FuelType      `json:"fuel_type" db:"fuel_type"`
    TransmissionType       string        `json:"transmission_type" db:"transmission_type"`
    Mileage                int           `json:"mileage" db:"mileage"`
    
    // Features and amenities
    Features               []VehicleFeature `json:"features" db:"features"`
    AccessibilityFeatures  []string      `json:"accessibility_features" db:"accessibility_features"`
    
    // Performance tracking
    TotalTrips             int           `json:"total_trips" db:"total_trips"`
    TotalDistance          float64       `json:"total_distance" db:"total_distance"`
    AverageRating          float64       `json:"average_rating" db:"average_rating"`
    
    // Maintenance
    LastMaintenanceDate    *time.Time    `json:"last_maintenance_date" db:"last_maintenance_date"`
    NextMaintenanceDate    *time.Time    `json:"next_maintenance_date" db:"next_maintenance_date"`
    MaintenanceAlerts      []MaintenanceAlert `json:"maintenance_alerts" db:"maintenance_alerts"`
    
    // Financial
    PurchasePrice          float64       `json:"purchase_price" db:"purchase_price"`
    CurrentValue           float64       `json:"current_value" db:"current_value"`
    MonthlyPayment         float64       `json:"monthly_payment" db:"monthly_payment"`
    
    // Documents
    Documents              []VehicleDocument `json:"documents" db:"documents"`
    Photos                 []VehiclePhoto    `json:"photos" db:"photos"`
    
    // Timestamps
    CreatedAt              time.Time     `json:"created_at" db:"created_at"`
    UpdatedAt              time.Time     `json:"updated_at" db:"updated_at"`
    VerifiedAt             *time.Time    `json:"verified_at" db:"verified_at"`
}

type VehicleType string

const (
    VehicleTypeEconomy    VehicleType = "economy"     // Standard cars
    VehicleTypeComfort    VehicleType = "comfort"     // Mid-size sedans
    VehicleTypePremium    VehicleType = "premium"     // Luxury vehicles
    VehicleTypeSUV        VehicleType = "suv"         // Sport utility vehicles
    VehicleTypeShared     VehicleType = "shared"      // Pool/shared rides
    VehicleTypeAccessible VehicleType = "accessible"  // Wheelchair accessible
    VehicleTypeElectric   VehicleType = "electric"    // Electric vehicles
)

type VehicleCategory string

const (
    CategoryPersonal      VehicleCategory = "personal"       // Personal vehicle
    CategoryRental        VehicleCategory = "rental"         // Rental vehicle
    CategoryFleet         VehicleCategory = "fleet"          // Company fleet
    CategoryLeased        VehicleCategory = "leased"         // Leased vehicle
)

type VehicleStatus string

const (
    StatusActive          VehicleStatus = "active"           // Available for rides
    StatusInactive        VehicleStatus = "inactive"         // Not available
    StatusMaintenance     VehicleStatus = "maintenance"      // Under maintenance
    StatusSuspended       VehicleStatus = "suspended"        // Suspended by admin
    StatusRetired         VehicleStatus = "retired"          // No longer in service
)

type FuelType string

const (
    FuelTypeGasoline     FuelType = "gasoline"
    FuelTypeDiesel       FuelType = "diesel"
    FuelTypeElectric     FuelType = "electric"
    FuelTypeHybrid       FuelType = "hybrid"
    FuelTypePlug InHybrid FuelType = "plug_in_hybrid"
    FuelTypeHydrogen     FuelType = "hydrogen"
)

type VehicleFeature string

const (
    FeatureAirConditioning VehicleFeature = "air_conditioning"
    FeatureHeating         VehicleFeature = "heating"
    FeatureWiFi            VehicleFeature = "wifi"
    FeatureCharger         VehicleFeature = "charger"
    FeatureAuxInput        VehicleFeature = "aux_input"
    FeatureBluetooth       VehicleFeature = "bluetooth"
    FeatureChildSeat       VehicleFeature = "child_seat"
    FeatureLuggage         VehicleFeature = "luggage_space"
    FeaturePetFriendly     VehicleFeature = "pet_friendly"
    FeatureQuietRide       VehicleFeature = "quiet_ride"
)
```

### **2. Vehicle Registration Process**
```go
func (vs *VehicleService) RegisterVehicle(ctx context.Context, req *RegisterVehicleRequest) (*RegisterVehicleResponse, error) {
    // 1. Validate input
    if err := vs.validateVehicleRegistration(req); err != nil {
        return nil, fmt.Errorf("validation failed: %v", err)
    }
    
    // 2. Check if driver exists and is verified
    driver, err := vs.driverRepo.GetDriverByUserID(ctx, req.OwnerID)
    if err != nil {
        return nil, fmt.Errorf("driver not found: %v", err)
    }
    
    if !vs.isDriverVerified(driver) {
        return nil, fmt.Errorf("driver must be verified before registering vehicles")
    }
    
    // 3. Check for duplicate vehicles
    existingVehicle, _ := vs.vehicleRepo.GetVehicleByVIN(ctx, req.VIN)
    if existingVehicle != nil {
        return nil, fmt.Errorf("vehicle with VIN %s already registered", req.VIN)
    }
    
    existingPlate, _ := vs.vehicleRepo.GetVehicleByLicensePlate(ctx, req.LicensePlate)
    if existingPlate != nil {
        return nil, fmt.Errorf("vehicle with license plate %s already registered", req.LicensePlate)
    }
    
    // 4. Determine vehicle category and features
    vehicleType := vs.determineVehicleType(req.Make, req.Model, req.Year)
    seatingCapacity := vs.getSeatingCapacity(req.Make, req.Model, req.Year)
    
    // 5. Create vehicle record
    vehicle := &Vehicle{
        ID:                    generateUUID(),
        OwnerID:               req.OwnerID,
        Make:                  req.Make,
        Model:                 req.Model,
        Year:                  req.Year,
        Color:                 req.Color,
        LicensePlate:          req.LicensePlate,
        VIN:                   req.VIN,
        VehicleType:           vehicleType,
        SeatingCapacity:       seatingCapacity,
        Category:              req.Category,
        Status:                StatusInactive, // Start inactive until verified
        IsAvailable:           false,
        RegistrationStatus:    VerificationPending,
        InsuranceStatus:       VerificationPending,
        InspectionStatus:      VerificationPending,
        InsuranceProvider:     req.InsuranceProvider,
        InsurancePolicyNumber: req.InsurancePolicyNumber,
        InsuranceExpiry:       req.InsuranceExpiry,
        RegistrationNumber:    req.RegistrationNumber,
        RegistrationExpiry:    req.RegistrationExpiry,
        RegistrationState:     req.RegistrationState,
        FuelType:              req.FuelType,
        TransmissionType:      req.TransmissionType,
        Mileage:               req.Mileage,
        Features:              req.Features,
        PurchasePrice:         req.PurchasePrice,
        MonthlyPayment:        req.MonthlyPayment,
        CreatedAt:             time.Now(),
        UpdatedAt:             time.Now(),
    }
    
    // 6. Save vehicle
    if err := vs.vehicleRepo.CreateVehicle(ctx, vehicle); err != nil {
        return nil, fmt.Errorf("vehicle creation failed: %v", err)
    }
    
    // 7. Create required documents
    requiredDocs := []VehicleDocumentType{
        DocumentTypeRegistration,
        DocumentTypeInsurance,
        DocumentTypeInspection,
        DocumentTypeVehiclePhotos,
    }
    
    for _, docType := range requiredDocs {
        doc := &VehicleDocument{
            ID:         generateUUID(),
            VehicleID:  vehicle.ID,
            Type:       docType,
            Status:     VerificationPending,
            CreatedAt:  time.Now(),
        }
        vs.documentService.CreateDocument(ctx, doc)
    }
    
    // 8. Schedule verification
    go vs.scheduleVehicleVerification(vehicle.ID)
    
    // 9. Audit log
    vs.auditLogger.Log(AuditEvent{
        UserID:    req.OwnerID,
        Action:    "vehicle_registered",
        Timestamp: time.Now(),
        Metadata: map[string]interface{}{
            "vehicle_id":     vehicle.ID,
            "license_plate":  vehicle.LicensePlate,
            "vin":           vehicle.VIN,
        },
    })
    
    return &RegisterVehicleResponse{
        VehicleID: vehicle.ID,
        Status:    "registered",
        Message:   "Vehicle registered successfully. Please upload required documents for verification.",
    }, nil
}

func (vs *VehicleService) determineVehicleType(make, model string, year int) VehicleType {
    // Vehicle classification logic based on make/model/year
    luxuryBrands := map[string]bool{
        "mercedes": true, "bmw": true, "audi": true, "lexus": true,
        "cadillac": true, "lincoln": true, "volvo": true, "acura": true,
        "infiniti": true, "porsche": true, "maserati": true, "jaguar": true,
    }
    
    suvModels := map[string]bool{
        "suv": true, "crossover": true, "x5": true, "q7": true,
        "escalade": true, "navigator": true, "suburban": true,
        "tahoe": true, "expedition": true, "pilot": true,
    }
    
    economyBrands := map[string]bool{
        "toyota": true, "honda": true, "nissan": true, "ford": true,
        "chevrolet": true, "hyundai": true, "kia": true, "mazda": true,
    }
    
    makeLower := strings.ToLower(make)
    modelLower := strings.ToLower(model)
    
    // Check for luxury brands
    if luxuryBrands[makeLower] {
        return VehicleTypePremium
    }
    
    // Check for SUV models
    if suvModels[modelLower] || strings.Contains(modelLower, "suv") {
        return VehicleTypeSUV
    }
    
    // Electric vehicles
    if strings.Contains(modelLower, "electric") || 
       strings.Contains(modelLower, "ev") ||
       makeLower == "tesla" {
        return VehicleTypeElectric
    }
    
    // Economy vehicles
    if economyBrands[makeLower] && year >= 2015 {
        return VehicleTypeEconomy
    }
    
    // Default to comfort
    return VehicleTypeComfort
}
```

### **3. Vehicle Verification System**
```go
type VehicleDocument struct {
    ID              string                `json:"id" db:"id"`
    VehicleID       string                `json:"vehicle_id" db:"vehicle_id"`
    Type            VehicleDocumentType   `json:"type" db:"type"`
    URL             string                `json:"url" db:"url"`
    Status          VerificationStatus    `json:"status" db:"status"`
    UploadedAt      time.Time            `json:"uploaded_at" db:"uploaded_at"`
    VerifiedAt      *time.Time           `json:"verified_at" db:"verified_at"`
    VerifiedBy      *string              `json:"verified_by" db:"verified_by"`
    RejectionReason *string              `json:"rejection_reason" db:"rejection_reason"`
    ExpiryDate      *time.Time           `json:"expiry_date" db:"expiry_date"`
    Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
}

type VehicleDocumentType string

const (
    DocumentTypeRegistration    VehicleDocumentType = "registration"
    DocumentTypeInsurance       VehicleDocumentType = "insurance"
    DocumentTypeInspection      VehicleDocumentType = "inspection"
    DocumentTypeVehiclePhotos   VehicleDocumentType = "vehicle_photos"
    DocumentTypeTitle           VehicleDocumentType = "title"
    DocumentTypeLease           VehicleDocumentType = "lease"
)

func (vs *VehicleService) VerifyVehicleDocument(ctx context.Context, req *VerifyVehicleDocumentRequest) error {
    // 1. Get document
    document, err := vs.documentService.GetDocument(ctx, req.DocumentID)
    if err != nil {
        return fmt.Errorf("document not found: %v", err)
    }
    
    // 2. Verify admin permissions
    admin, err := vs.userService.GetUser(ctx, req.VerifierID)
    if err != nil || admin.Role != "admin" {
        return fmt.Errorf("unauthorized: admin access required")
    }
    
    // 3. Perform document-specific verification
    switch document.Type {
    case DocumentTypeRegistration:
        if err := vs.verifyRegistrationDocument(ctx, document, req); err != nil {
            return err
        }
    case DocumentTypeInsurance:
        if err := vs.verifyInsuranceDocument(ctx, document, req); err != nil {
            return err
        }
    case DocumentTypeInspection:
        if err := vs.verifyInspectionDocument(ctx, document, req); err != nil {
            return err
        }
    case DocumentTypeVehiclePhotos:
        if err := vs.verifyVehiclePhotos(ctx, document, req); err != nil {
            return err
        }
    }
    
    // 4. Update document status
    document.Status = req.Status
    document.VerifiedAt = &time.Time{}
    *document.VerifiedAt = time.Now()
    document.VerifiedBy = &req.VerifierID
    
    if req.Status == VerificationRejected {
        document.RejectionReason = &req.RejectionReason
    }
    
    if err := vs.documentService.UpdateDocument(ctx, document); err != nil {
        return fmt.Errorf("document update failed: %v", err)
    }
    
    // 5. Update vehicle verification status
    vehicle, err := vs.vehicleRepo.GetVehicleByID(ctx, document.VehicleID)
    if err != nil {
        return fmt.Errorf("vehicle not found: %v", err)
    }
    
    // Update specific verification status
    switch document.Type {
    case DocumentTypeRegistration:
        vehicle.RegistrationStatus = req.Status
    case DocumentTypeInsurance:
        vehicle.InsuranceStatus = req.Status
    case DocumentTypeInspection:
        vehicle.InspectionStatus = req.Status
    }
    
    // 6. Check if vehicle is fully verified
    if vs.isVehicleFullyVerified(vehicle) {
        vehicle.Status = StatusActive
        vehicle.IsAvailable = true
        vehicle.VerifiedAt = &time.Time{}
        *vehicle.VerifiedAt = time.Now()
        
        // Notify driver
        vs.notifyDriverVehicleApproved(vehicle.OwnerID, vehicle.ID)
    }
    
    vehicle.UpdatedAt = time.Now()
    if err := vs.vehicleRepo.UpdateVehicle(ctx, vehicle); err != nil {
        return fmt.Errorf("vehicle update failed: %v", err)
    }
    
    // 7. Audit log
    vs.auditLogger.Log(AuditEvent{
        UserID:    req.VerifierID,
        Action:    "vehicle_document_verified",
        Timestamp: time.Now(),
        Metadata: map[string]interface{}{
            "document_id":   req.DocumentID,
            "vehicle_id":    document.VehicleID,
            "document_type": document.Type,
            "status":        req.Status,
        },
    })
    
    return nil
}

func (vs *VehicleService) verifyRegistrationDocument(ctx context.Context, doc *VehicleDocument, req *VerifyVehicleDocumentRequest) error {
    // Extract information from registration document
    registrationInfo, err := vs.extractRegistrationInfo(doc.URL)
    if err != nil {
        return fmt.Errorf("failed to extract registration info: %v", err)
    }
    
    // Verify against vehicle information
    vehicle, err := vs.vehicleRepo.GetVehicleByID(ctx, doc.VehicleID)
    if err != nil {
        return fmt.Errorf("vehicle not found: %v", err)
    }
    
    // Cross-reference VIN
    if registrationInfo.VIN != vehicle.VIN {
        return fmt.Errorf("VIN mismatch: document VIN %s, vehicle VIN %s", 
            registrationInfo.VIN, vehicle.VIN)
    }
    
    // Check registration expiry
    if registrationInfo.ExpiryDate.Before(time.Now()) {
        return fmt.Errorf("registration expired on %s", registrationInfo.ExpiryDate.Format("2006-01-02"))
    }
    
    // Update vehicle with extracted information
    vehicle.RegistrationNumber = registrationInfo.RegistrationNumber
    vehicle.RegistrationExpiry = registrationInfo.ExpiryDate
    vehicle.RegistrationState = registrationInfo.State
    
    return vs.vehicleRepo.UpdateVehicle(ctx, vehicle)
}

func (vs *VehicleService) verifyInsuranceDocument(ctx context.Context, doc *VehicleDocument, req *VerifyVehicleDocumentRequest) error {
    // Extract insurance information
    insuranceInfo, err := vs.extractInsuranceInfo(doc.URL)
    if err != nil {
        return fmt.Errorf("failed to extract insurance info: %v", err)
    }
    
    // Verify with insurance provider
    isValid, err := vs.insuranceService.VerifyPolicy(ctx, &VerifyPolicyRequest{
        PolicyNumber: insuranceInfo.PolicyNumber,
        VIN:         insuranceInfo.VIN,
        Provider:    insuranceInfo.Provider,
    })
    
    if err != nil {
        return fmt.Errorf("insurance verification failed: %v", err)
    }
    
    if !isValid {
        return fmt.Errorf("insurance policy could not be verified")
    }
    
    // Check if insurance covers rideshare activities
    if !insuranceInfo.CoversCommercialUse {
        return fmt.Errorf("insurance does not cover commercial/rideshare use")
    }
    
    // Update vehicle with insurance information
    vehicle, err := vs.vehicleRepo.GetVehicleByID(ctx, doc.VehicleID)
    if err != nil {
        return fmt.Errorf("vehicle not found: %v", err)
    }
    
    vehicle.InsuranceProvider = insuranceInfo.Provider
    vehicle.InsurancePolicyNumber = insuranceInfo.PolicyNumber
    vehicle.InsuranceExpiry = insuranceInfo.ExpiryDate
    
    return vs.vehicleRepo.UpdateVehicle(ctx, vehicle)
}
```

### **4. Vehicle Availability & Location Tracking**
```go
type VehicleAvailabilityService struct {
    vehicleRepo     VehicleRepository
    locationService LocationService
    tripService     TripService
    cache          *RedisCache
    eventPublisher EventPublisher
}

func (vas *VehicleAvailabilityService) UpdateVehicleAvailability(ctx context.Context, req *UpdateAvailabilityRequest) error {
    // 1. Get vehicle
    vehicle, err := vas.vehicleRepo.GetVehicleByID(ctx, req.VehicleID)
    if err != nil {
        return fmt.Errorf("vehicle not found: %v", err)
    }
    
    // 2. Verify ownership
    if vehicle.OwnerID != req.DriverID {
        return fmt.Errorf("unauthorized: driver does not own this vehicle")
    }
    
    // 3. Check if vehicle can be made available
    if req.IsAvailable && !vs.canVehicleBeAvailable(vehicle) {
        return fmt.Errorf("vehicle cannot be made available: %s", vs.getAvailabilityBlockReason(vehicle))
    }
    
    // 4. Update availability
    previousAvailability := vehicle.IsAvailable
    vehicle.IsAvailable = req.IsAvailable
    vehicle.UpdatedAt = time.Now()
    
    // 5. Update location if provided
    if req.Location != nil {
        vehicle.CurrentLocation = req.Location
    }
    
    // 6. Save changes
    if err := vas.vehicleRepo.UpdateVehicle(ctx, vehicle); err != nil {
        return fmt.Errorf("vehicle update failed: %v", err)
    }
    
    // 7. Update cache
    vas.cache.SetVehicleAvailability(vehicle.ID, vehicle.IsAvailable)
    
    // 8. Publish availability change event
    if previousAvailability != vehicle.IsAvailable {
        event := &VehicleAvailabilityChangedEvent{
            VehicleID:   vehicle.ID,
            DriverID:    vehicle.OwnerID,
            IsAvailable: vehicle.IsAvailable,
            Location:    vehicle.CurrentLocation,
            Timestamp:   time.Now(),
        }
        vas.eventPublisher.Publish("vehicle.availability.changed", event)
    }
    
    return nil
}

func (vas *VehicleAvailabilityService) canVehicleBeAvailable(vehicle *Vehicle) bool {
    // Check vehicle status
    if vehicle.Status != StatusActive {
        return false
    }
    
    // Check verification status
    if !vas.isVehicleFullyVerified(vehicle) {
        return false
    }
    
    // Check document expiry
    if vas.hasExpiredDocuments(vehicle) {
        return false
    }
    
    // Check if maintenance is due
    if vas.isMaintenanceDue(vehicle) {
        return false
    }
    
    // Check if driver is currently on a trip
    if vas.isDriverOnTrip(vehicle.OwnerID) {
        return false
    }
    
    return true
}

func (vas *VehicleAvailabilityService) GetAvailableVehiclesNearLocation(ctx context.Context, req *GetAvailableVehiclesRequest) ([]*Vehicle, error) {
    // 1. Query database for vehicles within radius
    vehicles, err := vas.vehicleRepo.GetVehiclesWithinRadius(ctx, &VehicleLocationQuery{
        Latitude:     req.Latitude,
        Longitude:    req.Longitude,
        RadiusMeters: req.RadiusMeters,
        VehicleTypes: req.VehicleTypes,
        MinRating:    req.MinRating,
        Features:     req.RequiredFeatures,
    })
    
    if err != nil {
        return nil, fmt.Errorf("database query failed: %v", err)
    }
    
    // 2. Filter available vehicles
    availableVehicles := make([]*Vehicle, 0)
    for _, vehicle := range vehicles {
        if vehicle.IsAvailable && vas.canVehicleBeAvailable(vehicle) {
            availableVehicles = append(availableVehicles, vehicle)
        }
    }
    
    // 3. Sort by distance and rating
    vas.sortVehiclesByPreference(availableVehicles, req.Latitude, req.Longitude)
    
    // 4. Apply limit
    if req.Limit > 0 && len(availableVehicles) > req.Limit {
        availableVehicles = availableVehicles[:req.Limit]
    }
    
    return availableVehicles, nil
}
```

### **5. Maintenance Management**
```go
type MaintenanceService struct {
    vehicleRepo     VehicleRepository
    maintenanceRepo MaintenanceRepository
    scheduler       MaintenanceScheduler
    notificationService NotificationService
    auditLogger     AuditLogger
}

type MaintenanceRecord struct {
    ID              string             `json:"id" db:"id"`
    VehicleID       string             `json:"vehicle_id" db:"vehicle_id"`
    Type            MaintenanceType    `json:"type" db:"type"`
    Description     string             `json:"description" db:"description"`
    Status          MaintenanceStatus  `json:"status" db:"status"`
    
    // Scheduling
    ScheduledDate   time.Time          `json:"scheduled_date" db:"scheduled_date"`
    CompletedDate   *time.Time         `json:"completed_date" db:"completed_date"`
    
    // Service details
    ServiceProvider string             `json:"service_provider" db:"service_provider"`
    Cost            float64            `json:"cost" db:"cost"`
    Mileage         int                `json:"mileage" db:"mileage"`
    
    // Parts and labor
    PartsReplaced   []string           `json:"parts_replaced" db:"parts_replaced"`
    LaborHours      float64            `json:"labor_hours" db:"labor_hours"`
    
    // Documentation
    InvoiceURL      string             `json:"invoice_url" db:"invoice_url"`
    Photos          []string           `json:"photos" db:"photos"`
    Notes           string             `json:"notes" db:"notes"`
    
    // Timestamps
    CreatedAt       time.Time          `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time          `json:"updated_at" db:"updated_at"`
}

type MaintenanceType string

const (
    MaintenanceOilChange       MaintenanceType = "oil_change"
    MaintenanceTireRotation    MaintenanceType = "tire_rotation"
    MaintenanceBrakeInspection MaintenanceType = "brake_inspection"
    MaintenanceTransmission    MaintenanceType = "transmission"
    MaintenanceAirFilter       MaintenanceType = "air_filter"
    MaintenanceInspection      MaintenanceType = "inspection"
    MaintenanceRepair          MaintenanceType = "repair"
    MaintenanceRecall          MaintenanceType = "recall"
)

func (ms *MaintenanceService) ScheduleMaintenance(ctx context.Context, req *ScheduleMaintenanceRequest) (*MaintenanceRecord, error) {
    // 1. Validate vehicle ownership
    vehicle, err := ms.vehicleRepo.GetVehicleByID(ctx, req.VehicleID)
    if err != nil {
        return nil, fmt.Errorf("vehicle not found: %v", err)
    }
    
    if vehicle.OwnerID != req.DriverID {
        return nil, fmt.Errorf("unauthorized: driver does not own this vehicle")
    }
    
    // 2. Create maintenance record
    maintenance := &MaintenanceRecord{
        ID:              generateUUID(),
        VehicleID:       req.VehicleID,
        Type:            req.Type,
        Description:     req.Description,
        Status:          MaintenanceStatusScheduled,
        ScheduledDate:   req.ScheduledDate,
        ServiceProvider: req.ServiceProvider,
        Mileage:         vehicle.Mileage,
        CreatedAt:       time.Now(),
        UpdatedAt:       time.Now(),
    }
    
    // 3. Save maintenance record
    if err := ms.maintenanceRepo.CreateMaintenance(ctx, maintenance); err != nil {
        return nil, fmt.Errorf("maintenance scheduling failed: %v", err)
    }
    
    // 4. Update vehicle status if necessary
    if ms.shouldSuspendVehicle(req.Type, req.ScheduledDate) {
        vehicle.Status = StatusMaintenance
        vehicle.IsAvailable = false
        vehicle.UpdatedAt = time.Now()
        ms.vehicleRepo.UpdateVehicle(ctx, vehicle)
    }
    
    // 5. Schedule reminders
    ms.scheduler.ScheduleMaintenanceReminder(maintenance.ID, req.ScheduledDate)
    
    return maintenance, nil
}

func (ms *MaintenanceService) CheckMaintenanceDue(ctx context.Context, vehicleID string) (*MaintenanceDueCheck, error) {
    vehicle, err := ms.vehicleRepo.GetVehicleByID(ctx, vehicleID)
    if err != nil {
        return nil, fmt.Errorf("vehicle not found: %v", err)
    }
    
    maintenanceCheck := &MaintenanceDueCheck{
        VehicleID: vehicleID,
        Alerts:    make([]MaintenanceAlert, 0),
    }
    
    // Check mileage-based maintenance
    mileageDue := ms.checkMileageBasedMaintenance(vehicle)
    maintenanceCheck.Alerts = append(maintenanceCheck.Alerts, mileageDue...)
    
    // Check time-based maintenance
    timeDue := ms.checkTimeBasedMaintenance(vehicle)
    maintenanceCheck.Alerts = append(maintenanceCheck.Alerts, timeDue...)
    
    // Check inspection due
    inspectionDue := ms.checkInspectionDue(vehicle)
    if inspectionDue != nil {
        maintenanceCheck.Alerts = append(maintenanceCheck.Alerts, *inspectionDue)
    }
    
    // Determine overall status
    maintenanceCheck.Status = ms.determineMaintenanceStatus(maintenanceCheck.Alerts)
    
    return maintenanceCheck, nil
}

type MaintenanceAlert struct {
    Type        MaintenanceType `json:"type"`
    Urgency     AlertUrgency    `json:"urgency"`
    Message     string          `json:"message"`
    DueDate     *time.Time      `json:"due_date"`
    DueMileage  *int           `json:"due_mileage"`
    Description string          `json:"description"`
}

type AlertUrgency string

const (
    UrgencyLow      AlertUrgency = "low"      // Due in 30+ days
    UrgencyMedium   AlertUrgency = "medium"   // Due in 7-30 days
    UrgencyHigh     AlertUrgency = "high"     // Due in 1-7 days
    UrgencyCritical AlertUrgency = "critical" // Overdue or immediate
)
```

---

## 🔧 Technical Implementation Details

### **Database Schema**
```sql
-- Vehicles table
CREATE TABLE vehicles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id),
    make VARCHAR(50) NOT NULL,
    model VARCHAR(50) NOT NULL,
    year INTEGER NOT NULL CHECK (year >= 1990 AND year <= EXTRACT(YEAR FROM CURRENT_DATE) + 1),
    color VARCHAR(30) NOT NULL,
    license_plate VARCHAR(20) UNIQUE NOT NULL,
    vin VARCHAR(17) UNIQUE NOT NULL,
    vehicle_type VARCHAR(20) NOT NULL,
    seating_capacity INTEGER NOT NULL CHECK (seating_capacity >= 1 AND seating_capacity <= 12),
    category VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'inactive',
    is_available BOOLEAN DEFAULT FALSE,
    current_location POINT,
    registration_status VARCHAR(20) DEFAULT 'pending',
    insurance_status VARCHAR(20) DEFAULT 'pending',
    inspection_status VARCHAR(20) DEFAULT 'pending',
    insurance_provider VARCHAR(100),
    insurance_policy_number VARCHAR(50),
    insurance_expiry DATE,
    registration_number VARCHAR(50),
    registration_expiry DATE,
    registration_state VARCHAR(5),
    fuel_type VARCHAR(20),
    transmission_type VARCHAR(20),
    mileage INTEGER DEFAULT 0,
    features JSONB DEFAULT '[]',
    accessibility_features JSONB DEFAULT '[]',
    total_trips INTEGER DEFAULT 0,
    total_distance DECIMAL(10,2) DEFAULT 0.00,
    average_rating DECIMAL(3,2) DEFAULT 5.00,
    last_maintenance_date DATE,
    next_maintenance_date DATE,
    maintenance_alerts JSONB DEFAULT '[]',
    purchase_price DECIMAL(10,2),
    current_value DECIMAL(10,2),
    monthly_payment DECIMAL(8,2),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    verified_at TIMESTAMP
);

-- Vehicle documents table
CREATE TABLE vehicle_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    url TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    uploaded_at TIMESTAMP DEFAULT NOW(),
    verified_at TIMESTAMP,
    verified_by UUID REFERENCES users(id),
    rejection_reason TEXT,
    expiry_date DATE,
    metadata JSONB DEFAULT '{}'
);

-- Vehicle photos table
CREATE TABLE vehicle_photos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    photo_type VARCHAR(20) NOT NULL, -- 'exterior', 'interior', 'dashboard'
    is_primary BOOLEAN DEFAULT FALSE,
    uploaded_at TIMESTAMP DEFAULT NOW()
);

-- Maintenance records table
CREATE TABLE maintenance_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    status VARCHAR(20) DEFAULT 'scheduled',
    scheduled_date DATE NOT NULL,
    completed_date DATE,
    service_provider VARCHAR(100),
    cost DECIMAL(8,2),
    mileage INTEGER,
    parts_replaced JSONB DEFAULT '[]',
    labor_hours DECIMAL(4,2),
    invoice_url TEXT,
    photos JSONB DEFAULT '[]',
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_vehicles_owner_id ON vehicles(owner_id);
CREATE INDEX idx_vehicles_location ON vehicles USING GIST(current_location);
CREATE INDEX idx_vehicles_available ON vehicles(is_available, status);
CREATE INDEX idx_vehicles_type ON vehicles(vehicle_type);
CREATE INDEX idx_vehicles_license_plate ON vehicles(license_plate);
CREATE INDEX idx_vehicles_vin ON vehicles(vin);
CREATE INDEX idx_vehicle_documents_vehicle_id ON vehicle_documents(vehicle_id);
CREATE INDEX idx_vehicle_documents_status ON vehicle_documents(status);
CREATE INDEX idx_maintenance_records_vehicle_id ON maintenance_records(vehicle_id);
CREATE INDEX idx_maintenance_records_scheduled_date ON maintenance_records(scheduled_date);
```

### **gRPC Service Implementation**
```go
func (vs *VehicleService) GetVehicle(ctx context.Context, req *pb.GetVehicleRequest) (*pb.GetVehicleResponse, error) {
    vehicle, err := vs.vehicleRepo.GetVehicleByID(ctx, req.VehicleId)
    if err != nil {
        if errors.Is(err, ErrVehicleNotFound) {
            return nil, status.Errorf(codes.NotFound, "vehicle not found")
        }
        return nil, status.Errorf(codes.Internal, "failed to get vehicle: %v", err)
    }
    
    pbVehicle := &pb.Vehicle{
        Id:              vehicle.ID,
        OwnerId:         vehicle.OwnerID,
        Make:            vehicle.Make,
        Model:           vehicle.Model,
        Year:            int32(vehicle.Year),
        Color:           vehicle.Color,
        LicensePlate:    vehicle.LicensePlate,
        Vin:             vehicle.VIN,
        VehicleType:     string(vehicle.VehicleType),
        SeatingCapacity: int32(vehicle.SeatingCapacity),
        Status:          string(vehicle.Status),
        IsAvailable:     vehicle.IsAvailable,
        AverageRating:   vehicle.AverageRating,
        TotalTrips:      int32(vehicle.TotalTrips),
        CreatedAt:       timestamppb.New(vehicle.CreatedAt),
        UpdatedAt:       timestamppb.New(vehicle.UpdatedAt),
    }
    
    if vehicle.CurrentLocation != nil {
        pbVehicle.CurrentLocation = &pb.Location{
            Latitude:  vehicle.CurrentLocation.Latitude,
            Longitude: vehicle.CurrentLocation.Longitude,
        }
    }
    
    return &pb.GetVehicleResponse{Vehicle: pbVehicle}, nil
}
```

---

## 📊 Performance & Monitoring

### **Vehicle Metrics**
```go
type VehicleMetrics struct {
    TotalVehicles        prometheus.Gauge
    AvailableVehicles    prometheus.Gauge
    VehiclesByType       *prometheus.GaugeVec
    VerificationDuration prometheus.Histogram
    MaintenanceAlerts    prometheus.Counter
}

func (vm *VehicleMetrics) RecordVehicleRegistration(vehicleType string) {
    vm.TotalVehicles.Inc()
    vm.VehiclesByType.WithLabelValues(vehicleType).Inc()
}

func (vm *VehicleMetrics) RecordAvailabilityChange(wasAvailable, isAvailable bool) {
    if !wasAvailable && isAvailable {
        vm.AvailableVehicles.Inc()
    } else if wasAvailable && !isAvailable {
        vm.AvailableVehicles.Dec()
    }
}
```

The Vehicle Service ensures that only properly registered, verified, and maintained vehicles are available for rides, providing a critical safety and quality layer for the rideshare platform.
