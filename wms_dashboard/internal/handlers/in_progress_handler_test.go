package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

func setupInProgressTestDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite db: %v", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(1)
	}

	err = db.AutoMigrate(
		&models.InventoryMovement{},
		&models.QCHold{},
		&models.InventoryAdjustment{},
		&models.InventoryKitting{},
		&models.Product{},
		&models.Warehouse{},
		&models.Locator{},
	)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	// Create view manually in sqlite test database
	viewSql := `
CREATE VIEW IF NOT EXISTS in_progress_documents AS
SELECT id, document_no, 'Movement (' || movement_type || ')' AS doc_type, created_at, status, '/wms/movements/' || id AS link FROM inventory_movements WHERE status NOT IN ('COMPLETED', 'REJECTED')
UNION ALL
SELECT id, document_no, 'QC Hold' AS doc_type, created_at, status, '/wms/qc-holds' AS link FROM qc_holds WHERE status = 'ACTIVE'
UNION ALL
SELECT id, document_no, 'Adjustment' AS doc_type, created_at, status, '/wms/adjustments' AS link FROM inventory_adjustments WHERE status = 'OPEN'
UNION ALL
SELECT id, document_no, 'Kitting' AS doc_type, created_at, status, '/wms/kitting' AS link FROM inventory_kittings WHERE status = 'OPEN';
`
	if err := db.Exec(viewSql).Error; err != nil {
		t.Fatalf("failed to create sqlite test view: %v", err)
	}

	database.DB = db
}

func TestServeInProgressDocs(t *testing.T) {
	setupInProgressTestDB(t)

	app := fiber.New(fiber.Config{
		Views: nil,
	})

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_name", "Test Admin")
		c.Locals("user_role", "System Admin")
		c.Locals("user_email", "admin@omnisync.com")
		return c.Next()
	})

	app.Get("/wms/in-progress", ServeInProgressDocs)

	// Test 1: Full HTML page load
	req := httptest.NewRequest("GET", "/wms/in-progress", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed mock request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	// Test 2: HTMX fragment request (rows only)
	reqHTMX := httptest.NewRequest("GET", "/wms/in-progress?rows_only=true", nil)
	reqHTMX.Header.Set("HX-Request", "true")
	respHTMX, _ := app.Test(reqHTMX)

	if respHTMX.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for HTMX rows_only request, got %d", respHTMX.StatusCode)
	}
}
