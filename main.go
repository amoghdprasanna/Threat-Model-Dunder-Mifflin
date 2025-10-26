package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/logger"
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

// NEW: ProductMove model for tracking inventory movements
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
	SessionDuration      = 8 * time.Hour
	MaxFailedLogins      = 5
	AccountLockDuration  = 15 * time.Minute
	MaxNameLength        = 200
	MaxDescriptionLength = 500
	MaxQuantity          = 1000000
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
// DATABASE INITIALIZATION
// ============================================================================

func InitDatabase() error {
	// Get DSN from environment or use default
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=dundermifflin port=5432 sslmode=disable"
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		// Add connection pool settings for performance (NFR-002)
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

	// Auto-migrate models (including new ProductMove model)
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

	log.Println("✓ Database connected and migrated successfully (PostgreSQL)")
	log.Println("✓ ProductMove tracking system initialized")
	return nil
}

// Seed initial users with hashed passwords
func SeedUsers() error {
	users := []struct {
		Username string
		Email    string
		Role     string
		Password string
	}{
		{Username: "admin", Email: "admin@dundermifflin.com", Role: RoleAdmin, Password: "Admin123!"},
		{Username: "mscott", Email: "michael@dundermifflin.com", Role: RoleManager, Password: "Manager123!"},
		{Username: "dwight", Email: "dwight@dundermifflin.com", Role: RoleEmployee, Password: "Employee123!"},
		{Username: "jim", Email: "jim@dundermifflin.com", Role: RoleAssociate, Password: "Sales123!"},
	}

	for _, u := range users {
		var existing User
		result := DB.Where("username = ?", u.Username).First(&existing)
		if result.RowsAffected == 0 {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), BcryptCost)
			if err != nil {
				return err
			}
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

// AuthMiddleware validates session and adds user context
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
		return c.Status(401).JSON(fiber.Map{
			"error": "Authentication required",
			"code":  "UNAUTHORIZED",
		})
	}

	var session Session
	if err := DB.Where("session_id = ? AND expires_at > ?", sessionID, time.Now()).First(&session).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid or expired session",
			"code":  "SESSION_EXPIRED",
		})
	}

	var user User
	if err := DB.First(&user, session.UserID).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "User not found",
			"code":  "USER_NOT_FOUND",
		})
	}

	if !user.IsActive {
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

// RoleMiddleware checks if user has required role
func RoleMiddleware(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals("user").(User)
		if !ok {
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

		LogAudit(user.UserID, user.Username, "ACCESS_DENIED", "", 0, "", "", c.IP(), false, "Insufficient permissions")

		return c.Status(403).JSON(fiber.Map{
			"error": "Insufficient permissions",
			"code":  "FORBIDDEN",
		})
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// LogAudit creates an audit log entry
func LogAudit(userID uint, username, action, entityType string, entityID uint, oldValue, newValue, ipAddress string, success bool, errorMsg string) {
	log := AuditLog{
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
	DB.Create(&log)
}

// NEW: RecordProductMove - Track inventory movement
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
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Username and password are required",
			"code":  "VALIDATION_ERROR",
		})
	}

	var user User
	if err := DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		LogAudit(0, req.Username, "LOGIN", "User", 0, "", "", c.IP(), false, "User not found")
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid credentials",
			"code":  "INVALID_CREDENTIALS",
		})
	}

	// Check if account is locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return c.Status(403).JSON(fiber.Map{
			"error":        "Account temporarily locked due to multiple failed login attempts",
			"code":         "ACCOUNT_LOCKED",
			"locked_until": user.LockedUntil.Format(time.RFC3339),
		})
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// Increment failed login count
		user.FailedLogins++

		// Lock account if too many failed attempts
		if user.FailedLogins >= MaxFailedLogins {
			lockUntil := time.Now().Add(AccountLockDuration)
			user.LockedUntil = &lockUntil
			log.Printf("🔒 Account locked: %s (until %s)", user.Username, lockUntil.Format(time.RFC3339))
		}

		DB.Save(&user)
		LogAudit(user.UserID, user.Username, "LOGIN", "User", user.UserID, "", "", c.IP(), false, "Invalid password")

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
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create session",
			"code":  "SESSION_ERROR",
		})
	}

	session := Session{
		SessionID: sessionID,
		UserID:    user.UserID,
		ExpiresAt: time.Now().Add(SessionDuration),
		IPAddress: c.IP(),
		UserAgent: string(c.Request().Header.UserAgent()),
	}

	if err := DB.Create(&session).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create session",
			"code":  "DATABASE_ERROR",
		})
	}

	// Set cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  session.ExpiresAt,
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
	})

	LogAudit(user.UserID, user.Username, "LOGIN", "User", user.UserID, "", user.Role, c.IP(), true, "")

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

	// Delete session
	DB.Delete(&session)

	// Clear cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
	})

	LogAudit(user.UserID, user.Username, "LOGOUT", "User", user.UserID, "", "", c.IP(), true, "")

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

// NEW: Check Email Availability (UserService.checkEmailAvailable)
func CheckEmailAvailable(c *fiber.Ctx) error {
	email := c.Query("email")

	if email == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Email parameter is required",
			"code":  "VALIDATION_ERROR",
		})
	}

	// Validate email format (basic validation)
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid email format",
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

// NEW: Check Username Availability
func CheckUsernameAvailable(c *fiber.Ctx) error {
	username := c.Query("username")

	if username == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Username parameter is required",
			"code":  "VALIDATION_ERROR",
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
	// Filters
	search := c.Query("search", "")
	category := c.Query("category", "")

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "50"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	query := DB.Model(&Product{})

	// Apply search filter
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(sku) LIKE ? OR LOWER(category) LIKE ?",
			searchTerm, searchTerm, searchTerm)
	}

	// Apply category filter
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
	if err := DB.First(&product, id).Error; err != nil {
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
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate required fields
	if product.SKU == "" || product.Name == "" || product.Category == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Missing required fields: SKU, Name, and Category are required",
			"code":  "VALIDATION_ERROR",
		})
	}

	// Validate quantity
	if product.Quantity < 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "Quantity cannot be negative",
			"code":  "VALIDATION_ERROR",
		})
	}

	// Set metadata
	product.CreatedBy = user.UserID
	product.UpdatedBy = user.UserID
	product.Version = 1

	// Create product
	if err := DB.Create(&product).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return c.Status(409).JSON(fiber.Map{
				"error": "Product with this SKU already exists",
				"code":  "DUPLICATE_SKU",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create product",
			"code":  "DATABASE_ERROR",
		})
	}

	// Log audit
	LogAudit(user.UserID, user.Username, "CREATE_PRODUCT", "Product", product.ProductID,
		"", product.Name, c.IP(), true, "")

	// NEW: Record initial inventory move
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
	if err := DB.First(&existing, id).Error; err != nil {
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
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Optimistic locking check
	if updates.Version != existing.Version {
		return c.Status(409).JSON(fiber.Map{
			"error":   "Product was modified by another user. Please refresh and try again.",
			"code":    "VERSION_CONFLICT",
			"version": existing.Version,
		})
	}

	// Validate quantity
	if updates.Quantity < 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "Quantity cannot be negative",
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
		LogAudit(user.UserID, user.Username, "UPDATE_PRODUCT", "Product", existing.ProductID,
			strconv.Itoa(oldQty), strconv.Itoa(newQty), c.IP(), false, err.Error())
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update product",
			"code":  "DATABASE_ERROR",
		})
	}

	// Log audit
	LogAudit(user.UserID, user.Username, "UPDATE_PRODUCT", "Product", existing.ProductID,
		strconv.Itoa(oldQty), strconv.Itoa(newQty), c.IP(), true, "")

	// NEW: Record product movement
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
	if err := DB.First(&product, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Product not found",
			"code":  "NOT_FOUND",
		})
	}

	if err := DB.Delete(&product).Error; err != nil {
		LogAudit(user.UserID, user.Username, "DELETE_PRODUCT", "Product", product.ProductID,
			product.Name, "", c.IP(), false, err.Error())
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete product",
			"code":  "DATABASE_ERROR",
		})
	}

	LogAudit(user.UserID, user.Username, "DELETE_PRODUCT", "Product", product.ProductID,
		product.Name, "", c.IP(), true, "")

	// Delete related alerts
	DB.Where("product_id = ?", product.ProductID).Delete(&LowStockAlert{})

	return c.JSON(fiber.Map{
		"message": "Product deleted successfully",
	})
}

// NEW: Get Product Movement History (InventoryService.getProductMoves)
func GetProductMoves(c *fiber.Ctx) error {
	productID := c.Params("id")

	// Verify product exists
	var product Product
	if err := DB.First(&product, productID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Product not found",
			"code":  "NOT_FOUND",
		})
	}

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "50"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	// Optional filters
	moveType := c.Query("move_type", "")

	query := DB.Where("product_id = ?", productID)

	if moveType != "" {
		query = query.Where("move_type = ?", strings.ToUpper(moveType))
	}

	// Get total count
	var total int64
	query.Model(&ProductMove{}).Count(&total)

	// Get paginated moves
	var moves []ProductMove
	offset := (page - 1) * pageSize
	if err := query.Order("timestamp DESC").Limit(pageSize).Offset(offset).Find(&moves).Error; err != nil {
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

// NEW: Get All Product Moves (across all products)
func GetAllProductMoves(c *fiber.Ctx) error {
	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "50"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	// Filters
	moveType := c.Query("move_type", "")
	userID := c.Query("user_id", "")

	query := DB.Model(&ProductMove{})

	if moveType != "" {
		query = query.Where("move_type = ?", strings.ToUpper(moveType))
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
		// Alert already exists
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
	log.Printf("⚠️  Low stock alert created for product: %s (Qty: %d, Threshold: %d)",
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
	if err := DB.First(&alert, id).Error; err != nil {
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
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to acknowledge alert",
			"code":  "DATABASE_ERROR",
		})
	}

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

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "50"))
	offset := (page - 1) * pageSize

	// Filters
	userID := c.Query("user_id", "")
	action := c.Query("action", "")
	entityType := c.Query("entity_type", "")

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
		"version":   "2.1",
		"timestamp": time.Now(),
		"features": fiber.Map{
			"product_moves_tracking":      true,
			"email_availability_check":    true,
			"username_availability_check": true,
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
			DB.Where("expires_at < ?", time.Now()).Delete(&Session{})
		}
	}()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Dunder Mifflin Inventory System v2.1",
		ErrorHandler: CustomErrorHandler,
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000,http://localhost:8080",
		AllowCredentials: true,
	}))

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
	api.Post("/auth/login", Login)

	// NEW: User availability checks (public - for registration forms)
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

	// NEW: Product movement history routes
	protected.Get("/products/:id/moves", GetProductMoves) // Get moves for specific product
	protected.Get("/inventory/moves", GetAllProductMoves) // Get all moves

	// Low stock alerts (managers and admins only)
	protected.Get("/alerts", RoleMiddleware(RoleManager, RoleAdmin), GetLowStockAlerts)
	protected.Post("/alerts/:id/acknowledge", RoleMiddleware(RoleManager, RoleAdmin), AcknowledgeAlert)

	// Audit logs (admins only)
	protected.Get("/audit-logs", RoleMiddleware(RoleAdmin), GetAuditLogs)

	// Start server
	log.Println("✓ Server starting on http://localhost:3000")
	log.Println("✓ Default users created - check console for credentials")
	log.Println("✓ NEW: Product movement tracking enabled")
	log.Println("✓ NEW: Email/Username availability check endpoints added")
	log.Fatal(app.Listen(":3000"))
}

// Custom error handler
func CustomErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error": message,
		"code":  "SERVER_ERROR",
	})
}
