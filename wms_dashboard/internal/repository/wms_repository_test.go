package repository

import (
	"testing"
	"time"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"

	"github.com/google/uuid"
)

func TestInventoryMovementFIFOAndReservations(t *testing.T) {
	setupTestDB(t)

	// 1. Setup seed data
	prod := &models.Product{
		SKU:  "FIFO-PROD",
		Name: "FIFO Test Product",
	}
	_ = CreateProduct(prod)

	wh := &models.Warehouse{
		Code: "WH-FIFO",
		Name: "FIFO Hub",
	}
	_ = CreateWarehouse(wh)

	loc := &models.Locator{
		ID:          uuid.New().String(),
		WarehouseID: wh.ID,
		Code:        "WH-FIFO-A-1",
		Zone:        "A",
		Aisle:       "1",
		Shelf:       "1",
		Level:       "1",
		IsActive:    true,
	}
	database.DB.Create(loc)

	// Scenario A: Inbound Receipt (Verify stock gets correctly registered under generated batch lot)
	inboundMove := &models.InventoryMovement{
		MovementType: "INBOUND",
		Status:       "OPEN",
		CreatedBy:    "user-1",
		Remarks:      "Inbound receipt test",
	}
	inboundLines := []models.InventoryMovementLine{
		{
			ProductID:         prod.ID,
			ToLocatorID:       loc.ID,
			RequestedQuantity: 20,
		},
	}

	err := CreateInventoryMovement(inboundMove, inboundLines)
	if err != nil {
		t.Fatalf("failed to create inbound movement: %v", err)
	}

	// Verify status is open and line created
	var savedInbound models.InventoryMovement
	database.DB.Preload("Lines").First(&savedInbound, "id = ?", inboundMove.ID)
	if len(savedInbound.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(savedInbound.Lines))
	}

	// Journalize the Inbound Movement
	err = JournalizeInventoryMovement(savedInbound.ID)
	if err != nil {
		t.Fatalf("failed to journalize inbound: %v", err)
	}

	// Verify that Storage record was created with generated batch lot
	var storageLots []models.Storage
	database.DB.Find(&storageLots, "product_id = ?", prod.ID)
	if len(storageLots) != 1 {
		t.Fatalf("expected exactly 1 storage lot to be created, got %d", len(storageLots))
	}
	lot1 := storageLots[0]
	if lot1.QtyOnHand != 20 || lot1.BatchNumber == "" {
		t.Errorf("unexpected storage lot properties: %+v", lot1)
	}

	// Clean up storage lots to set up a precise FIFO scenario
	database.DB.Exec("DELETE FROM storages")

	// Seed two precise lots:
	// Lot A (Oldest): QtyOnHand = 10, ReceivedAt = yesterday
	// Lot B (Newest): QtyOnHand = 15, ReceivedAt = today
	lotA := models.Storage{
		ID:          uuid.New().String(),
		ProductID:   prod.ID,
		LocatorID:   loc.ID,
		BatchNumber: "BAT-A-OLD",
		ReceivedAt:  time.Now().Add(-24 * time.Hour),
		QtyOnHand:   10,
		QtyReserved: 0,
		QtyOnHold:   0,
		UpdatedAt:   time.Now(),
	}
	lotB := models.Storage{
		ID:          uuid.New().String(),
		ProductID:   prod.ID,
		LocatorID:   loc.ID,
		BatchNumber: "BAT-B-NEW",
		ReceivedAt:  time.Now(),
		QtyOnHand:   15,
		QtyReserved: 0,
		QtyOnHold:   0,
		UpdatedAt:   time.Now(),
	}
	database.DB.Create(&lotA)
	database.DB.Create(&lotB)

	// Scenario B: Outbound Allocation (Verify FIFO - oldest batches allocated first)
	// We request 15 items. FIFO should take 10 from lotA (oldest) and 5 from lotB (newest).
	outboundMove := &models.InventoryMovement{
		MovementType: "OUTBOUND",
		Status:       "OPEN",
		CreatedBy:    "user-1",
		Remarks:      "Outbound FIFO test",
	}
	outboundLines := []models.InventoryMovementLine{
		{
			ProductID:         prod.ID,
			RequestedQuantity: 15,
		},
	}

	err = CreateInventoryMovement(outboundMove, outboundLines)
	if err != nil {
		t.Fatalf("failed to create outbound movement: %v", err)
	}

	// Verify reservations on lots
	var updatedLotA, updatedLotB models.Storage
	database.DB.First(&updatedLotA, "id = ?", lotA.ID)
	database.DB.First(&updatedLotB, "id = ?", lotB.ID)

	if updatedLotA.QtyReserved != 10 {
		t.Errorf("expected lotA to be fully reserved (10), got %d", updatedLotA.QtyReserved)
	}
	if updatedLotB.QtyReserved != 5 {
		t.Errorf("expected lotB to be partially reserved (5), got %d", updatedLotB.QtyReserved)
	}

	// Scenario C: Double Booking / Insufficient Stock Prevention
	// Current unreserved stock is: lotA available = 0, lotB available = 10. Total available = 10.
	// Attempting to create an outbound movement for 12 items should fail.
	doubleBookMove := &models.InventoryMovement{
		MovementType: "OUTBOUND",
		Status:       "OPEN",
		CreatedBy:    "user-1",
	}
	doubleBookLines := []models.InventoryMovementLine{
		{
			ProductID:         prod.ID,
			RequestedQuantity: 12,
		},
	}
	err = CreateInventoryMovement(doubleBookMove, doubleBookLines)
	if err == nil {
		t.Error("expected double booking to fail due to insufficient unreserved stock, but it succeeded")
	}

	// Scenario D: Journalizing Outbound (Verify physical deductions are correctly executed)
	err = JournalizeInventoryMovement(outboundMove.ID)
	if err != nil {
		t.Fatalf("failed to journalize outbound: %v", err)
	}

	// Verify physical deductions in DB
	database.DB.First(&updatedLotA, "id = ?", lotA.ID)
	database.DB.First(&updatedLotB, "id = ?", lotB.ID)

	if updatedLotA.QtyOnHand != 0 || updatedLotA.QtyReserved != 0 {
		t.Errorf("expected lotA to be depleted: QtyOnHand=%d, QtyReserved=%d", updatedLotA.QtyOnHand, updatedLotA.QtyReserved)
	}
	if updatedLotB.QtyOnHand != 10 || updatedLotB.QtyReserved != 0 {
		t.Errorf("expected lotB to have 10 left: QtyOnHand=%d, QtyReserved=%d", updatedLotB.QtyOnHand, updatedLotB.QtyReserved)
	}

	// Test Rejecting a Movement (Verify reservation release)
	// Clear all other storage lots first to ensure allocation goes to tempLot
	database.DB.Exec("DELETE FROM storages")

	// Seed a temporary lot with active reservation
	tempLot := models.Storage{
		ID:          uuid.New().String(),
		ProductID:   prod.ID,
		LocatorID:   loc.ID,
		BatchNumber: "BAT-TEMP",
		ReceivedAt:  time.Now(),
		QtyOnHand:   10,
		QtyReserved: 0,
		QtyOnHold:   0,
		UpdatedAt:   time.Now(),
	}
	database.DB.Create(&tempLot)

	cancelMove := &models.InventoryMovement{
		MovementType: "OUTBOUND",
		Status:       "OPEN",
		CreatedBy:    "user-1",
	}
	cancelLines := []models.InventoryMovementLine{
		{
			ProductID:         prod.ID,
			RequestedQuantity: 5,
		},
	}
	_ = CreateInventoryMovement(cancelMove, cancelLines)

	// Verify reservation added
	database.DB.First(&tempLot, "id = ?", tempLot.ID)
	if tempLot.QtyReserved != 5 {
		t.Errorf("expected reservation of 5, got %d", tempLot.QtyReserved)
	}

	// Reject movement
	err = RejectInventoryMovement(cancelMove.ID, "Customer cancelled")
	if err != nil {
		t.Fatalf("failed to reject movement: %v", err)
	}

	// Verify reservation released
	database.DB.First(&tempLot, "id = ?", tempLot.ID)
	if tempLot.QtyReserved != 0 {
		t.Errorf("expected reservation to be released back to 0, got %d", tempLot.QtyReserved)
	}
}

func TestCrossDockLifecycle(t *testing.T) {
	setupTestDB(t)

	prod := &models.Product{
		SKU:  "CD-PROD",
		Name: "Cross Dock Product",
	}
	_ = CreateProduct(prod)

	wh := &models.Warehouse{
		Code: "wh-main-0001",
		Name: "Main WH",
	}
	database.DB.Create(wh)

	// Ensure transit locator exists
	transitLoc := &models.Locator{
		ID:          "loc-crossdock-01",
		WarehouseID: wh.Code,
		Code:        "CD-1-01-1",
		Zone:        "ZONE-CROSSDOCK",
		Aisle:       "CD-1",
		Shelf:       "01",
		Level:       "1",
		IsActive:    true,
	}
	database.DB.Create(transitLoc)

	// 1. Create Cross-Dock Inbound Movement
	move := &models.InventoryMovement{
		MovementType: "INBOUND",
		IsCrossDock:  true,
		Status:       "OPEN",
		CreatedBy:    "user-1",
	}
	lines := []models.InventoryMovementLine{
		{
			ProductID:         prod.ID,
			RequestedQuantity: 20,
		},
	}

	err := CreateInventoryMovement(move, lines)
	if err != nil {
		t.Fatalf("failed to create cross dock movement: %v", err)
	}

	// 2. Claim task
	err = UpdateMovementStatus(move.ID, "IN_PROGRESS")
	if err != nil {
		t.Fatalf("failed to claim task: %v", err)
	}

	// 3. Process Inbound
	err = ProcessCrossDockInbound(move.ID)
	if err != nil {
		t.Fatalf("failed to process cross-dock inbound: %v", err)
	}

	// Verify status is INBOUND
	var checkMove models.InventoryMovement
	database.DB.Preload("Lines").First(&checkMove, "id = ?", move.ID)
	if checkMove.Status != "INBOUND" {
		t.Errorf("expected status INBOUND, got %s", checkMove.Status)
	}

	// Verify storage created in transit locator
	var storage models.Storage
	err = database.DB.First(&storage, "locator_id = ? AND product_id = ?", "loc-crossdock-01", prod.ID).Error
	if err != nil {
		t.Fatalf("failed to find transit storage record: %v", err)
	}
	if storage.QtyOnHand != 20 {
		t.Errorf("expected qty 20 on staging, got %d", storage.QtyOnHand)
	}

	// Verify batch number populated on line
	if checkMove.Lines[0].BatchNumber == "" {
		t.Errorf("expected batch number to be populated on line")
	}

	// 4. Process Shipping (Initiate loading)
	err = ProcessCrossDockShipping(move.ID)
	if err != nil {
		t.Fatalf("failed to process cross-dock shipping: %v", err)
	}

	database.DB.First(&checkMove, "id = ?", move.ID)
	if checkMove.Status != "SHIPPING" {
		t.Errorf("expected status SHIPPING, got %s", checkMove.Status)
	}

	// 5. Process Outbound (Dispatch)
	err = ProcessCrossDockOutbound(move.ID)
	if err != nil {
		t.Fatalf("failed to process cross-dock outbound: %v", err)
	}

	database.DB.First(&checkMove, "id = ?", move.ID)
	if checkMove.Status != "OUTBOUND" {
		t.Errorf("expected status OUTBOUND, got %s", checkMove.Status)
	}

	// Verify storage depleted
	database.DB.First(&storage, "locator_id = ? AND product_id = ?", "loc-crossdock-01", prod.ID)
	if storage.QtyOnHand != 0 {
		t.Errorf("expected staging storage depleted to 0, got %d", storage.QtyOnHand)
	}

	// 6. Complete
	err = UpdateMovementStatus(move.ID, "COMPLETED")
	if err != nil {
		t.Fatalf("failed to complete cross-dock movement: %v", err)
	}

	database.DB.First(&checkMove, "id = ?", move.ID)
	if checkMove.Status != "COMPLETED" {
		t.Errorf("expected status COMPLETED, got %s", checkMove.Status)
	}
}
