package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// MODELS
// ============================================================================

// Product model with enhanced fields
type Product struct {
	ProductID         uint      `gorm:"primaryKey" json:"product_id"`
	SKU               string    `gorm:"unique;not null" json:"sku"`
	Name              string    `gorm:"not null;index" json:"name"`
	Description       string    `json:"description"`
	Category          string    `gorm:"not null" json:"category"`
	Quantity          int       `gorm:"not null" json:"quantity"`
	LowStockThreshold int       `gorm:"default:10" json:"low_stock_threshold"`
	Price             float64   `json:"price"`
	Version           int       `gorm:"default:1" json:"version"` // For optimistic locking
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	CreatedBy         uint      `json:"created_by"`
	UpdatedBy         uint      `json:"updated_by"`
}

// ProductMove model for tracking inventory movements
type ProductMove struct {
	MoveID      uint      `gorm:"primaryKey" json:"move_id"`
	ProductID   uint      `gorm:"not null;index" json:"product_id"`
	ProductName string    `json:"product_name"`
	MoveType    string    `gorm:"not null;index" json:"move_type"` // IN, OUT, ADJUSTMENT
	Quantity    int       `gorm:"not null" json:"quantity"`
	OldQuantity int       `json:"old_quantity"`
	NewQuantity int       `json:"new_quantity"`
	Reason      string    `json:"reason"`
	UserID      uint      `gorm:"index" json:"user_id"`
	Username    string    `json:"username"`
	IPAddress   string    `json:"ip_address"`
	Timestamp   time.Time `gorm:"not null;index" json:"timestamp"`
}

// User model with password hashing
type User struct {
	UserID       uint       `gorm:"primaryKey" json:"user_id"`
	Username     string     `gorm:"unique;not null;index" json:"username"`
	Email        string     `gorm:"unique;not null" json:"email"`
	Role         string     `gorm:"not null" json:"role"` // Administrator, Warehouse Manager, Warehouse Employee, Sales Associate
	PasswordHash string     `json:"-"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	FailedLogins int        `gorm:"default:0" json:"-"`
	LockedUntil  *time.Time `json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Session model for authentication
type Session struct {
	SessionID string    `gorm:"primaryKey" json:"session_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}

// AuditLog model for tracking all changes (FR-005)
type AuditLog struct {
	LogID      uint      `gorm:"primaryKey" json:"log_id"`
	UserID     uint      `gorm:"index" json:"user_id"`
	Username   string    `json:"username"`
	Action     string    `gorm:"not null;index" json:"action"` // CREATE, UPDATE, DELETE, LOGIN, LOGOUT
	EntityType string    `json:"entity_type"`                  // Product, User, Session
	EntityID   uint      `json:"entity_id"`
	OldValue   string    `json:"old_value"`
	NewValue   string    `json:"new_value"`
	IPAddress  string    `json:"ip_address"`
	Timestamp  time.Time `gorm:"not null;index" json:"timestamp"`
	Success    bool      `gorm:"default:true" json:"success"`
	ErrorMsg   string    `json:"error_msg"`
}

// LowStockAlert model (FR-004)
type LowStockAlert struct {
	AlertID        uint       `gorm:"primaryKey" json:"alert_id"`
	ProductID      uint       `gorm:"index" json:"product_id"`
	ProductName    string     `json:"product_name"`
	CurrentQty     int        `json:"current_qty"`
	Threshold      int        `json:"threshold"`
	Severity       string     `json:"severity"` // CRITICAL, WARNING
	CreatedAt      time.Time  `json:"created_at"`
	Acknowledged   bool       `gorm:"default:false" json:"acknowledged"`
	AcknowledgedBy uint       `json:"acknowledged_by"`
	AcknowledgedAt *time.Time `json:"acknowledged_at"`
}

var DB *gorm.DB

// ============================================================================
// CONFIGURATION
// ============================================================================

const (
	SessionDuration      = 30 * time.Minute // Updated to 30 minutes per security requirements
	MaxFailedLogins      = 5
	AccountLockDuration  = 30 * time.Minute
	MaxNameLength        = 100 // Updated per security requirements
	MaxDescriptionLength = 500
	MaxQuantity          = 999999
	BcryptCost           = 12
)

// Role constants
const (
	RoleAdmin     = "Administrator"
	RoleManager   = "Warehouse Manager"
	RoleEmployee  = "Warehouse Employee"
	RoleAssociate = "Sales Associate"
)

// Move type constants
const (
	MoveTypeIN         = "IN"
	MoveTypeOUT        = "OUT"
	MoveTypeADJUSTMENT = "ADJUSTMENT"
)

// ============================================================================
// INPUT VALIDATION
// ============================================================================

// Validation patterns
var (
	// SR-INPUT-02: Alphanumeric, spaces, periods, hyphens only
	productNameRegex = regexp.MustCompile(`^[A-Za-z0-9 .-]+$`)
	skuRegex         = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	categoryRegex    = regexp.MustCompile(`^[A-Za-z0-9 &-]+$`)
	emailRegex       = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	usernameRegex    = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// ValidateProductName validates product names (SR-INPUT-01, SR-INPUT-02, SR-INPUT-04)
func ValidateProductName(name string) error {
	if name == "" {
		return fmt.Errorf("product name is required")
	}
	if len(name) > MaxNameLength {
		return fmt.Errorf("product name must not exceed %d characters", MaxNameLength)
	}
	if !productNameRegex.MatchString(name) {
		return fmt.Errorf("product name contains invalid characters")
	}
	return nil
}

// ValidateSKU validates SKU format
func ValidateSKU(sku string) error {
	if sku == "" {
		return fmt.Errorf("SKU is required")
	}
	if len(sku) > 50 {
		return fmt.Errorf("SKU must not exceed 50 characters")
	}
	if !skuRegex.MatchString(sku) {
		return fmt.Errorf("SKU contains invalid characters")
	}
	return nil
}

// ValidateDescription validates product description (SR-INPUT-04)
func ValidateDescription(desc string) error {
	if len(desc) > MaxDescriptionLength {
		return fmt.Errorf("description must not exceed %d characters", MaxDescriptionLength)
	}
	// Remove any potential XSS characters
	if strings.ContainsAny(desc, "<>\"'&") {
		return fmt.Errorf("description contains invalid characters")
	}
	return nil
}

// ValidateCategory validates category
func ValidateCategory(category string) error {
	if category == "" {
		return fmt.Errorf("category is required")
	}
	if len(category) > 100 {
		return fmt.Errorf("category must not exceed 100 characters")
	}
	if !categoryRegex.MatchString(category) {
		return fmt.Errorf("category contains invalid characters")
	}
	return nil
}

// ValidateQuantity validates quantity (SR-INPUT-03)
func ValidateQuantity(qty int) error {
	if qty < 0 {
		return fmt.Errorf("quantity cannot be negative")
	}
	if qty > MaxQuantity {
		return fmt.Errorf("quantity cannot exceed %d", MaxQuantity)
	}
	return nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// ValidateUsername validates username
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if len(username) < 3 || len(username) > 50 {
		return fmt.Errorf("username must be between 3 and 50 characters")
	}
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username contains invalid characters")
	}
	return nil
}

// ValidatePassword validates password strength (SR-AUTH-02)
func ValidatePassword(password string) error {
	if len(password) < 12 {
		return fmt.Errorf("password must be at least 12 characters")
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasNumber = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return fmt.Errorf("password must contain at least one uppercase letter, lowercase letter, number, and special character")
	}

	return nil
}

// SanitizeInput removes potentially dangerous characters
func SanitizeInput(input string) string {
	// Remove HTML/script tags and dangerous characters
	input = strings.ReplaceAll(input, "<", "")
	input = strings.ReplaceAll(input, ">", "")
	input = strings.ReplaceAll(input, "\"", "")
	input = strings.ReplaceAll(input, "'", "")
	input = strings.ReplaceAll(input, "&", "")
	return strings.TrimSpace(input)
}

// ============================================================================
// DATABASE INITIALIZATION
// ============================================================================
func InitDB(dsn string) error {
	var err error

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true, // OK: ORM-level setting
	})
	if err != nil {
		return err
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	// Connection pool settings (real pooling)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(1 * time.Minute)

	// Optional: verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return sqlDB.PingContext(ctx)
}
func InitDatabase() error {
	// Get DSN from environment or use default (SR-DATA-04)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=dundermifflin port=5432 sslmode=disable"
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		return err
	}

	// Configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto-migrate models
	err = DB.AutoMigrate(&Product{}, &User{}, &Session{}, &AuditLog{}, &LowStockAlert{}, &ProductMove{})
	if err != nil {
		return err
	}

	// Create indexes for performance
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_products_quantity ON products(quantity)")
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC)")
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)")
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_product_moves_product_id ON product_moves(product_id)")
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_product_moves_timestamp ON product_moves(timestamp DESC)")

	// SR-DATA-06: Restrict DELETE permissions on audit logs (implement via database triggers if needed)
	// This would typically be done at the database level with GRANT/REVOKE statements

	log.Println("✓ Database connected and migrated successfully (PostgreSQL)")
	log.Println("✓ ProductMove tracking system initialized")
	return nil
}

// Seed initial users with hashed passwords (SR-AUTH-01)
// func SeedUsers() error {
// 	users := []struct {
// 		Username string
// 		Email    string
// 		Role     string
// 		Password string
// 	}{
// 		{Username: "admin", Email: "admin@dundermifflin.com", Role: RoleAdmin, Password: "Admin123!@#$"},
// 		{Username: "mscott", Email: "michael@dundermifflin.com", Role: RoleManager, Password: "Manager123!@#$"},
// 		{Username: "dwight", Email: "dwight@dundermifflin.com", Role: RoleEmployee, Password: "Employee123!@#$"},
// 		{Username: "jim", Email: "jim@dundermifflin.com", Role: RoleAssociate, Password: "Sales123!@#$"},
// 	}

// 	for _, u := range users {
// 		var existing User
// 		result := DB.Where("username = ?", u.Username).First(&existing)
// 		if result.RowsAffected == 0 {
// 			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), BcryptCost)
// 			if err != nil {
// 				return err
// 			}
// 			user := User{
// 				Username:     u.Username,
// 				Email:        u.Email,
// 				Role:         u.Role,
// 				PasswordHash: string(hashedPassword),
// 				IsActive:     true,
// 			}
// 			if err := DB.Create(&user).Error; err != nil {
// 				return err
// 			}
// 			log.Printf("✓ Created user: %s (password: %s)", u.Username, u.Password)
// 		}
// 	}
// 	return nil
// }

// Seed initial users with hashed passwords (SR-AUTH-01)
func SeedUsers() error {
	users := []struct {
		Username string
		Email    string
		Role     string
		Password string
	}{
		{Username: "admin", Email: "admin@dundermifflin.com", Role: RoleAdmin, Password: "Admin123!@#$"},
		{Username: "mscott", Email: "michael@dundermifflin.com", Role: RoleManager, Password: "Manager123!@#$"},
		{Username: "dwight", Email: "dwight@dundermifflin.com", Role: RoleEmployee, Password: "Employee123!@#$"},
		{Username: "jim", Email: "jim@dundermifflin.com", Role: RoleAssociate, Password: "Sales123!@#$"},
	}

	for _, u := range users {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), BcryptCost)
		if err != nil {
			return err
		}

		var existing User
		result := DB.Where("username = ?", u.Username).First(&existing)
		if result.RowsAffected == 0 {
			// Create new user
			user := User{
				Username:     u.Username,
				Email:        u.Email,
				Role:         u.Role,
				PasswordHash: string(hashedPassword),
				IsActive:     true,
			}
			if err := DB.Create(&user).Error; err != nil {
				return err
			}
			log.Printf("✓ Created user: %s (password: %s)", u.Username, u.Password)
		} else {
			// Update existing user's password hash to ensure it matches
			existing.PasswordHash = string(hashedPassword)
			existing.IsActive = true
			existing.FailedLogins = 0
			existing.LockedUntil = nil
			if err := DB.Save(&existing).Error; err != nil {
				return err
			}
			log.Printf("✓ Updated user: %s (password reset to: %s)", u.Username, u.Password)
		}
	}
	return nil
}

// ============================================================================
// AUTHENTICATION & AUTHORIZATION MIDDLEWARE
// ============================================================================

// Generate secure session ID
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// AuthMiddleware validates session and adds user context (SR-AUTH-06, SR-AUTH-07)
func AuthMiddleware(c *fiber.Ctx) error {
	sessionID := c.Cookies("session_id")
	if sessionID == "" {
		// Check Authorization header as fallback
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			sessionID = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if sessionID == "" {
		LogSecurityEvent(0, "", "UNAUTHORIZED_ACCESS", "Session", 0, "", "", c.IP(), false, "No session token provided")
		return c.Status(401).JSON(fiber.Map{
			"error": "Authentication required",
			"code":  "UNAUTHORIZED",
		})
	}

	var session Session
	if err := DB.Where("session_id = ? AND expires_at > ?", sessionID, time.Now()).First(&session).Error; err != nil {
		LogSecurityEvent(0, "", "INVALID_SESSION", "Session", 0, "", "", c.IP(), false, "Invalid or expired session")
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid or expired session",
			"code":  "SESSION_EXPIRED",
		})
	}

	var user User
	if err := DB.First(&user, session.UserID).Error; err != nil {
		LogSecurityEvent(0, "", "USER_NOT_FOUND", "User", 0, "", "", c.IP(), false, "User not found for valid session")
		return c.Status(401).JSON(fiber.Map{
			"error": "User not found",
			"code":  "USER_NOT_FOUND",
		})
	}

	if !user.IsActive {
		LogSecurityEvent(user.UserID, user.Username, "ACCOUNT_DISABLED", "User", user.UserID, "", "", c.IP(), false, "Inactive account attempted access")
		return c.Status(403).JSON(fiber.Map{
			"error": "Account is disabled",
			"code":  "ACCOUNT_DISABLED",
		})
	}

	// Add user to context
	c.Locals("user", user)
	c.Locals("session", session)
	return c.Next()
}

// RoleMiddleware checks if user has required role (SR-AUTH-06, SR-AUTH-07)
func RoleMiddleware(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals("user").(User)
		if !ok {
			LogSecurityEvent(0, "", "AUTH_REQUIRED", "", 0, "", "", c.IP(), false, "Authentication required")
			return c.Status(401).JSON(fiber.Map{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
		}

		for _, role := range allowedRoles {
			if user.Role == role {
				return c.Next()
			}
		}

		// SR-AUDIT-03: Log failed authorization attempts
		LogSecurityEvent(user.UserID, user.Username, "ACCESS_DENIED", c.Path(), 0, "", user.Role, c.IP(), false, "Insufficient permissions")

		return c.Status(403).JSON(fiber.Map{
			"error": "Insufficient permissions",
			"code":  "FORBIDDEN",
		})
	}
}

// ============================================================================
// LOGGING FUNCTIONS
// ============================================================================

// LogAudit creates an audit log entry (SR-AUDIT-01, SR-AUDIT-02, SR-AUDIT-04)
func LogAudit(userID uint, username, action, entityType string, entityID uint, oldValue, newValue, ipAddress string, success bool, errorMsg string) {
	auditLog := AuditLog{
		UserID:     userID,
		Username:   username,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		OldValue:   oldValue,
		NewValue:   newValue,
		IPAddress:  ipAddress,
		Timestamp:  time.Now(),
		Success:    success,
		ErrorMsg:   errorMsg,
	}

	if err := DB.Create(&auditLog).Error; err != nil {
		log.Printf("⚠️  Failed to create audit log: %v", err)
	}
}

// LogSecurityEvent logs security-related events (SR-AUDIT-01, SR-AUDIT-03)
func LogSecurityEvent(userID uint, username, action, resource string, resourceID uint, oldValue, newValue, ipAddress string, success bool, details string) {
	// Log to console for immediate visibility (SR-DATA-02: Don't log sensitive data)
	logLevel := "INFO"
	if !success {
		logLevel = "SECURITY"
	}

	log.Printf("[%s] User: %s (ID: %d) | Action: %s | Resource: %s | IP: %s | Success: %t | Details: %s",
		logLevel, username, userID, action, resource, ipAddress, success, details)

	// Also store in audit log
	LogAudit(userID, username, action, resource, resourceID, oldValue, newValue, ipAddress, success, details)
}

// RecordProductMove - Track inventory movement (SR-AUDIT-02)
func RecordProductMove(productID uint, productName string, moveType string, quantity, oldQty, newQty int, reason string, userID uint, username, ipAddress string) {
	move := ProductMove{
		ProductID:   productID,
		ProductName: productName,
		MoveType:    moveType,
		Quantity:    quantity,
		OldQuantity: oldQty,
		NewQuantity: newQty,
		Reason:      reason,
		UserID:      userID,
		Username:    username,
		IPAddress:   ipAddress,
		Timestamp:   time.Now(),
	}

	if err := DB.Create(&move).Error; err != nil {
		log.Printf("⚠️  Failed to record product move: %v", err)
	} else {
		log.Printf("📦 Recorded %s movement for %s: %d → %d (Qty change: %d)",
			moveType, productName, oldQty, newQty, quantity)
	}
}

// ============================================================================
// AUTHENTICATION ROUTES
// ============================================================================

func Login(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		LogSecurityEvent(0, req.Username, "LOGIN_FAILED", "User", 0, "", "", c.IP(), false, "Invalid request body")
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		LogSecurityEvent(0, req.Username, "LOGIN_FAILED", "User", 0, "", "", c.IP(), false, "Missing credentials")
		return c.Status(400).JSON(fiber.Map{
			"error": "Username and password are required",
			"code":  "VALIDATION_ERROR",
		})
	}

	var user User
	if err := DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		// SR-ERROR-03: Generic error message for authentication failures
		LogSecurityEvent(0, req.Username, "LOGIN_FAILED", "User", 0, "", "", c.IP(), false, "Invalid username")
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid credentials",
			"code":  "INVALID_CREDENTIALS",
		})
	}

	// SR-AUTH-03: Check if account is locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		LogSecurityEvent(user.UserID, user.Username, "LOGIN_FAILED", "User", user.UserID, "", "", c.IP(), false, "Account locked")
		return c.Status(403).JSON(fiber.Map{
			"error":        "Account temporarily locked due to multiple failed login attempts",
			"code":         "ACCOUNT_LOCKED",
			"locked_until": user.LockedUntil.Format(time.RFC3339),
		})
	}

	// Verify password (SR-AUTH-01)
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// Increment failed login count
		user.FailedLogins++

		// SR-AUTH-03: Lock account if too many failed attempts
		if user.FailedLogins >= MaxFailedLogins {
			lockUntil := time.Now().Add(AccountLockDuration)
			user.LockedUntil = &lockUntil
			log.Printf("🔒 Account locked: %s (until %s)", user.Username, lockUntil.Format(time.RFC3339))
		}

		DB.Save(&user)
		LogSecurityEvent(user.UserID, user.Username, "LOGIN_FAILED", "User", user.UserID, "", "", c.IP(), false, "Invalid password")

		// SR-ERROR-03: Generic error message
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid credentials",
			"code":  "INVALID_CREDENTIALS",
		})
	}

	// Reset failed logins on successful login
	user.FailedLogins = 0
	user.LockedUntil = nil
	DB.Save(&user)

	// Generate session
	sessionID, err := generateSessionID()
	if err != nil {
		LogSecurityEvent(user.UserID, user.Username, "LOGIN_FAILED", "Session", 0, "", "", c.IP(), false, "Failed to generate session")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create session",
			"code":  "SESSION_ERROR",
		})
	}

	session := Session{
		SessionID: sessionID,
		UserID:    user.UserID,
		ExpiresAt: time.Now().Add(SessionDuration), // SR-AUTH-04
		IPAddress: c.IP(),
		UserAgent: string(c.Request().Header.UserAgent()),
	}

	if err := DB.Create(&session).Error; err != nil {
		LogSecurityEvent(user.UserID, user.Username, "LOGIN_FAILED", "Session", 0, "", "", c.IP(), false, "Database error creating session")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create session",
			"code":  "DATABASE_ERROR",
		})
	}

	// SR-AUTH-05: Set secure cookie
	// c.Cookie(&fiber.Cookie{
	// 	Name:     "session_id",
	// 	Value:    sessionID,
	// 	Expires:  session.ExpiresAt,
	// 	HTTPOnly: true,
	// 	Secure:   true, // Set to true in production with HTTPS (SR-DATA-01)
	// 	SameSite: "Strict",
	// })

	// SR-AUTH-05: Set secure cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/", // ALL routes
		Domain:   "",  // Empty for localhost
		MaxAge:   int(SessionDuration.Seconds()),
		Expires:  session.ExpiresAt,
		Secure:   false, // FALSE for http://
		HTTPOnly: true,  // Prevents JS access (security)
		SameSite: "Lax", // Lax not Strict (allows navigation)
	})

	// SR-AUDIT-01: Log successful authentication
	LogSecurityEvent(user.UserID, user.Username, "LOGIN_SUCCESS", "User", user.UserID, "", user.Role, c.IP(), true, "")

	return c.JSON(fiber.Map{
		"message":  "Login successful",
		"user_id":  user.UserID,
		"username": user.Username,
		"role":     user.Role,
		"email":    user.Email,
	})
}

func Logout(c *fiber.Ctx) error {
	user := c.Locals("user").(User)
	session := c.Locals("session").(Session)

	// Delete session from database
	DB.Delete(&session)

	// Clear cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
	})

	// SR-AUDIT-01: Log logout
	LogSecurityEvent(user.UserID, user.Username, "LOGOUT", "User", user.UserID, "", "", c.IP(), true, "")

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

func GetCurrentUser(c *fiber.Ctx) error {
	user := c.Locals("user").(User)

	return c.JSON(fiber.Map{
		"user_id":  user.UserID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})
}

// CheckEmailAvailable checks if email is available
func CheckEmailAvailable(c *fiber.Ctx) error {
	email := c.Query("email")

	if email == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Email parameter is required",
			"code":  "VALIDATION_ERROR",
		})
	}

	// Validate email format
	if err := ValidateEmail(email); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
			"code":  "INVALID_EMAIL",
		})
	}

	var user User
	result := DB.Where("email = ?", email).First(&user)

	return c.JSON(fiber.Map{
		"email":     email,
		"available": result.RowsAffected == 0,
		"message": func() string {
			if result.RowsAffected == 0 {
				return "Email is available"
			}
			return "Email is already registered"
		}(),
	})
}

// CheckUsernameAvailable checks if username is available
func CheckUsernameAvailable(c *fiber.Ctx) error {
	username := c.Query("username")

	if username == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Username parameter is required",
			"code":  "VALIDATION_ERROR",
		})
	}

	// Validate username format
	if err := ValidateUsername(username); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
			"code":  "INVALID_USERNAME",
		})
	}

	var user User
	result := DB.Where("username = ?", username).First(&user)

	return c.JSON(fiber.Map{
		"username":  username,
		"available": result.RowsAffected == 0,
		"message": func() string {
			if result.RowsAffected == 0 {
				return "Username is available"
			}
			return "Username is already taken"
		}(),
	})
}

// ============================================================================
// PRODUCT ROUTES
// ============================================================================

func GetInventory(c *fiber.Ctx) error {
	// Filters - sanitize inputs
	search := SanitizeInput(c.Query("search", ""))
	category := SanitizeInput(c.Query("category", ""))

	// Pagination with validation (SR-INPUT-03)
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.Query("page_size", "50"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	query := DB.Model(&Product{})

	// Apply search filter - using parameterized query (SR-INPUT-05)
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(sku) LIKE ? OR LOWER(category) LIKE ?",
			searchTerm, searchTerm, searchTerm)
	}

	// Apply category filter - using parameterized query
	if category != "" {
		query = query.Where("category = ?", category)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get paginated results
	var products []Product
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("name ASC").Find(&products).Error; err != nil {
		log.Printf("ERROR: Failed to fetch products: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch products",
			"code":  "DATABASE_ERROR",
		})
	}

	return c.JSON(fiber.Map{
		"products":    products,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func GetProduct(c *fiber.Ctx) error {
	id := c.Params("id")

	var product Product
	// SR-INPUT-05: Using parameterized query
	if err := DB.First(&product, "product_id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Product not found",
			"code":  "NOT_FOUND",
		})
	}

	return c.JSON(product)
}

func AddProduct(c *fiber.Ctx) error {
	user := c.Locals("user").(User)

	var product Product
	if err := c.BodyParser(&product); err != nil {
		LogSecurityEvent(user.UserID, user.Username, "CREATE_PRODUCT_FAILED", "Product", 0, "", "", c.IP(), false, "Invalid request body")
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// SR-INPUT-01, SR-INPUT-02, SR-INPUT-04: Validate all inputs
	if err := ValidateSKU(product.SKU); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
	}

	if err := ValidateProductName(product.Name); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
	}

	if err := ValidateCategory(product.Category); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
	}

	if err := ValidateDescription(product.Description); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
	}

	// SR-INPUT-03: Validate quantity
	if err := ValidateQuantity(product.Quantity); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
	}

	// Sanitize inputs
	product.Name = SanitizeInput(product.Name)
	product.Description = SanitizeInput(product.Description)
	product.Category = SanitizeInput(product.Category)

	// Set metadata
	product.CreatedBy = user.UserID
	product.UpdatedBy = user.UserID
	product.Version = 1

	// SR-INPUT-05: Create product using GORM (parameterized)
	if err := DB.Create(&product).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			LogSecurityEvent(user.UserID, user.Username, "CREATE_PRODUCT_FAILED", "Product", 0, "", product.SKU, c.IP(), false, "Duplicate SKU")
			return c.Status(409).JSON(fiber.Map{
				"error": "Product with this SKU already exists",
				"code":  "DUPLICATE_SKU",
			})
		}
		log.Printf("ERROR: Failed to create product: %v", err)
		LogSecurityEvent(user.UserID, user.Username, "CREATE_PRODUCT_FAILED", "Product", 0, "", "", c.IP(), false, "Database error")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create product",
			"code":  "DATABASE_ERROR",
		})
	}

	// SR-AUDIT-02: Log product creation
	LogAudit(user.UserID, user.Username, "CREATE_PRODUCT", "Product", product.ProductID,
		"", product.Name, c.IP(), true, "")

	// Record initial inventory move
	RecordProductMove(
		product.ProductID,
		product.Name,
		MoveTypeIN,
		product.Quantity,
		0,
		product.Quantity,
		"Initial inventory",
		user.UserID,
		user.Username,
		c.IP(),
	)

	// Check for low stock
	if product.Quantity <= product.LowStockThreshold {
		CreateLowStockAlert(product)
	}

	return c.Status(201).JSON(product)
}

func UpdateProduct(c *fiber.Ctx) error {
	user := c.Locals("user").(User)
	id := c.Params("id")

	var existing Product
	// SR-INPUT-05: Parameterized query
	if err := DB.First(&existing, "product_id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Product not found",
			"code":  "NOT_FOUND",
		})
	}

	var updates struct {
		Quantity int `json:"quantity"`
		Version  int `json:"version"`
	}

	if err := c.BodyParser(&updates); err != nil {
		LogSecurityEvent(user.UserID, user.Username, "UPDATE_PRODUCT_FAILED", "Product", existing.ProductID, "", "", c.IP(), false, "Invalid request body")
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Optimistic locking check
	if updates.Version != existing.Version {
		LogSecurityEvent(user.UserID, user.Username, "UPDATE_PRODUCT_FAILED", "Product", existing.ProductID, "", "", c.IP(), false, "Version conflict")
		return c.Status(409).JSON(fiber.Map{
			"error":   "Product was modified by another user. Please refresh and try again.",
			"code":    "VERSION_CONFLICT",
			"version": existing.Version,
		})
	}

	// SR-INPUT-03: Validate quantity
	if err := ValidateQuantity(updates.Quantity); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
	}

	oldQty := existing.Quantity
	newQty := updates.Quantity
	quantityChange := newQty - oldQty

	// Update product
	existing.Quantity = newQty
	existing.Version++
	existing.UpdatedBy = user.UserID

	if err := DB.Save(&existing).Error; err != nil {
		log.Printf("ERROR: Failed to update product: %v", err)
		LogAudit(user.UserID, user.Username, "UPDATE_PRODUCT", "Product", existing.ProductID,
			strconv.Itoa(oldQty), strconv.Itoa(newQty), c.IP(), false, err.Error())
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update product",
			"code":  "DATABASE_ERROR",
		})
	}

	// SR-AUDIT-02: Log update
	LogAudit(user.UserID, user.Username, "UPDATE_PRODUCT", "Product", existing.ProductID,
		strconv.Itoa(oldQty), strconv.Itoa(newQty), c.IP(), true, "")

	// Record product movement
	moveType := MoveTypeADJUSTMENT
	if quantityChange > 0 {
		moveType = MoveTypeIN
	} else if quantityChange < 0 {
		moveType = MoveTypeOUT
	}

	reason := fmt.Sprintf("Quantity adjusted by %s", user.Username)
	RecordProductMove(
		existing.ProductID,
		existing.Name,
		moveType,
		quantityChange,
		oldQty,
		newQty,
		reason,
		user.UserID,
		user.Username,
		c.IP(),
	)

	// Check for low stock
	if existing.Quantity <= existing.LowStockThreshold {
		CreateLowStockAlert(existing)
	}

	return c.JSON(existing)
}

func DeleteProduct(c *fiber.Ctx) error {
	user := c.Locals("user").(User)
	id := c.Params("id")

	var product Product
	// SR-INPUT-05: Parameterized query
	if err := DB.First(&product, "product_id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Product not found",
			"code":  "NOT_FOUND",
		})
	}

	if err := DB.Delete(&product).Error; err != nil {
		log.Printf("ERROR: Failed to delete product: %v", err)
		LogAudit(user.UserID, user.Username, "DELETE_PRODUCT", "Product", product.ProductID,
			product.Name, "", c.IP(), false, err.Error())
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete product",
			"code":  "DATABASE_ERROR",
		})
	}

	// SR-AUDIT-02: Log deletion
	LogAudit(user.UserID, user.Username, "DELETE_PRODUCT", "Product", product.ProductID,
		product.Name, "", c.IP(), true, "")

	// Delete related alerts
	DB.Where("product_id = ?", product.ProductID).Delete(&LowStockAlert{})

	return c.JSON(fiber.Map{
		"message": "Product deleted successfully",
	})
}

// GetProductMoves gets movement history for a specific product
func GetProductMoves(c *fiber.Ctx) error {
	productID := c.Params("id")

	// Verify product exists - parameterized query
	var product Product
	if err := DB.First(&product, "product_id = ?", productID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Product not found",
			"code":  "NOT_FOUND",
		})
	}

	// Pagination with validation
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.Query("page_size", "50"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	// Optional filters - sanitized
	moveType := strings.ToUpper(SanitizeInput(c.Query("move_type", "")))

	query := DB.Where("product_id = ?", productID)

	if moveType != "" {
		query = query.Where("move_type = ?", moveType)
	}

	// Get total count
	var total int64
	query.Model(&ProductMove{}).Count(&total)

	// Get paginated moves
	var moves []ProductMove
	offset := (page - 1) * pageSize
	if err := query.Order("timestamp DESC").Limit(pageSize).Offset(offset).Find(&moves).Error; err != nil {
		log.Printf("ERROR: Failed to fetch product moves: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch product moves",
			"code":  "DATABASE_ERROR",
		})
	}

	return c.JSON(fiber.Map{
		"product_id":   product.ProductID,
		"product_name": product.Name,
		"moves":        moves,
		"total":        total,
		"page":         page,
		"page_size":    pageSize,
		"total_pages":  (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetAllProductMoves gets all product moves across all products
func GetAllProductMoves(c *fiber.Ctx) error {
	// Pagination with validation
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.Query("page_size", "50"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	// Filters - sanitized
	moveType := strings.ToUpper(SanitizeInput(c.Query("move_type", "")))
	userID := SanitizeInput(c.Query("user_id", ""))

	query := DB.Model(&ProductMove{})

	if moveType != "" {
		query = query.Where("move_type = ?", moveType)
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get paginated moves
	var moves []ProductMove
	offset := (page - 1) * pageSize
	if err := query.Order("timestamp DESC").Limit(pageSize).Offset(offset).Find(&moves).Error; err != nil {
		log.Printf("ERROR: Failed to fetch product moves: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch product moves",
			"code":  "DATABASE_ERROR",
		})
	}

	return c.JSON(fiber.Map{
		"moves":       moves,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// ============================================================================
// LOW STOCK ALERT ROUTES (FR-004)
// ============================================================================

func CreateLowStockAlert(product Product) {
	// Check if alert already exists
	var existingAlert LowStockAlert
	result := DB.Where("product_id = ? AND acknowledged = false", product.ProductID).First(&existingAlert)
	if result.RowsAffected > 0 {
		return
	}

	severity := "WARNING"
	if product.Quantity == 0 {
		severity = "CRITICAL"
	}

	alert := LowStockAlert{
		ProductID:   product.ProductID,
		ProductName: product.Name,
		CurrentQty:  product.Quantity,
		Threshold:   product.LowStockThreshold,
		Severity:    severity,
		CreatedAt:   time.Now(),
	}

	DB.Create(&alert)
	log.Printf("Low stock alert created for product: %s (Qty: %d, Threshold: %d)",
		product.Name, product.Quantity, product.LowStockThreshold)
}

func GetLowStockAlerts(c *fiber.Ctx) error {
	var alerts []LowStockAlert

	acknowledged := c.Query("acknowledged", "false")

	query := DB
	if acknowledged == "false" {
		query = query.Where("acknowledged = false")
	}

	if err := query.Order("severity DESC, current_qty ASC, created_at DESC").Find(&alerts).Error; err != nil {
		log.Printf("ERROR: Failed to retrieve alerts: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve alerts",
			"code":  "DATABASE_ERROR",
		})
	}

	return c.JSON(fiber.Map{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

func AcknowledgeAlert(c *fiber.Ctx) error {
	user := c.Locals("user").(User)
	id := c.Params("id")

	var alert LowStockAlert
	if err := DB.First(&alert, "alert_id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Alert not found",
			"code":  "NOT_FOUND",
		})
	}

	now := time.Now()
	alert.Acknowledged = true
	alert.AcknowledgedBy = user.UserID
	alert.AcknowledgedAt = &now

	if err := DB.Save(&alert).Error; err != nil {
		log.Printf("ERROR: Failed to acknowledge alert: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to acknowledge alert",
			"code":  "DATABASE_ERROR",
		})
	}

	LogAudit(user.UserID, user.Username, "ACKNOWLEDGE_ALERT", "Alert", alert.AlertID, "", "", c.IP(), true, "")

	return c.JSON(fiber.Map{
		"message": "Alert acknowledged successfully",
		"alert":   alert,
	})
}

// ============================================================================
// AUDIT LOG ROUTES (FR-005)
// ============================================================================

func GetAuditLogs(c *fiber.Ctx) error {
	var logs []AuditLog

	// Pagination with validation
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.Query("page_size", "50"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	// Filters - sanitized
	userID := SanitizeInput(c.Query("user_id", ""))
	action := SanitizeInput(c.Query("action", ""))
	entityType := SanitizeInput(c.Query("entity_type", ""))

	query := DB
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}

	var total int64
	query.Model(&AuditLog{}).Count(&total)

	if err := query.Order("timestamp DESC").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
		log.Printf("ERROR: Failed to retrieve audit logs: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve audit logs",
			"code":  "DATABASE_ERROR",
		})
	}

	return c.JSON(fiber.Map{
		"logs":        logs,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// ============================================================================
// HEALTH CHECK
// ============================================================================

func HealthCheck(c *fiber.Ctx) error {
	// Check database connection
	sqlDB, err := DB.DB()
	if err != nil {
		return c.Status(503).JSON(fiber.Map{
			"status":  "unhealthy",
			"message": "Database connection error",
		})
	}

	if err := sqlDB.Ping(); err != nil {
		return c.Status(503).JSON(fiber.Map{
			"status":  "unhealthy",
			"message": "Database ping failed",
		})
	}

	return c.JSON(fiber.Map{
		"status":    "healthy",
		"message":   "Dunder Mifflin Inventory System is running",
		"version":   "2.2-secure",
		"timestamp": time.Now(),
		"features": fiber.Map{
			"product_moves_tracking":      true,
			"email_availability_check":    true,
			"username_availability_check": true,
			"enhanced_security":           true,
			"comprehensive_logging":       true,
		},
	})
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	// Initialize database
	if err := InitDatabase(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Seed users
	if err := SeedUsers(); err != nil {
		log.Fatal("Failed to seed users:", err)
	}

	// Clean up expired sessions periodically
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			result := DB.Where("expires_at < ?", time.Now()).Delete(&Session{})
			if result.RowsAffected > 0 {
				log.Printf("🧹 Cleaned up %d expired sessions", result.RowsAffected)
			}
		}
	}()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Dunder Mifflin Inventory System v2.2 (Secure)",
		ErrorHandler: CustomErrorHandler,
		BodyLimit:    1 * 1024 * 1024, // SR-INPUT-06: 1MB max payload
	})

	// Middleware stack in order (as per security requirements)

	// 1. Logger
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${method} ${path} | IP: ${ip} | Latency: ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))

	// 2. Recover from panics
	app.Use(recover.New())

	// // 3. CORS (SR-SEC-03)
	// app.Use(cors.New(cors.Config{
	// 	AllowOrigins:     "https://localhost:3000,https://localhost:8080", // Use HTTPS in production
	// 	AllowCredentials: true,
	// 	AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
	// 	AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
	// }))

	// 3. CORS (SR-SEC-03)
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000,http://127.0.0.1:3000,http://localhost:8080",
		AllowCredentials: true, // CRITICAL for cookies
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// 4. Security Headers (SR-SEC-03)
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// c.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		c.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://m.media-amazon.com https://wallpapers.com https://*.wallpapers.com")
		return c.Next()
	})

	// 5. Rate Limiter (SR-SEC-02)
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			LogSecurityEvent(0, "", "RATE_LIMIT_EXCEEDED", c.Path(), 0, "", "", c.IP(), false, "Too many requests")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests. Please try again later.",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
		},
	}))

	// Special rate limiter for login endpoint
	loginLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			LogSecurityEvent(0, "", "LOGIN_RATE_LIMIT", "Login", 0, "", "", c.IP(), false, "Too many login attempts")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many login attempts. Please try again later.",
				"code":  "LOGIN_RATE_LIMIT",
			})
		},
	})

	// Favicon middleware
	app.Use(favicon.New(favicon.Config{
		File: "./public/img/favicon.png",
	}))

	// Handle apple touch icons
	app.Get("/apple-touch-icon.png", func(c *fiber.Ctx) error {
		return c.SendFile("./public/img/apple-touch-icon.png")
	})
	app.Get("/apple-touch-icon-precomposed.png", func(c *fiber.Ctx) error {
		return c.SendFile("./public/img/apple-touch-icon.png")
	})

	// Serve static files
	app.Static("/", "./public")

	// API Routes
	api := app.Group("/api/v1")

	// Health check (public)
	api.Get("/health", HealthCheck)

	// Authentication routes (public)
	api.Post("/auth/login", loginLimiter, Login)

	// User availability checks (public - for registration forms)
	api.Get("/users/check-email", CheckEmailAvailable)
	api.Get("/users/check-username", CheckUsernameAvailable)

	// Protected routes
	protected := api.Group("", AuthMiddleware)
	protected.Post("/auth/logout", Logout)
	protected.Get("/auth/me", GetCurrentUser)

	// Product routes (role-based access)
	protected.Get("/products", GetInventory)
	protected.Get("/products/:id", GetProduct)
	protected.Post("/products", RoleMiddleware(RoleEmployee, RoleManager, RoleAdmin), AddProduct)
	protected.Put("/products/:id", RoleMiddleware(RoleEmployee, RoleManager, RoleAdmin), UpdateProduct)
	protected.Delete("/products/:id", RoleMiddleware(RoleManager, RoleAdmin), DeleteProduct)

	// Product movement history routes
	protected.Get("/products/:id/moves", GetProductMoves)
	protected.Get("/inventory/moves", GetAllProductMoves)

	// Low stock alerts (managers and admins only)
	protected.Get("/alerts", RoleMiddleware(RoleManager, RoleAdmin), GetLowStockAlerts)
	protected.Post("/alerts/:id/acknowledge", RoleMiddleware(RoleManager, RoleAdmin), AcknowledgeAlert)

	// Audit logs (admins only)
	protected.Get("/audit-logs", RoleMiddleware(RoleAdmin), GetAuditLogs)

	// Start server
	log.Println("✓ Server starting on http://localhost:3000")
	log.Println("✓ Default users created - check console for credentials")
	log.Println("✓ Security features enabled:")
	log.Println("  - Input validation and sanitization")
	log.Println("  - Rate limiting (100 req/min global, 5 req/min login)")
	log.Println("  - Security headers (CSP, X-Frame-Options, HSTS)")
	log.Println("  - Comprehensive audit logging")
	log.Println("  - Session timeout: 30 minutes")
	log.Println("  - Account lockout: 5 failed attempts")
	log.Println("  - Bcrypt password hashing (cost: 12)")
	log.Fatal(app.Listen(":3000"))
}

// CustomErrorHandler - SR-ERROR-01, SR-ERROR-04
func CustomErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Log detailed error server-side (SR-ERROR-02)
	log.Printf("ERROR [%d]: %s | Path: %s | IP: %s", code, err.Error(), c.Path(), c.IP())

	// SR-ERROR-01: Return generic error to user
	return c.Status(code).JSON(fiber.Map{
		"error": message,
		"code":  "SERVER_ERROR",
	})
}
