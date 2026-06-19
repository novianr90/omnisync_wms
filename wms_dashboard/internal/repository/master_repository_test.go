package repository

import (
	"math"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

func approxEq(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

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
		&models.Account{},
		&models.InventoryLedger{},
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
		{ID: "seq-trf", UsageTable: "inventory_transfers", Prefix: "TRF", FiscalYear: 2026, CurrentNumber: 1, NumberLength: 5},
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

func TestFetchLocatorOccupancies(t *testing.T) {
	setupTestDB(t)

	wh := &models.Warehouse{ID: uuid.New().String(), Code: "WH-OCC", Name: "Occ Hub", IsActive: true}
	database.DB.Create(wh)

	// Locator with no capacity limit (max=0)
	locUnlimited := &models.Locator{
		ID: uuid.New().String(), WarehouseID: wh.ID,
		Code: "WH-OCC-A-1", Zone: "A", Aisle: "1", Shelf: "1", Level: "1", IsActive: true,
	}
	// Locator with capacity (weight=100kg, volume=1m³)
	locCapped := &models.Locator{
		ID: uuid.New().String(), WarehouseID: wh.ID,
		Code: "WH-OCC-A-2", Zone: "A", Aisle: "1", Shelf: "1", Level: "2", IsActive: true,
		MaxWeight: 100, MaxVolume: 1.0,
	}
	database.DB.Create(locUnlimited)
	database.DB.Create(locCapped)

	prod := &models.Product{ID: uuid.New().String(), SKU: "OCC-PROD", Name: "Heavy Item", UnitWeight: 10, UnitVolume: 0.1}
	database.DB.Create(prod)

	// Put 6 units in the capped locator → 60kg / 0.6m³ → 60% util (Amber band)
	storage := &models.Storage{
		ID: uuid.New().String(), ProductID: prod.ID, LocatorID: locCapped.ID,
		BatchNumber: "BATCH-1", QtyOnHand: 6,
	}
	database.DB.Create(storage)

	occs, err := FetchLocatorOccupancies()
	if err != nil {
		t.Fatalf("FetchLocatorOccupancies error: %v", err)
	}
	if len(occs) < 2 {
		t.Fatalf("expected at least 2 occupancy rows, got %d", len(occs))
	}

	occMap := make(map[string]LocatorOccupancy)
	for _, o := range occs {
		occMap[o.LocatorID] = o
	}

	// Unlimited locator: UtilPct must be 0 regardless of stock
	u := occMap[locUnlimited.ID]
	if u.UtilPct != 0 {
		t.Errorf("unlimited locator UtilPct should be 0, got %f", u.UtilPct)
	}
	if u.ColorBand() != "green" {
		t.Errorf("unlimited locator ColorBand should be green, got %s", u.ColorBand())
	}

	// Capped locator: 6*10=60kg / 100 = 60% → Amber
	c := occMap[locCapped.ID]
	if !approxEq(c.CurrentWeight, 60.0, 0.001) {
		t.Errorf("expected CurrentWeight ~60, got %f", c.CurrentWeight)
	}
	if !approxEq(c.CurrentVolume, 0.6, 0.001) {
		t.Errorf("expected CurrentVolume ~0.6, got %f", c.CurrentVolume)
	}
	if !approxEq(c.UtilPct, 60.0, 0.001) {
		t.Errorf("expected UtilPct ~60, got %f", c.UtilPct)
	}
	if c.ColorBand() != "amber" {
		t.Errorf("expected amber color band, got %s", c.ColorBand())
	}
}

func TestLocatorOccupancy_ColorBands(t *testing.T) {
	cases := []struct {
		util float64
		want string
	}{
		{0, "green"},
		{49.9, "green"},
		{50, "amber"},
		{89.9, "amber"},
		{90, "red"},
		{100, "red"},
	}
	for _, tc := range cases {
		o := LocatorOccupancy{UtilPct: tc.util, MaxWeight: 100}
		if got := o.ColorBand(); got != tc.want {
			t.Errorf("UtilPct %.1f: expected %s, got %s", tc.util, tc.want, got)
		}
	}
}

func TestLocatorOccupancy_MaxOfWeightAndVolume(t *testing.T) {
	setupTestDB(t)

	wh := &models.Warehouse{ID: uuid.New().String(), Code: "WH-MAX", Name: "Max Hub", IsActive: true}
	database.DB.Create(wh)

	// Locator: weight limit 200kg, volume limit 1m³
	loc := &models.Locator{
		ID: uuid.New().String(), WarehouseID: wh.ID,
		Code: "WH-MAX-A-1", Zone: "A", Aisle: "1", Shelf: "1", Level: "1", IsActive: true,
		MaxWeight: 200, MaxVolume: 1.0,
	}
	database.DB.Create(loc)

	// Product: light but bulky (low weight, high volume)
	prod := &models.Product{ID: uuid.New().String(), SKU: "BULKY", Name: "Bulky Item", UnitWeight: 1, UnitVolume: 0.2}
	database.DB.Create(prod)

	// 5 units → 5kg (2.5% weight util) / 1.0m³ (100% volume util) → should pick volume = 100%
	storage := &models.Storage{
		ID: uuid.New().String(), ProductID: prod.ID, LocatorID: loc.ID,
		BatchNumber: "BATCH-BULKY", QtyOnHand: 5,
	}
	database.DB.Create(storage)

	occs, err := FetchLocatorOccupancies()
	if err != nil {
		t.Fatalf("FetchLocatorOccupancies error: %v", err)
	}
	var found LocatorOccupancy
	for _, o := range occs {
		if o.LocatorID == loc.ID {
			found = o
		}
	}
	if found.UtilPct != 100.0 {
		t.Errorf("expected UtilPct 100 (volume-dominated), got %f", found.UtilPct)
	}
	if found.ColorBand() != "red" {
		t.Errorf("expected red band, got %s", found.ColorBand())
	}
}

func TestFetchLocatorOccupancies_PendingInbound(t *testing.T) {
	setupTestDB(t)

	wh := &models.Warehouse{ID: uuid.New().String(), Code: "WH-PEND", Name: "Pending Hub", IsActive: true}
	database.DB.Create(wh)

	loc := &models.Locator{
		ID: uuid.New().String(), WarehouseID: wh.ID,
		Code: "WH-PEND-A-1", Zone: "A", Aisle: "1", Shelf: "1", Level: "1", IsActive: true,
		MaxWeight: 100, MaxVolume: 1.0,
	}
	database.DB.Create(loc)

	prod := &models.Product{ID: uuid.New().String(), SKU: "PEND-PROD", Name: "Pending Item", UnitWeight: 20, UnitVolume: 0.2}
	database.DB.Create(prod)

	// No stock on hand — locator is empty
	// But there is an open INBOUND movement for 3 units → 60kg pending / 0.6m³ pending → 60% util
	mvt := &models.InventoryMovement{
		ID:           uuid.New().String(),
		DocumentNo:   "MOV-PEND-001",
		MovementType: "INBOUND",
		Status:       "OPEN",
		CreatedBy:    "test",
	}
	database.DB.Create(mvt)

	line := &models.InventoryMovementLine{
		ID:                uuid.New().String(),
		MovementID:        mvt.ID,
		ProductID:         prod.ID,
		ToLocatorID:       loc.ID,
		RequestedQuantity: 3,
	}
	database.DB.Create(line)

	occs, err := FetchLocatorOccupancies()
	if err != nil {
		t.Fatalf("FetchLocatorOccupancies error: %v", err)
	}

	var found LocatorOccupancy
	for _, o := range occs {
		if o.LocatorID == loc.ID {
			found = o
		}
	}

	// Confirmed stock = 0; pending = 3 × 20 = 60kg → 60% util
	if found.CurrentWeight != 0 {
		t.Errorf("expected CurrentWeight 0 (no stock), got %f", found.CurrentWeight)
	}
	if !approxEq(found.PendingWeight, 60.0, 0.001) {
		t.Errorf("expected PendingWeight ~60, got %f", found.PendingWeight)
	}
	if !approxEq(found.UtilPct, 60.0, 0.001) {
		t.Errorf("expected UtilPct ~60 (from pending), got %f", found.UtilPct)
	}
	if found.ColorBand() != "amber" {
		t.Errorf("expected amber band, got %s", found.ColorBand())
	}
	if !found.HasPending() {
		t.Errorf("expected HasPending() = true")
	}

	// Reject the movement → pending should drop to 0
	mvt.Status = "REJECTED"
	database.DB.Save(mvt)

	occs2, _ := FetchLocatorOccupancies()
	for _, o := range occs2 {
		if o.LocatorID == loc.ID {
			if o.PendingWeight != 0 {
				t.Errorf("after rejection, expected PendingWeight 0, got %f", o.PendingWeight)
			}
			if o.UtilPct != 0 {
				t.Errorf("after rejection, expected UtilPct 0, got %f", o.UtilPct)
			}
		}
	}
}
