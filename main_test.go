package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDB creates an in-memory database for testing
func SetupTestDB() error {
	var err error
	DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return err
	}

	err = DB.AutoMigrate(&Product{}, &User{})
	return err
}

// TC-001: Test adding a new product (FR-001)
func TestAddProduct_Success(t *testing.T) {
	SetupTestDB()

	app := fiber.New()
	app.Post("/api/v1/products", AddProduct)

	product := Product{
		Name:        "Copy Paper A4",
		Description: "Premium white paper",
		Quantity:    1000,
	}

	body, _ := json.Marshal(product)
	req := httptest.NewRequest("POST", "/api/v1/products", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	assert.Equal(t, 201, resp.StatusCode)

	// Verify product in database
	var dbProduct Product
	DB.Where("name = ?", "Copy Paper A4").First(&dbProduct)
	assert.Equal(t, "Copy Paper A4", dbProduct.Name)
	assert.Equal(t, 1000, dbProduct.Quantity)
}

// TC-002: Test adding duplicate product (FR-001 Error Flow)
func TestAddProduct_Duplicate(t *testing.T) {
	SetupTestDB()

	// Add initial product
	DB.Create(&Product{
		Name:        "Copy Paper A4",
		Description: "Premium white paper",
		Quantity:    1000,
	})

	app := fiber.New()
	app.Post("/api/v1/products", AddProduct)

	product := Product{
		Name:        "Copy Paper A4",
		Description: "Another copy paper",
		Quantity:    500,
	}

	body, _ := json.Marshal(product)
	req := httptest.NewRequest("POST", "/api/v1/products", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	assert.Equal(t, 409, resp.StatusCode)
}

// TC-003: Test viewing inventory (FR-002)
func TestGetInventory_Success(t *testing.T) {
	SetupTestDB()

	// Add test products
	DB.Create(&Product{Name: "Copy Paper A4", Quantity: 1000})
	DB.Create(&Product{Name: "Legal Pad", Quantity: 500})
	DB.Create(&Product{Name: "Stapler", Quantity: 50})

	app := fiber.New()
	app.Get("/api/v1/products", GetInventory)

	req := httptest.NewRequest("GET", "/api/v1/products", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, float64(3), result["count"])
}

// TC-004: Test search/filter inventory (FR-002)
func TestGetInventory_WithSearch(t *testing.T) {
	SetupTestDB()

	// Add test products
	DB.Create(&Product{Name: "Copy Paper A4", Quantity: 1000})
	DB.Create(&Product{Name: "Legal Pad", Quantity: 500})

	app := fiber.New()
	app.Get("/api/v1/products", GetInventory)

	req := httptest.NewRequest("GET", "/api/v1/products?search=paper", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, float64(1), result["count"])
}

// TC-005: Test updating product quantity (FR-003)
func TestUpdateProduct_Success(t *testing.T) {
	SetupTestDB()

	// Create initial product
	product := Product{
		Name:     "Copy Paper A4",
		Quantity: 1000,
	}
	DB.Create(&product)

	app := fiber.New()
	app.Put("/api/v1/products/:id", UpdateProduct)

	updateData := map[string]int{"quantity": 1500}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/api/v1/products/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	assert.Equal(t, 200, resp.StatusCode)

	// Verify update in database
	var dbProduct Product
	DB.First(&dbProduct, 1)
	assert.Equal(t, 1500, dbProduct.Quantity)
}

// TC-006: Test updating with invalid quantity (FR-003 Error Flow)
func TestUpdateProduct_NegativeQuantity(t *testing.T) {
	SetupTestDB()

	// Create initial product
	product := Product{
		Name:     "Copy Paper A4",
		Quantity: 1000,
	}
	DB.Create(&product)

	app := fiber.New()
	app.Put("/api/v1/products/:id", UpdateProduct)

	updateData := map[string]int{"quantity": -100}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/api/v1/products/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	assert.Equal(t, 400, resp.StatusCode)

	// Verify quantity unchanged
	var dbProduct Product
	DB.First(&dbProduct, 1)
	assert.Equal(t, 1000, dbProduct.Quantity)
}

// TC-007: Test updating non-existent product
func TestUpdateProduct_NotFound(t *testing.T) {
	SetupTestDB()

	app := fiber.New()
	app.Put("/api/v1/products/:id", UpdateProduct)

	updateData := map[string]int{"quantity": 1500}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/api/v1/products/999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	assert.Equal(t, 404, resp.StatusCode)
}

// TC-008: Test getting single product
func TestGetProduct_Success(t *testing.T) {
	SetupTestDB()

	product := Product{
		Name:        "Copy Paper A4",
		Description: "Premium white paper",
		Quantity:    1000,
	}
	DB.Create(&product)

	app := fiber.New()
	app.Get("/api/v1/products/:id", GetProduct)

	req := httptest.NewRequest("GET", "/api/v1/products/1", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, 200, resp.StatusCode)

	var result Product
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "Copy Paper A4", result.Name)
}

// TC-009: Test deleting a product
func TestDeleteProduct_Success(t *testing.T) {
	SetupTestDB()

	product := Product{
		Name:     "Copy Paper A4",
		Quantity: 1000,
	}
	DB.Create(&product)

	app := fiber.New()
	app.Delete("/api/v1/products/:id", DeleteProduct)

	req := httptest.NewRequest("DELETE", "/api/v1/products/1", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, 200, resp.StatusCode)

	// Verify product is deleted
	var dbProduct Product
	result := DB.First(&dbProduct, 1)
	assert.Error(t, result.Error)
}
