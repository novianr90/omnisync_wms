package repository

import (
	"testing"
	"time"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"

	"github.com/google/uuid"
)

func TestKittingOperationsAndFIFO(t *testing.T) {
	setupTestDB(t)

	// 1. Setup seed data
	wh := &models.Warehouse{
		Code: "WH-KIT",
		Name: "Kitting Warehouse",
	}
	_ = CreateWarehouse(wh)

	loc := &models.Locator{
		ID:          uuid.New().String(),
		WarehouseID: wh.ID,
		Code:        "WH-KIT-A-1",
		Zone:        "A",
		Aisle:       "1",
		Shelf:       "1",
		Level:       "1",
		IsActive:    true,
	}
	database.DB.Create(loc)

	// Finished bundle product
	bundle := &models.Product{
		SKU:      "BNDL-KEYBOARD-BUNDLE",
		Name:     "Keyboard Bundle",
		IsBundle: true,
		Price:    0.0, // Calculated dynamically
	}
	_ = CreateProduct(bundle)

	// Component products
	compKey := &models.Product{
		SKU:   "COMP-KEYBOARD",
		Name:  "Mechanical Keyboard",
		Price: 50.0,
	}
	compMouse := &models.Product{
		SKU:   "COMP-MOUSE",
		Name:  "Gaming Mouse",
		Price: 20.0,
	}
	_ = CreateProduct(compKey)
	_ = CreateProduct(compMouse)

	// Seed component stocks:
	// compKey has two lots to verify oldest-first component deduction (FIFO)
	keyLot1 := models.Storage{
		ID:          uuid.New().String(),
		ProductID:   compKey.ID,
		LocatorID:   loc.ID,
		BatchNumber: "BAT-KEY-OLD",
		ReceivedAt:  time.Now().Add(-24 * time.Hour),
		QtyOnHand:   3,
		QtyReserved: 0,
		QtyOnHold:   0,
		UpdatedAt:   time.Now(),
	}
	keyLot2 := models.Storage{
		ID:          uuid.New().String(),
		ProductID:   compKey.ID,
		LocatorID:   loc.ID,
		BatchNumber: "BAT-KEY-NEW",
		ReceivedAt:  time.Now(),
		QtyOnHand:   10,
		QtyReserved: 0,
		QtyOnHold:   0,
		UpdatedAt:   time.Now(),
	}

	// compMouse has a single lot with some reservation/hold
	mouseLot := models.Storage{
		ID:          uuid.New().String(),
		ProductID:   compMouse.ID,
		LocatorID:   loc.ID,
		BatchNumber: "BAT-MOUSE",
		ReceivedAt:  time.Now(),
		QtyOnHand:   8,
		QtyReserved: 2,
		QtyOnHold:   1, // Available = 8 - 2 - 1 = 5
		UpdatedAt:   time.Now(),
	}
	database.DB.Create(&keyLot1)
	database.DB.Create(&keyLot2)
	database.DB.Create(&mouseLot)

	// Scenario A: Validation - Fail if requesting more than available unreserved/unfrozen component stock
	// compKey available: 13, compMouse available: 5.
	// Requesting 6 of compMouse should fail because only 5 is available.
	kitOrderFail := &models.InventoryKitting{
		FinishedProductID: bundle.ID,
		FinishedLocatorID: loc.ID,
		FinishedQty:       2,
		Remarks:           "Kitting failure test",
		CreatedBy:         "operator-1",
	}
	linesFail := []models.InventoryKittingLine{
		{
			ProductID:   compKey.ID,
			LocatorID:   loc.ID,
			ConsumedQty: 2,
		},
		{
			ProductID:   compMouse.ID,
			LocatorID:   loc.ID,
			ConsumedQty: 6, // Exceeds available 5
		},
	}
	err := CreateKittingOrder(kitOrderFail, linesFail)
	if err == nil {
		t.Error("expected kitting creation to fail due to insufficient component mouse stock, but it succeeded")
	}

	// Scenario B: Successful Kitting Creation
	kitOrder := &models.InventoryKitting{
		FinishedProductID: bundle.ID,
		FinishedLocatorID: loc.ID,
		FinishedQty:       2,
		Remarks:           "Kitting success test",
		CreatedBy:         "operator-1",
		Status:            "OPEN",
	}
	linesSuccess := []models.InventoryKittingLine{
		{
			ProductID:   compKey.ID,
			LocatorID:   loc.ID,
			ConsumedQty: 4, // Will consume 3 from keyLot1 (oldest) and 1 from keyLot2 (newest)
		},
		{
			ProductID:   compMouse.ID,
			LocatorID:   loc.ID,
			ConsumedQty: 2,
		},
	}
	err = CreateKittingOrder(kitOrder, linesSuccess)
	if err != nil {
		t.Fatalf("failed to create valid kitting order: %v", err)
	}

	// Verify kitting is saved
	var savedKit models.InventoryKitting
	database.DB.Preload("ComponentLines").First(&savedKit, "id = ?", kitOrder.ID)
	if savedKit.Status != "OPEN" || len(savedKit.ComponentLines) != 2 {
		t.Fatalf("unexpected kitting order details: %+v", savedKit)
	}

	// Scenario C: Journalize Kitting (Deduct components oldest-first, create bundle storage, update bundle cost)
	err = JournalizeKittingOrder(savedKit.ID)
	if err != nil {
		t.Fatalf("failed to journalize kitting: %v", err)
	}

	// Verify component stock levels (FIFO check)
	var updatedKeyLot1, updatedKeyLot2, updatedMouseLot models.Storage
	database.DB.First(&updatedKeyLot1, "id = ?", keyLot1.ID)
	database.DB.First(&updatedKeyLot2, "id = ?", keyLot2.ID)
	database.DB.First(&updatedMouseLot, "id = ?", mouseLot.ID)

	if updatedKeyLot1.QtyOnHand != 0 {
		t.Errorf("expected keyLot1 (oldest) to be fully consumed (0), got %d", updatedKeyLot1.QtyOnHand)
	}
	if updatedKeyLot2.QtyOnHand != 9 {
		t.Errorf("expected keyLot2 to have 9 remaining (10 - 1), got %d", updatedKeyLot2.QtyOnHand)
	}
	if updatedMouseLot.QtyOnHand != 6 {
		t.Errorf("expected mouseLot to have 6 remaining (8 - 2 consumed), got %d", updatedMouseLot.QtyOnHand)
	}

	// Verify Finished Product (Bundle) stock lot is created in target locator
	var bundleStorage []models.Storage
	database.DB.Find(&bundleStorage, "product_id = ? AND locator_id = ?", bundle.ID, loc.ID)
	if len(bundleStorage) != 1 {
		t.Fatalf("expected exactly 1 storage lot for the finished bundle, got %d", len(bundleStorage))
	}
	if bundleStorage[0].QtyOnHand != 2 || bundleStorage[0].BatchNumber == "" {
		t.Errorf("unexpected bundle storage properties: %+v", bundleStorage[0])
	}

	// Verify Bundle Product Price in catalog is dynamically recalculated based on actual consumed costs
	// Total cost = (4 * 50.0) + (2 * 20.0) = 200.0 + 40.0 = 240.0
	// FinishedQty = 2, so Unit Cost = 240.0 / 2 = 120.0
	var updatedBundle models.Product
	database.DB.First(&updatedBundle, "id = ?", bundle.ID)
	if updatedBundle.Price != 120.0 {
		t.Errorf("expected bundle price to be updated to 120.0, got %f", updatedBundle.Price)
	}

	// Scenario D: Rejecting an Open Kitting Order
	rejectKit := &models.InventoryKitting{
		FinishedProductID: bundle.ID,
		FinishedLocatorID: loc.ID,
		FinishedQty:       1,
		Remarks:           "To be rejected",
		CreatedBy:         "operator-1",
		Status:            "OPEN",
	}
	_ = CreateKittingOrder(rejectKit, []models.InventoryKittingLine{
		{
			ProductID:   compKey.ID,
			LocatorID:   loc.ID,
			ConsumedQty: 1,
		},
	})

	err = RejectKittingOrder(rejectKit.ID, "Incorrect parts list")
	if err != nil {
		t.Fatalf("failed to reject kitting: %v", err)
	}

	var savedRejectedKit models.InventoryKitting
	database.DB.First(&savedRejectedKit, "id = ?", rejectKit.ID)
	if savedRejectedKit.Status != "REJECTED" {
		t.Errorf("expected rejected status, got %s", savedRejectedKit.Status)
	}
}
