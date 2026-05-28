package repository

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

func setupTestDB(t *testing.T) {
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
		&models.Storage{},
		&models.InventoryMovement{},
		&models.InventoryMovementLine{},
		&models.QCHold{},
		&models.SequenceGenerator{},
		&models.InventoryKitting{},
		&models.InventoryKittingLine{},
		&models.InventoryAdjustment{},
		&models.InventoryAdjustmentLine{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	// Seed default sequences for testing
	db.Create([]models.SequenceGenerator{
		{ID: "seq-mov", UsageTable: "inventory_movements", Prefix: "MOV", FiscalYear: 2026, CurrentNumber: 1, NumberLength: 5},
		{ID: "seq-adj", UsageTable: "inventory_adjustments", Prefix: "ADJ", FiscalYear: 2026, CurrentNumber: 1, NumberLength: 5},
		{ID: "seq-kit", UsageTable: "inventory_kittings", Prefix: "KIT", FiscalYear: 2026, CurrentNumber: 1, NumberLength: 5},
		{ID: "seq-qch", UsageTable: "qc_holds", Prefix: "QCH", FiscalYear: 2026, CurrentNumber: 1, NumberLength: 5},
		{ID: "seq-stor", UsageTable: "storages", Prefix: "BAT", FiscalYear: 2026, CurrentNumber: 1, NumberLength: 6},
	})

	database.DB = db
}

func TestProductCRUDAndSafeguards(t *testing.T) {
	setupTestDB(t)

	// 1. Create a product
	p := &models.Product{
		SKU:         "TEST-SKU-1",
		Name:        "Test Keyboard",
		Description: "Mechanical key",
		Category:    "Hardware",
		Price:       59.99,
	}
	if err := CreateProduct(p); err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	// 2. Fetch by ID
	fetched, err := FetchProductByID(p.ID)
	if err != nil {
		t.Fatalf("failed to fetch product: %v", err)
	}
	if fetched.SKU != p.SKU {
		t.Errorf("expected SKU %s, got %s", p.SKU, fetched.SKU)
	}

	// 3. Update product
	p.Price = 69.99
	if err := UpdateProduct(p); err != nil {
		t.Fatalf("failed to update product: %v", err)
	}
	fetched, _ = FetchProductByID(p.ID)
	if fetched.Price != 69.99 {
		t.Errorf("expected updated price 69.99, got %f", fetched.Price)
	}

	// 4. Test Soft Delete
	if err := DeleteProduct(p.ID); err != nil {
		t.Fatalf("failed to delete product: %v", err)
	}
	// Verify it can't be fetched normally
	_, err = FetchProductByID(p.ID)
	if err == nil {
		t.Fatal("expected error fetching deleted product, got nil")
	}
	// Verify unscoped find still sees it (Soft Delete verification!)
	var count int64
	database.DB.Unscoped().Model(&models.Product{}).Where("id = ?", p.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected unscoped count 1 (soft deleted), got %d", count)
	}

	// 5. Recreate and test Safeguard A (Active physical stock exists)
	p2 := &models.Product{SKU: "TEST-SKU-2", Name: "Stocked Product"}
	_ = CreateProduct(p2)

	wh := &models.Warehouse{
		ID:       uuid.New().String(),
		Code:     "WH-TEST",
		Name:     "Test Hub",
		IsActive: true,
	}
	database.DB.Create(wh)

	loc := &models.Locator{
		ID:          uuid.New().String(),
		WarehouseID: wh.ID,
		Code:        "WH-TEST-A-1",
		Zone:        "A",
		Aisle:       "1",
		Shelf:       "1",
		Level:       "1",
		IsActive:    true,
	}
	database.DB.Create(loc)

	storage := &models.Storage{
		ID:        uuid.New().String(),
		ProductID: p2.ID,
		LocatorID: loc.ID,
		QtyOnHand: 10,
	}
	database.DB.Create(storage)

	err = DeleteProduct(p2.ID)
	if err == nil || err.Error() != "cannot delete product: physical stock exists on warehouse shelves" {
		t.Errorf("expected deletion to fail due to physical stock, got: %v", err)
	}

	// 6. Test Safeguard B (Historical movement referenced)
	storage.QtyOnHand = 0
	database.DB.Save(storage)

	line := &models.InventoryMovementLine{
		ID:        uuid.New().String(),
		ProductID: p2.ID,
	}
	database.DB.Create(line)

	err = DeleteProduct(p2.ID)
	if err == nil || err.Error() != "cannot delete product: referenced in historical movement documents" {
		t.Errorf("expected deletion to fail due to historical movements, got: %v", err)
	}
}

func TestWarehouseAndLocatorSafeguards(t *testing.T) {
	setupTestDB(t)

	wh := &models.Warehouse{Code: "WH-TEST", Name: "Test Hub"}
	_ = CreateWarehouse(wh)

	loc := &models.Locator{
		ID:          uuid.New().String(),
		WarehouseID: wh.ID,
		Code:        "WH-TEST-A-1",
		Zone:        "A",
		Aisle:       "1",
		Shelf:       "1",
		Level:       "1",
		IsActive:    true,
	}
	database.DB.Create(loc)

	// Test Warehouse Deletion Safeguard
	err := DeleteWarehouse(wh.ID)
	if err == nil || err.Error() != "cannot delete warehouse: active shelf locators exist inside it" {
		t.Errorf("expected warehouse deletion to fail due to active child locators, got: %v", err)
	}

	// Test Locator Deletion Safeguard
	p := &models.Product{SKU: "SKU-3", Name: "P3"}
	_ = CreateProduct(p)

	storage := &models.Storage{
		ID:        uuid.New().String(),
		ProductID: p.ID,
		LocatorID: loc.ID,
		QtyOnHand: 5,
	}
	database.DB.Create(storage)

	err = DeleteLocator(loc.ID)
	if err == nil || err.Error() != "cannot delete locator: active physical stock exists on this shelf" {
		t.Errorf("expected locator deletion to fail due to active stock, got: %v", err)
	}
}

func TestUoMCRUDAndSafeguards(t *testing.T) {
	setupTestDB(t)

	// 1. Create UoM
	u := &models.UoM{
		Code:        "kg",
		Name:        "Kilogram",
		Description: "Weight unit",
	}
	if err := CreateUoM(u); err != nil {
		t.Fatalf("failed to create uom: %v", err)
	}

	// 2. Fetch by ID
	fetched, err := FetchUoMByID(u.ID)
	if err != nil {
		t.Fatalf("failed to fetch uom: %v", err)
	}
	if fetched.Code != u.Code {
		t.Errorf("expected code %s, got %s", u.Code, fetched.Code)
	}

	// 3. Update UoM
	u.Name = "Kilo"
	if err := UpdateUoM(u); err != nil {
		t.Fatalf("failed to update uom: %v", err)
	}
	fetched, _ = FetchUoMByID(u.ID)
	if fetched.Name != "Kilo" {
		t.Errorf("expected updated name Kilo, got %s", fetched.Name)
	}

	// 4. Test Safeguard A: Cannot delete UoM if referenced by active product
	p := &models.Product{
		SKU:   "PROD-1",
		Name:  "Test Product",
		UoMID: u.ID,
	}
	_ = CreateProduct(p)

	err = DeleteUoM(u.ID)
	if err == nil || err.Error() != "cannot delete unit of measure: referenced by active products in the master catalog" {
		t.Errorf("expected uom deletion to fail due to product reference, got: %v", err)
	}

	// Remove product reference
	p.UoMID = ""
	_ = UpdateProduct(p)

	// 5. Test Safeguard B: Cannot delete UoM if referenced in conversions
	u2 := &models.UoM{Code: "pack", Name: "Pack"}
	_ = CreateUoM(u2)

	conv := &models.UoMConversion{
		FromUoMID:      u2.ID,
		ToUoMID:        u.ID,
		MultiplyFactor: 1.0,
	}
	_ = CreateConversion(conv)

	err = DeleteUoM(u.ID)
	if err == nil || err.Error() != "cannot delete unit of measure: referenced in active conversion rules" {
		t.Errorf("expected uom deletion to fail due to conversion reference, got: %v", err)
	}

	// Remove conversion
	_ = DeleteConversion(conv.ID)

	// 6. Test successful Soft Delete
	if err := DeleteUoM(u.ID); err != nil {
		t.Fatalf("failed to delete uom: %v", err)
	}
	// Verify unscoped soft delete
	var count int64
	database.DB.Unscoped().Model(&models.UoM{}).Where("id = ?", u.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected unscoped count 1 (soft deleted uom), got %d", count)
	}
}
