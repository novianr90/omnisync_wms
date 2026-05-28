package repository

import (
	"testing"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"

	"github.com/google/uuid"
)

func TestQCHoldScenarios(t *testing.T) {
	setupTestDB(t)

	// 1. Setup seed data
	prod := &models.Product{
		SKU:  "QC-PROD-1",
		Name: "QC Test Item",
	}
	if err := CreateProduct(prod); err != nil {
		t.Fatalf("failed to create seed product: %v", err)
	}

	wh := &models.Warehouse{
		Code: "WH-QC",
		Name: "QC Warehouse",
	}
	if err := CreateWarehouse(wh); err != nil {
		t.Fatalf("failed to create seed warehouse: %v", err)
	}

	loc := &models.Locator{
		ID:          uuid.New().String(),
		WarehouseID: wh.ID,
		Code:        "WH-QC-Z1",
		Zone:        "Z",
		Aisle:       "1",
		Shelf:       "1",
		Level:       "1",
		IsActive:    true,
	}
	if err := database.DB.Create(loc).Error; err != nil {
		t.Fatalf("failed to create seed locator: %v", err)
	}

	storage := &models.Storage{
		ID:          uuid.New().String(),
		ProductID:   prod.ID,
		LocatorID:   loc.ID,
		BatchNumber: "BAT-0001",
		QtyOnHand:   10,
		QtyReserved: 2,
		QtyOnHold:   0,
	}
	if err := database.DB.Create(storage).Error; err != nil {
		t.Fatalf("failed to create seed storage lot: %v", err)
	}

	// Scenario A: Successful Freezing of Stock
	err := CreateQCHold(storage.ID, 5, "DAMAGED", "Scratch on surface", "admin-user")
	if err != nil {
		t.Fatalf("unexpected error during CreateQCHold: %v", err)
	}

	// Assert storage lot values
	var updatedStorage models.Storage
	database.DB.First(&updatedStorage, "id = ?", storage.ID)
	if updatedStorage.QtyOnHold != 5 {
		t.Errorf("expected QtyOnHold to be 5, got %d", updatedStorage.QtyOnHold)
	}
	available := updatedStorage.QtyOnHand - updatedStorage.QtyReserved - updatedStorage.QtyOnHold // 10 - 2 - 5 = 3
	if available != 3 {
		t.Errorf("expected available stock to be 3, got %d", available)
	}

	// Verify QCHold record was created
	var hold models.QCHold
	err = database.DB.First(&hold, "storage_id = ?", storage.ID).Error
	if err != nil {
		t.Fatalf("failed to find QCHold record in DB: %v", err)
	}
	if hold.Qty != 5 || hold.Status != "ACTIVE" || hold.Reason != "DAMAGED" || hold.CreatedBy != "admin-user" {
		t.Errorf("unexpected values in QCHold record: %+v", hold)
	}

	// Scenario B: Over-freezing should fail (requesting 4 when only 3 available)
	err = CreateQCHold(storage.ID, 4, "INVESTIGATION", "Over hold test", "admin-user")
	if err == nil {
		t.Error("expected over-freezing to return error, but got nil")
	}

	// Scenario C: Releasing Stock
	err = ReleaseQCHold(hold.ID, "manager-user")
	if err != nil {
		t.Fatalf("unexpected error during ReleaseQCHold: %v", err)
	}

	// Assert storage lot QtyOnHold is decremented back
	database.DB.First(&updatedStorage, "id = ?", storage.ID)
	if updatedStorage.QtyOnHold != 0 {
		t.Errorf("expected QtyOnHold to return to 0, got %d", updatedStorage.QtyOnHold)
	}

	// Verify QCHold status is RELEASED
	var releasedHold models.QCHold
	database.DB.First(&releasedHold, "id = ?", hold.ID)
	if releasedHold.Status != "RELEASED" || releasedHold.ReleasedBy != "manager-user" || releasedHold.ReleasedAt == nil {
		t.Errorf("unexpected values in released QCHold record: %+v", releasedHold)
	}

	// Scenario D: Fetching QCHolds & Storages with stock
	holds, err := FetchQCHolds()
	if err != nil {
		t.Fatalf("FetchQCHolds error: %v", err)
	}
	if len(holds) != 1 {
		t.Errorf("expected 1 hold in list, got %d", len(holds))
	}

	storages, err := FetchStoragesWithAvailableStock()
	if err != nil {
		t.Fatalf("FetchStoragesWithAvailableStock error: %v", err)
	}
	// The lot has 10 total on hand, 2 reserved, 0 on hold -> 8 available, so it should be in this list
	found := false
	for _, s := range storages {
		if s.ID == storage.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected storage lot %s to be listed in available stock, but it wasn't", storage.ID)
	}
}
