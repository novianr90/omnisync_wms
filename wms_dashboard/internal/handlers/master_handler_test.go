package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func init() {
	// Change working directory to project root so templates can be loaded correctly
	_ = os.Chdir("../..")
}

func setupHandlerTestDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite db: %v", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(1)
	}

	err = db.AutoMigrate(
		&models.UoM{},
		&models.UoMConversion{},
		&models.Product{},
		&models.Warehouse{},
		&models.Locator{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	database.DB = db
}

func TestServeProductsMaster(t *testing.T) {
	setupHandlerTestDB(t)

	// Seed UoM and Product
	uom := &models.UoM{ID: "uom-1", Code: "pcs", Name: "Pieces"}
	database.DB.Create(uom)

	p := &models.Product{
		ID:        "prod-1",
		SKU:       "TEST-PRODUCT",
		Name:      "Test Keyboard for Handler",
		UoMID:     uom.ID,
		Price:     49.99,
		IsBundle:  false,
	}
	database.DB.Create(p)

	app := fiber.New(fiber.Config{
		Views: nil, // We use renderPage / renderPartial which parses manually
	})

	// Mock local values for user profile context required by renderPage
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_name", "Test Admin")
		c.Locals("user_role", "System Admin")
		c.Locals("user_email", "admin@omnisync.com")
		return c.Next()
	})

	app.Get("/wms/masters/products", ServeProductsMaster)

	// Test 1: Full HTML page load (HX-Request is empty/false)
	req := httptest.NewRequest("GET", "/wms/masters/products", nil)
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed mock request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	// Test 2: HTMX fragment request (HX-Request is true)
	reqHTMX := httptest.NewRequest("GET", "/wms/masters/products?rows_only=true", nil)
	reqHTMX.Header.Set("HX-Request", "true")
	respHTMX, _ := app.Test(reqHTMX)

	if respHTMX.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for HTMX request, got %d", respHTMX.StatusCode)
	}
}

func TestServeNewProductForm(t *testing.T) {
	setupHandlerTestDB(t)

	uom := &models.UoM{ID: "uom-1", Code: "pcs", Name: "Pieces"}
	database.DB.Create(uom)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_name", "Test Admin")
		c.Locals("user_role", "System Admin")
		c.Locals("user_email", "admin@omnisync.com")
		return c.Next()
	})
	app.Get("/wms/masters/products/new", ServeNewProductForm)

	req := httptest.NewRequest("GET", "/wms/masters/products/new", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for new product form, got %d", resp.StatusCode)
	}
}

func TestServeLocatorsMasterWithOccupancy(t *testing.T) {
	setupHandlerTestDB(t)

	// Also migrate Storage and Product for occupancy query
	database.DB.AutoMigrate(&models.Storage{})

	wh := &models.Warehouse{ID: "wh-occ-h", Code: "WH-OCC-H", Name: "Occ Test WH", IsActive: true}
	database.DB.Create(wh)
	loc := &models.Locator{
		ID: "loc-occ-h", WarehouseID: wh.ID,
		Code: "WH-OCC-H-A-1", Zone: "A", Aisle: "1", Shelf: "1", Level: "1",
		MaxWeight: 100, MaxVolume: 1.0, IsActive: true,
	}
	database.DB.Create(loc)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_name", "Test Admin")
		c.Locals("user_role", "System Admin")
		c.Locals("user_email", "admin@omnisync.com")
		return c.Next()
	})
	app.Get("/wms/masters/locators", ServeLocatorsMaster)

	req := httptest.NewRequest("GET", "/wms/masters/locators", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestCreateLocatorWithCapacity(t *testing.T) {
	setupHandlerTestDB(t)

	wh := &models.Warehouse{ID: "wh-cap", Code: "WH-CAP", Name: "Cap WH", IsActive: true}
	database.DB.Create(wh)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_name", "Admin")
		c.Locals("user_role", "System Admin")
		c.Locals("user_email", "admin@test.com")
		return c.Next()
	})
	app.Post("/wms/masters/locators", CreateLocator)

	body := "warehouse_id=wh-cap&zone=Zone-A&aisle=Aisle-1&shelf=Shelf-1&level=Level-1&max_weight=500&max_volume=2.5"
	req := httptest.NewRequest("POST", "/wms/masters/locators", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = io.NopCloser(strings.NewReader(body))
	req.ContentLength = int64(len(body))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	// Verify capacity was persisted
	var loc models.Locator
	database.DB.Where("warehouse_id = ?", "wh-cap").First(&loc)
	if loc.MaxWeight != 500 {
		t.Errorf("expected MaxWeight 500, got %f", loc.MaxWeight)
	}
	if loc.MaxVolume != 2.5 {
		t.Errorf("expected MaxVolume 2.5, got %f", loc.MaxVolume)
	}
}
