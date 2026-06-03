package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

func TestFetchInProgressDocs(t *testing.T) {
	setupTestDB(t)

	// Create view manually in sqlite test database
	viewSql := `
CREATE VIEW IF NOT EXISTS in_progress_documents AS
SELECT id, document_no, 'Movement (' || movement_type || ')' AS doc_type, created_at, status, '/' AS link FROM inventory_movements WHERE status NOT IN ('COMPLETED', 'REJECTED')
UNION ALL
SELECT id, document_no, 'QC Hold' AS doc_type, created_at, status, '/wms/qc-holds' AS link FROM qc_holds WHERE status = 'ACTIVE'
UNION ALL
SELECT id, document_no, 'Adjustment' AS doc_type, created_at, status, '/wms/adjustments' AS link FROM inventory_adjustments WHERE status = 'OPEN'
UNION ALL
SELECT id, document_no, 'Kitting' AS doc_type, created_at, status, '/wms/kitting' AS link FROM inventory_kittings WHERE status = 'OPEN';
`
	if err := database.DB.Exec(viewSql).Error; err != nil {
		t.Fatalf("failed to create sqlite test view: %v", err)
	}

	// Create a dummy product for kitting reference (finished product)
	prod := &models.Product{
		ID:   uuid.New().String(),
		SKU:  "KIT-REF-PROD",
		Name: "Kit Reference Product",
	}
	if err := database.DB.Create(prod).Error; err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	// Create a dummy locator for kitting reference
	wh := &models.Warehouse{
		ID:   uuid.New().String(),
		Code: "WH-KIT",
		Name: "Kit Warehouse",
	}
	database.DB.Create(wh)
	loc := &models.Locator{
		ID:          uuid.New().String(),
		WarehouseID: wh.ID,
		Zone:        "A",
		Aisle:       "1",
		Shelf:       "1",
		Level:       "1",
		Code:        "WH-KIT-A-1-1-1",
	}
	database.DB.Create(loc)

	now := time.Now()

	// 1. Seed Inventory Movements
	movActive := models.InventoryMovement{
		ID:           uuid.New().String(),
		DocumentNo:   "MOV-ACTIVE",
		MovementType: "INBOUND",
		Status:       "IN_PROGRESS",
		CreatedAt:    now.Add(-4 * time.Hour),
		CreatedBy:    "user-1",
	}
	movCompleted := models.InventoryMovement{
		ID:           uuid.New().String(),
		DocumentNo:   "MOV-COMPLETED",
		MovementType: "INBOUND",
		Status:       "COMPLETED",
		CreatedAt:    now.Add(-5 * time.Hour),
		CreatedBy:    "user-1",
	}
	database.DB.Create(&movActive)
	database.DB.Create(&movCompleted)

	// 2. Seed QC Holds
	holdActive := models.QCHold{
		ID:         uuid.New().String(),
		DocumentNo: "QCH-ACTIVE",
		StorageID:  uuid.New().String(), // dummy
		Qty:        10,
		Reason:     "INVESTIGATION",
		Status:     "ACTIVE",
		CreatedAt:  now.Add(-3 * time.Hour),
		CreatedBy:  "user-1",
	}
	holdReleased := models.QCHold{
		ID:         uuid.New().String(),
		DocumentNo: "QCH-RELEASED",
		StorageID:  uuid.New().String(), // dummy
		Qty:        10,
		Reason:     "INVESTIGATION",
		Status:     "RELEASED",
		CreatedAt:  now.Add(-6 * time.Hour),
		CreatedBy:  "user-1",
	}
	database.DB.Create(&holdActive)
	database.DB.Create(&holdReleased)

	// 3. Seed Inventory Adjustments
	adjActive := models.InventoryAdjustment{
		ID:         uuid.New().String(),
		DocumentNo: "ADJ-ACTIVE",
		Status:     "OPEN",
		ReasonCode: "FOUND",
		CreatedAt:  now.Add(-2 * time.Hour),
		CreatedBy:  "user-1",
	}
	adjCompleted := models.InventoryAdjustment{
		ID:         uuid.New().String(),
		DocumentNo: "ADJ-COMPLETED",
		Status:     "JOURNALED",
		ReasonCode: "FOUND",
		CreatedAt:  now.Add(-7 * time.Hour),
		CreatedBy:  "user-1",
	}
	database.DB.Create(&adjActive)
	database.DB.Create(&adjCompleted)

	// 4. Seed Inventory Kittings
	kitActive := models.InventoryKitting{
		ID:                uuid.New().String(),
		DocumentNo:        "KIT-ACTIVE",
		Status:            "OPEN",
		FinishedProductID: prod.ID,
		FinishedLocatorID: loc.ID,
		FinishedQty:       1,
		CreatedAt:         now.Add(-1 * time.Hour),
		CreatedBy:         "user-1",
	}
	kitCompleted := models.InventoryKitting{
		ID:                uuid.New().String(),
		DocumentNo:        "KIT-COMPLETED",
		Status:            "REJECTED",
		FinishedProductID: prod.ID,
		FinishedLocatorID: loc.ID,
		FinishedQty:       1,
		CreatedAt:         now.Add(-8 * time.Hour),
		CreatedBy:         "user-1",
	}
	database.DB.Create(&kitActive)
	database.DB.Create(&kitCompleted)

	// Test 1: Fetch All (should return exactly 4 active docs in chronological order descending)
	docs, err := FetchInProgressDocs("All", "")
	if err != nil {
		t.Fatalf("FetchInProgressDocs error: %v", err)
	}
	if len(docs) != 4 {
		t.Errorf("Expected 4 active documents, got %d", len(docs))
	}

	// Verify chronological descending order: KIT-ACTIVE (now-1h), ADJ-ACTIVE (now-2h), QCH-ACTIVE (now-3h), MOV-ACTIVE (now-4h)
	expectedOrder := []string{"KIT-ACTIVE", "ADJ-ACTIVE", "QCH-ACTIVE", "MOV-ACTIVE"}
	for i, expected := range expectedOrder {
		if docs[i].DocumentNo != expected {
			t.Errorf("Expected order index %d to be %s, got %s", i, expected, docs[i].DocumentNo)
		}
	}

	// Test 2: Filter by Document Type "Movement"
	movDocs, err := FetchInProgressDocs("Movement", "")
	if err != nil {
		t.Fatalf("FetchInProgressDocs type movement error: %v", err)
	}
	if len(movDocs) != 1 || movDocs[0].DocumentNo != "MOV-ACTIVE" {
		t.Errorf("Expected only 1 Movement document (MOV-ACTIVE), got %d: %v", len(movDocs), movDocs)
	}

	// Test 3: Filter by search term "QCH"
	searchDocs, err := FetchInProgressDocs("All", "QCH-ACTIVE")
	if err != nil {
		t.Fatalf("FetchInProgressDocs search error: %v", err)
	}
	if len(searchDocs) != 1 || searchDocs[0].DocumentNo != "QCH-ACTIVE" {
		t.Errorf("Expected search to return QCH-ACTIVE, got: %v", searchDocs)
	}
}
