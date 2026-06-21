package repository

import (
	"testing"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"

	"github.com/google/uuid"
)

func TestCycleCounting(t *testing.T) {
	setupTestDB(t)

	// 1. Setup seed data
	prod := &models.Product{
		SKU:  "CC-PROD",
		Name: "Cycle Count Item",
	}
	_ = CreateProduct(prod)

	wh := &models.Warehouse{
		Code: "WH-CC",
		Name: "Cycle Count Hub",
	}
	_ = CreateWarehouse(wh)

	loc := &models.Locator{
		ID:          uuid.New().String(),
		WarehouseID: wh.ID,
		Code:        "WH-CC-A1",
		Zone:        "A",
		Aisle:       "1",
		Shelf:       "1",
		Level:       "1",
		IsActive:    true,
	}
	database.DB.Create(loc)

	// Seed 10 items in storage
	lot := models.Storage{
		ID:          uuid.New().String(),
		ProductID:   prod.ID,
		LocatorID:   loc.ID,
		BatchNumber: "BAT-CC-01",
		QtyOnHand:   10,
	}
	database.DB.Create(&lot)

	userID := "admin-1"

	// 2. Create Cycle Count
	count, err := CreateCycleCount(userID, []string{loc.ID})
	if err != nil {
		t.Fatalf("Failed to create cycle count: %v", err)
	}
	
	if count.Status != "CREATED" {
		t.Errorf("Expected CREATED status, got %v", count.Status)
	}

	// 2.5 Start Cycle Count (Transitions CREATED -> IN_PROGRESS and freezes locators)
	err = StartCycleCount(count.ID)
	if err != nil {
		t.Fatalf("Failed to start cycle count: %v", err)
	}

	// Fetch to verify lines and locators
	c, _ := GetCycleCountByID(count.ID)
	if len(c.Lines) != 1 {
		t.Fatalf("Expected 1 count line, got %d", len(c.Lines))
	}
	if c.Lines[0].ExpectedQty != 10 {
		t.Errorf("Expected 10 expected_qty, got %d", c.Lines[0].ExpectedQty)
	}

	var verifyLoc models.Locator
	database.DB.First(&verifyLoc, "id = ?", loc.ID)
	if !verifyLoc.IsFrozen {
		t.Error("Locator was not frozen after creating cycle count")
	}

	// 3. Update Count Sheet (physical count = 8, variance = -2)
	err = UpdateCycleCountLine(c.Lines[0].ID, 8)
	if err != nil {
		t.Fatalf("Failed to update cycle count line: %v", err)
	}

	// 4. Reconcile Cycle Count
	adjID, err := ReconcileCycleCount(count.ID, userID)
	if err != nil {
		t.Fatalf("Failed to reconcile cycle count: %v", err)
	}

	if adjID == "" {
		t.Fatal("Expected an adjustment ID to be returned, got empty")
	}

	// Verify Locator is unfrozen
	database.DB.First(&verifyLoc, "id = ?", loc.ID)
	if verifyLoc.IsFrozen {
		t.Error("Locator is still frozen after reconciliation")
	}

	// 5. Journalize Adjustment
	err = JournalizeInventoryAdjustment(adjID)
	if err != nil {
		t.Fatalf("Failed to journalize adjustment: %v", err)
	}

	// 6. Verify Stock
	var checkLot models.Storage
	database.DB.First(&checkLot, "id = ?", lot.ID)
	if checkLot.QtyOnHand != 8 {
		t.Errorf("Expected stock to be adjusted to 8, got %d", checkLot.QtyOnHand)
	}
}
