package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

func TestRTVMovements(t *testing.T) {
	setupTestDB(t)

	// 1. Setup seed data
	prod := &models.Product{
		SKU:  "RTV-SKU-01",
		Name: "RTV Test Product",
	}
	_ = CreateProduct(prod)

	wh := &models.Warehouse{
		Code: "WH-RTV",
		Name: "RTV Warehouse",
	}
	_ = CreateWarehouse(wh)

	loc := &models.Locator{
		ID:          uuid.New().String(),
		WarehouseID: wh.ID,
		Code:        "WH-RTV-A-1",
		Zone:        "A",
		Aisle:       "1",
		Shelf:       "1",
		Level:       "1",
		IsActive:    true,
	}
	database.DB.Create(loc)

	// Seed standard storage lot
	storageRegular := models.Storage{
		ID:          uuid.New().String(),
		ProductID:   prod.ID,
		LocatorID:   loc.ID,
		BatchNumber: "BATCH-REG",
		ReceivedAt:  time.Now().Add(-2 * time.Hour),
		QtyOnHand:   50,
		QtyReserved: 0,
		QtyOnHold:   0,
		UpdatedAt:   time.Now(),
	}
	database.DB.Create(&storageRegular)

	// Seed QC Hold storage lot
	storageHold := models.Storage{
		ID:          uuid.New().String(),
		ProductID:   prod.ID,
		LocatorID:   loc.ID,
		BatchNumber: "BATCH-HOLD",
		ReceivedAt:  time.Now().Add(-1 * time.Hour),
		QtyOnHand:   30,
		QtyReserved: 0,
		QtyOnHold:   30,
		UpdatedAt:   time.Now(),
	}
	database.DB.Create(&storageHold)

	// Seed QCHold records
	qcHold1 := models.QCHold{
		ID:         uuid.New().String(),
		DocumentNo: "QCH-001",
		StorageID:  storageHold.ID,
		Qty:        10,
		Reason:     "DAMAGED",
		Status:     "ACTIVE",
		CreatedBy:  "operator-1",
		CreatedAt:  time.Now(),
	}
	qcHold2 := models.QCHold{
		ID:         uuid.New().String(),
		DocumentNo: "QCH-002",
		StorageID:  storageHold.ID,
		Qty:        20,
		Reason:     "INVESTIGATION",
		Status:     "ACTIVE",
		CreatedBy:  "operator-1",
		CreatedAt:  time.Now(),
	}
	database.DB.Create(&qcHold1)
	database.DB.Create(&qcHold2)

	// --- TEST SCENARIO 1: RTV from Regular Stock (FIFO Reservation & Journaling) ---
	rtvRegularMove := &models.InventoryMovement{
		MovementType: "RTV",
		Status:       "OPEN",
		CreatedBy:    "operator-1",
		Remarks:      "RTV Regular stock test",
	}
	rtvRegularLines := []models.InventoryMovementLine{
		{
			ProductID:         prod.ID,
			RequestedQuantity: 20,
			IsFromHold:        false,
		},
	}

	// Create should reserve 20 regular items (from FIFO regular storage, not hold)
	err := CreateInventoryMovement(rtvRegularMove, rtvRegularLines)
	if err != nil {
		t.Fatalf("failed to create regular RTV: %v", err)
	}

	// Verify reservations
	var regLot models.Storage
	database.DB.First(&regLot, "id = ?", storageRegular.ID)
	if regLot.QtyReserved != 20 {
		t.Errorf("expected 20 reserved, got %d", regLot.QtyReserved)
	}

	// Journalize regular RTV
	err = JournalizeInventoryMovement(rtvRegularMove.ID)
	if err != nil {
		t.Fatalf("failed to journalize regular RTV: %v", err)
	}

	// Verify regular stock deduction
	database.DB.First(&regLot, "id = ?", storageRegular.ID)
	if regLot.QtyOnHand != 30 || regLot.QtyReserved != 0 {
		t.Errorf("unexpected regular stock state: OnHand=%d, Reserved=%d", regLot.QtyOnHand, regLot.QtyReserved)
	}

	// --- TEST SCENARIO 2: RTV from Regular Stock (Specific Lot Reservation & Rejection) ---
	rtvRejectRegularMove := &models.InventoryMovement{
		MovementType: "RTV",
		Status:       "OPEN",
		CreatedBy:    "operator-1",
		Remarks:      "RTV Regular stock rejection test",
	}
	rtvRejectRegularLines := []models.InventoryMovementLine{
		{
			ProductID:         prod.ID,
			FromLocatorID:     loc.ID,
			BatchNumber:       "BATCH-REG",
			RequestedQuantity: 10,
			IsFromHold:        false,
		},
	}

	err = CreateInventoryMovement(rtvRejectRegularMove, rtvRejectRegularLines)
	if err != nil {
		t.Fatalf("failed to create reject-test regular RTV: %v", err)
	}

	// Verify reservation
	database.DB.First(&regLot, "id = ?", storageRegular.ID)
	if regLot.QtyReserved != 10 {
		t.Errorf("expected 10 reserved, got %d", regLot.QtyReserved)
	}

	// Reject regular RTV
	err = RejectInventoryMovement(rtvRejectRegularMove.ID, "Cancelled by vendor")
	if err != nil {
		t.Fatalf("failed to reject regular RTV: %v", err)
	}

	// Verify reservation release
	database.DB.First(&regLot, "id = ?", storageRegular.ID)
	if regLot.QtyReserved != 0 {
		t.Errorf("expected reservation to be released (0), got %d", regLot.QtyReserved)
	}

	// --- TEST SCENARIO 3: RTV from QC Hold Stock (Deduction & QC Hold Release) ---
	rtvHoldMove := &models.InventoryMovement{
		MovementType: "RTV",
		Status:       "OPEN",
		CreatedBy:    "operator-1",
		Remarks:      "RTV QC Hold stock test",
	}
	rtvHoldLines := []models.InventoryMovementLine{
		{
			ProductID:         prod.ID,
			FromLocatorID:     loc.ID,
			BatchNumber:       "BATCH-HOLD",
			RequestedQuantity: 25,
			IsFromHold:        true,
		},
	}

	// Create should validate hold stock but not add reservation
	err = CreateInventoryMovement(rtvHoldMove, rtvHoldLines)
	if err != nil {
		t.Fatalf("failed to create hold RTV: %v", err)
	}

	// Verify no reservation added on hold lot
	var holdLot models.Storage
	database.DB.First(&holdLot, "id = ?", storageHold.ID)
	if holdLot.QtyReserved != 0 {
		t.Errorf("expected 0 reserved, got %d", holdLot.QtyReserved)
	}

	// Journalize hold RTV
	err = JournalizeInventoryMovement(rtvHoldMove.ID)
	if err != nil {
		t.Fatalf("failed to journalize hold RTV: %v", err)
	}

	// Verify physical and hold stock deductions
	database.DB.First(&holdLot, "id = ?", storageHold.ID)
	if holdLot.QtyOnHand != 5 || holdLot.QtyOnHold != 5 {
		t.Errorf("unexpected hold stock state: OnHand=%d, OnHold=%d", holdLot.QtyOnHand, holdLot.QtyOnHold)
	}

	// Verify QCHold records updates:
	// We returned 25 from QC Hold.
	// qcHold1 had 10 (should be RELEASED, Qty=0).
	// qcHold2 had 20 (should be ACTIVE, Qty=5, consumed 15).
	var savedHold1, savedHold2 models.QCHold
	database.DB.First(&savedHold1, "id = ?", qcHold1.ID)
	database.DB.First(&savedHold2, "id = ?", qcHold2.ID)

	if savedHold1.Status != "RELEASED" || savedHold1.Qty != 0 {
		t.Errorf("expected hold 1 to be released: Status=%s, Qty=%d", savedHold1.Status, savedHold1.Qty)
	}
	if savedHold2.Status != "ACTIVE" || savedHold2.Qty != 5 {
		t.Errorf("expected hold 2 to be active with 5 remaining: Status=%s, Qty=%d", savedHold2.Status, savedHold2.Qty)
	}

	// --- TEST SCENARIO 4: RTV from QC Hold Stock (Reject does nothing to reservations) ---
	rtvRejectHoldMove := &models.InventoryMovement{
		MovementType: "RTV",
		Status:       "OPEN",
		CreatedBy:    "operator-1",
		Remarks:      "RTV QC Hold stock reject test",
	}
	rtvRejectHoldLines := []models.InventoryMovementLine{
		{
			ProductID:         prod.ID,
			FromLocatorID:     loc.ID,
			BatchNumber:       "BATCH-HOLD",
			RequestedQuantity: 5,
			IsFromHold:        true,
		},
	}

	err = CreateInventoryMovement(rtvRejectHoldMove, rtvRejectHoldLines)
	if err != nil {
		t.Fatalf("failed to create reject-test hold RTV: %v", err)
	}

	// Reject hold RTV
	err = RejectInventoryMovement(rtvRejectHoldMove.ID, "Cancelled")
	if err != nil {
		t.Fatalf("failed to reject hold RTV: %v", err)
	}

	// Verify no reservation changes
	database.DB.First(&holdLot, "id = ?", storageHold.ID)
	if holdLot.QtyReserved != 0 {
		t.Errorf("expected reserved to remain 0, got %d", holdLot.QtyReserved)
	}
}
