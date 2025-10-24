package main

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Product model
type Product struct {
	ProductID   uint      `gorm:"primaryKey" json:"product_id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	Quantity    int       `gorm:"not null" json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// User model
type User struct {
	UserID       uint   `gorm:"primaryKey" json:"user_id"`
	Username     string `gorm:"unique;not null" json:"username"`
	Role         string `json:"role"`
	PasswordHash string `json:"-"`
}

var DB *gorm.DB

// InitDatabase initializes the database connection
func InitDatabase() error {
	dsn := "host=localhost user=postgres dbname=dundermifflin port=5432 sslmode=disable"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	// Auto-migrate models
	err = DB.AutoMigrate(&Product{}, &User{})
	if err != nil {
		return err
	}

	log.Println("✓ Database connected and migrated successfully (PostgreSQL)")
	return nil
}

// AddProduct handles adding a new product (FR-001)
func AddProduct(c *fiber.Ctx) error {
	var product Product

	if err := c.BodyParser(&product); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Check if product already exists
	var existing Product
	result := DB.Where("name = ?", product.Name).First(&existing)
	if result.RowsAffected > 0 {
		return c.Status(409).JSON(fiber.Map{
			"error":      "Product already exists",
			"suggestion": "Consider updating the existing product",
		})
	}

	// Create new product
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	if err := DB.Create(&product).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create product",
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Product created successfully",
		"product": product,
	})
}

// GetInventory retrieves all products (FR-002)
func GetInventory(c *fiber.Ctx) error {
	var products []Product

	// Get query parameters for search/filter
	search := c.Query("search", "")

	query := DB
	if search != "" {
		// Use LIKE for compatibility with both PostgreSQL and SQLite
		query = query.Where("name LIKE ?", "%"+search+"%")
	}

	if err := query.Find(&products).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Unable to load inventory",
		})
	}

	return c.JSON(fiber.Map{
		"products": products,
		"count":    len(products),
	})
}

// GetProduct retrieves a single product by ID
func GetProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	var product Product

	if err := DB.First(&product, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Product not found",
		})
	}

	return c.JSON(product)
}

// UpdateProduct updates product quantity (FR-003)
func UpdateProduct(c *fiber.Ctx) error {
	id := c.Params("id")

	var product Product
	if err := DB.First(&product, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Product not found",
		})
	}

	var updateData struct {
		Quantity int `json:"quantity"`
	}

	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate quantity
	if updateData.Quantity < 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid quantity: cannot be negative",
		})
	}

	// Update product
	product.Quantity = updateData.Quantity
	product.UpdatedAt = time.Now()

	if err := DB.Save(&product).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error updating quantity",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Product updated successfully",
		"product": product,
	})
}

// DeleteProduct removes a product
func DeleteProduct(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := DB.Delete(&Product{}, id).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete product",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Product deleted successfully",
	})
}

func main() {
	// Initialize database
	if err := InitDatabase(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Dunder Mifflin Inventory System v1.0",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	// Serve static files
	app.Static("/", "./public")

	// API Routes
	api := app.Group("/api/v1")

	// Product routes
	api.Post("/products", AddProduct)       // FR-001
	api.Get("/products", GetInventory)      // FR-002
	api.Get("/products/:id", GetProduct)    // FR-002
	api.Put("/products/:id", UpdateProduct) // FR-003
	api.Delete("/products/:id", DeleteProduct)

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Dunder Mifflin Inventory System is running",
		})
	})

	// Start server
	log.Println("✓ Server starting on http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}
