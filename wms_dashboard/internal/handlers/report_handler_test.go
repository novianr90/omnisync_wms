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

func setupReportTestDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite db: %v", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(1)
	}

	err = db.AutoMigrate(
		&models.InventoryLedger{},
		&models.Product{},
		&models.Locator{},
		&models.Account{},
	)
	if err != nil {
		t.Fatalf("failed to migrate report test db: %v", err)
	}

	database.DB = db
}

func TestExportLedgerExcel(t *testing.T) {
	setupReportTestDB(t)

	app := fiber.New()
	app.Get("/wms/ledger/export/excel", ExportLedgerExcel)

	req := httptest.NewRequest("GET", "/wms/ledger/export/excel", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed mock request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	expectedType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if contentType != expectedType {
		t.Errorf("expected Content-Type %q, got %q", expectedType, contentType)
	}
}

func TestExportLedgerPDF(t *testing.T) {
	setupReportTestDB(t)

	app := fiber.New()
	app.Get("/wms/ledger/export/pdf", ExportLedgerPDF)

	req := httptest.NewRequest("GET", "/wms/ledger/export/pdf", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed mock request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	expectedType := "application/pdf"
	if contentType != expectedType {
		t.Errorf("expected Content-Type %q, got %q", expectedType, contentType)
	}
}
