package repository

import (
	"testing"
	"time"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"

	"github.com/google/uuid"
)

func TestInventoryAdjustments(t *testing.T) {
	setupTestDB(t)

	// 1. Setup seed data
	prod := &models.Product{
		SKU:  "ADJ-PROD",
		Name: "Adjustment Item",
	}
	_ = CreateProduct(prod)

	wh := &models.Warehouse{
		Code: "WH-ADJ",
		Name: "Adjustment Hub",
	}
	_ = CreateWarehouse(wh)

	loc := &models.Locator{
		ID:          uuid.New().String(),
		WarehouseID: wh.ID,
		Code:        "WH-ADJ-A1",
		Zone:        "A",
		Aisle:       "1",
		Shelf:       "1",
		Level:       "1",
		IsActive:    true,
	}
	database.DB.Create(loc)

	// Scenario A: Positive Adjustment (Found Stock)
	posAdj := &models.InventoryAdjustment{
		Status:     "OPEN",
		ReasonCode: "FOUND",
		Remarks:    "Found 5 items during audit",
		CreatedBy:  "admin-1",
	}
	posLines := []models.InventoryAdjustmentLine{
		{
			ProductID: prod.ID,
			LocatorID: loc.ID,
			QtyDelta:  5,
		},
	}

	err := CreateInventoryAdjustment(posAdj, posLines)
	if err != nil {
		t.Fatalf("failed to create positive adjustment: %v", err)
	}

	// Journalize Positive Adjustment
	err = JournalizeInventoryAdjustment(posAdj.ID)
	if err != nil {
		t.Fatalf("failed to journalize positive adjustment: %v", err)
	}

	// Verify that a new Storage record is created
	var storageLots []models.Storage
	database.DB.Find(&storageLots, "product_id = ? AND locator_id = ?", prod.ID, loc.ID)
	if len(storageLots) != 1 {
		t.Fatalf("expected exactly 1 storage lot, got %d", len(storageLots))
	}
	lot1 := storageLots[0]
	if lot1.QtyOnHand != 5 || lot1.BatchNumber == "" {
		t.Errorf("unexpected storage properties after positive adjustment: %+v", lot1)
	}

	// Clean up storage lots to set up a precise negative adjustment FIFO scenario
	database.DB.Exec("DELETE FROM storages")

	// Seed two precise lots:
	// Lot 1 (Oldest): OnHand = 5, ReceivedAt = yesterday
	// Lot 2 (Newest): OnHand = 10, ReceivedAt = today
	lotOld := models.Storage{
		ID:          uuid.New().String(),
		ProductID:   prod.ID,
		LocatorID:   loc.ID,
		BatchNumber: "BAT-ADJ-OLD",
		ReceivedAt:  time.Now().Add(-24 * time.Hour),
		QtyOnHand:   5,
		QtyReserved: 0,
		QtyOnHold:   0,
		UpdatedAt:   time.Now(),
	}
	lotNew := models.Storage{
		ID:          uuid.New().String(),
		ProductID:   prod.ID,
		LocatorID:   loc.ID,
		BatchNumber: "BAT-ADJ-NEW",
		ReceivedAt:  time.Now(),
		QtyOnHand:   10,
		QtyReserved: 0,
		QtyOnHold:   0,
		UpdatedAt:   time.Now(),
	}
	database.DB.Create(&lotOld)
	database.DB.Create(&lotNew)

	// Scenario B: Validation - Fail if deducting more than physically available unreserved/unfrozen stock
	// Total available = 15. Attempting to deduct 18 should fail immediately.
	negAdjFail := &models.InventoryAdjustment{
		Status:     "OPEN",
		ReasonCode: "LOST",
		Remarks:    "Lost stock test",
		CreatedBy:  "admin-1",
	}
	failLines := []models.InventoryAdjustmentLine{
		{
			ProductID: prod.ID,
			LocatorID: loc.ID,
			QtyDelta:  -18, // Deduct 18
		},
	}
	err = CreateInventoryAdjustment(negAdjFail, failLines)
	if err == nil {
		t.Error("expected negative adjustment creation to fail due to insufficient stock, but it succeeded")
	}

	// Scenario C: Successful Negative Adjustment (FIFO Deduction Check)
	// We want to deduct 8 items. FIFO should deplete lotOld (5) completely and take 3 from lotNew.
	negAdjSuccess := &models.InventoryAdjustment{
		Status:     "OPEN",
		ReasonCode: "DAMAGED",
		Remarks:    "Deducting 8 damaged items",
		CreatedBy:  "admin-1",
	}
	successLines := []models.InventoryAdjustmentLine{
		{
			ProductID: prod.ID,
			LocatorID: loc.ID,
			QtyDelta:  -8,
		},
	}
	err = CreateInventoryAdjustment(negAdjSuccess, successLines)
	if err != nil {
		t.Fatalf("failed to create valid negative adjustment: %v", err)
	}

	// Journalize Negative Adjustment
	err = JournalizeInventoryAdjustment(negAdjSuccess.ID)
	if err != nil {
		t.Fatalf("failed to journalize negative adjustment: %v", err)
	}

	// Verify physical deductions in GORM (FIFO Lot Check)
	var updatedLotOld, updatedLotNew models.Storage
	database.DB.First(&updatedLotOld, "id = ?", lotOld.ID)
	database.DB.First(&updatedLotNew, "id = ?", lotNew.ID)

	if updatedLotOld.QtyOnHand != 0 {
		t.Errorf("expected lotOld (oldest) to be fully depleted (0), got %d", updatedLotOld.QtyOnHand)
	}
	if updatedLotNew.QtyOnHand != 7 {
		t.Errorf("expected lotNew to have 7 remaining (10 - 3), got %d", updatedLotNew.QtyOnHand)
	}

	// Scenario D: Rejecting an Open Adjustment
	rejectAdj := &models.InventoryAdjustment{
		Status:     "OPEN",
		ReasonCode: "LOST",
		Remarks:    "To be rejected",
		CreatedBy:  "admin-1",
	}
	_ = CreateInventoryAdjustment(rejectAdj, []models.InventoryAdjustmentLine{
		{
			ProductID: prod.ID,
			LocatorID: loc.ID,
			QtyDelta:  -1,
		},
	})

	err = RejectInventoryAdjustment(rejectAdj.ID, "Auditor double count error")
	if err != nil {
		t.Fatalf("failed to reject adjustment: %v", err)
	}

	var savedRejected models.InventoryAdjustment
	database.DB.First(&savedRejected, "id = ?", rejectAdj.ID)
	if savedRejected.Status != "REJECTED" {
		t.Errorf("expected adjustment status to be REJECTED, got %s", savedRejected.Status)
	}
}
