package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

// seedValuationFixtures creates minimal but realistic fixtures and returns references
// to the created entities so tests can assert against them.
func seedValuationFixtures(t *testing.T) (prod1, prod2 *models.Product, wh *models.Warehouse, loc *models.Locator) {
	t.Helper()

	wh = &models.Warehouse{
		ID:       uuid.New().String(),
		Code:     "WH-VAL",
		Name:     "Valuation Warehouse",
		IsActive: true,
	}
	if err := database.DB.Create(wh).Error; err != nil {
		t.Fatalf("seed warehouse: %v", err)
	}

	loc = &models.Locator{
		ID:          uuid.New().String(),
		WarehouseID: wh.ID,
		Code:        "WH-VAL-A-1",
		Zone:        "A",
		Aisle:       "1",
		Shelf:       "1",
		Level:       "1",
		IsActive:    true,
	}
	if err := database.DB.Create(loc).Error; err != nil {
		t.Fatalf("seed locator: %v", err)
	}

	prod1 = &models.Product{
		ID:       uuid.New().String(),
		SKU:      "VAL-SKU-001",
		Name:     "Valuation Product Alpha",
		Category: "Electronics",
		Price:    100.0,
	}
	if err := database.DB.Create(prod1).Error; err != nil {
		t.Fatalf("seed product1: %v", err)
	}

	prod2 = &models.Product{
		ID:       uuid.New().String(),
		SKU:      "VAL-SKU-002",
		Name:     "Valuation Product Beta",
		Category: "Accessories",
		Price:    25.0,
	}
	if err := database.DB.Create(prod2).Error; err != nil {
		t.Fatalf("seed product2: %v", err)
	}

	return prod1, prod2, wh, loc
}

func TestFetchFIFOValuation_EmptyDB(t *testing.T) {
	setupTestDB(t)

	rows, summary, err := FetchFIFOValuation(ValuationFilter{})
	if err != nil {
		t.Fatalf("unexpected error on empty DB: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
	if summary.TotalValue != 0 || summary.TotalQty != 0 {
		t.Errorf("expected zero summary on empty DB, got %+v", summary)
	}
}

func TestFetchFIFOValuation_FallsBackToProductPrice(t *testing.T) {
	setupTestDB(t)
	prod1, _, _, loc := seedValuationFixtures(t)

	// One storage lot, no ledger entries — unit cost must come from products.price
	storage := models.Storage{
		ID:          uuid.New().String(),
		ProductID:   prod1.ID,
		LocatorID:   loc.ID,
		BatchNumber: "BAT-FALLBACK-01",
		ReceivedAt:  time.Now(),
		QtyOnHand:   10,
		UpdatedAt:   time.Now(),
	}
	database.DB.Create(&storage)

	rows, summary, err := FetchFIFOValuation(ValuationFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if row.UnitCost != prod1.Price {
		t.Errorf("expected unit cost %.2f (product price), got %.2f", prod1.Price, row.UnitCost)
	}
	expectedTotal := prod1.Price * float64(storage.QtyOnHand)
	if row.TotalValue != expectedTotal {
		t.Errorf("expected total value %.2f, got %.2f", expectedTotal, row.TotalValue)
	}
	if summary.TotalValue != expectedTotal {
		t.Errorf("expected summary total %.2f, got %.2f", expectedTotal, summary.TotalValue)
	}
	if summary.TotalQty != storage.QtyOnHand {
		t.Errorf("expected summary qty %d, got %d", storage.QtyOnHand, summary.TotalQty)
	}
}

func TestFetchFIFOValuation_AgingBuckets(t *testing.T) {
	setupTestDB(t)
	prod1, _, _, loc := seedValuationFixtures(t)

	now := time.Now()
	lots := []models.Storage{
		{
			ID:          uuid.New().String(),
			ProductID:   prod1.ID,
			LocatorID:   loc.ID,
			BatchNumber: "BAT-FRESH",
			ReceivedAt:  now, // 0 days → "0-30"
			QtyOnHand:   5,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New().String(),
			ProductID:   prod1.ID,
			LocatorID:   loc.ID,
			BatchNumber: "BAT-MODERATE",
			ReceivedAt:  now.AddDate(0, 0, -60), // 60 days → "31-90"
			QtyOnHand:   8,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New().String(),
			ProductID:   prod1.ID,
			LocatorID:   loc.ID,
			BatchNumber: "BAT-SLOW",
			ReceivedAt:  now.AddDate(0, 0, -120), // 120 days → "91+"
			QtyOnHand:   3,
			UpdatedAt:   now,
		},
	}
	for i := range lots {
		database.DB.Create(&lots[i])
	}

	rows, summary, err := FetchFIFOValuation(ValuationFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	buckets := map[string]int{}
	for _, r := range rows {
		buckets[r.AgingBucket]++
	}
	if buckets["0-30"] != 1 {
		t.Errorf("expected 1 row in 0-30 bucket, got %d", buckets["0-30"])
	}
	if buckets["31-90"] != 1 {
		t.Errorf("expected 1 row in 31-90 bucket, got %d", buckets["31-90"])
	}
	if buckets["91+"] != 1 {
		t.Errorf("expected 1 row in 91+ bucket, got %d", buckets["91+"])
	}

	// Summary bucket values: all use product price (no ledger entries)
	price := prod1.Price
	expectedFresh := price * 5
	expectedMod := price * 8
	expectedSlow := price * 3

	if summary.Bucket0_30 != expectedFresh {
		t.Errorf("Bucket0_30: expected %.2f, got %.2f", expectedFresh, summary.Bucket0_30)
	}
	if summary.Bucket31_90 != expectedMod {
		t.Errorf("Bucket31_90: expected %.2f, got %.2f", expectedMod, summary.Bucket31_90)
	}
	if summary.Bucket91Plus != expectedSlow {
		t.Errorf("Bucket91Plus: expected %.2f, got %.2f", expectedSlow, summary.Bucket91Plus)
	}
	if summary.TotalValue != expectedFresh+expectedMod+expectedSlow {
		t.Errorf("TotalValue mismatch: got %.2f", summary.TotalValue)
	}
}

func TestFetchFIFOValuation_FilterByAgingBucket(t *testing.T) {
	setupTestDB(t)
	prod1, _, _, loc := seedValuationFixtures(t)

	now := time.Now()
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: prod1.ID, LocatorID: loc.ID,
		BatchNumber: "BAT-F1", ReceivedAt: now, QtyOnHand: 10, UpdatedAt: now,
	})
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: prod1.ID, LocatorID: loc.ID,
		BatchNumber: "BAT-F2", ReceivedAt: now.AddDate(0, 0, -100), QtyOnHand: 4, UpdatedAt: now,
	})

	rowsFresh, _, err := FetchFIFOValuation(ValuationFilter{AsBucket: "0-30"})
	if err != nil {
		t.Fatalf("filter 0-30: %v", err)
	}
	if len(rowsFresh) != 1 {
		t.Errorf("expected 1 fresh row, got %d", len(rowsFresh))
	}
	if rowsFresh[0].AgingBucket != "0-30" {
		t.Errorf("wrong bucket: %s", rowsFresh[0].AgingBucket)
	}

	rowsSlow, _, err := FetchFIFOValuation(ValuationFilter{AsBucket: "91+"})
	if err != nil {
		t.Fatalf("filter 91+: %v", err)
	}
	if len(rowsSlow) != 1 {
		t.Errorf("expected 1 slow row, got %d", len(rowsSlow))
	}
	if rowsSlow[0].AgingBucket != "91+" {
		t.Errorf("wrong bucket: %s", rowsSlow[0].AgingBucket)
	}
}

func TestFetchFIFOValuation_FilterByWarehouse(t *testing.T) {
	setupTestDB(t)
	prod1, _, wh, loc := seedValuationFixtures(t)

	// Second warehouse with its own locator and stock
	wh2 := &models.Warehouse{
		ID: uuid.New().String(), Code: "WH-OTHER", Name: "Other WH", IsActive: true,
	}
	database.DB.Create(wh2)
	loc2 := &models.Locator{
		ID: uuid.New().String(), WarehouseID: wh2.ID, Code: "WH-OTHER-A-1",
		Zone: "A", Aisle: "1", Shelf: "1", Level: "1", IsActive: true,
	}
	database.DB.Create(loc2)
	now := time.Now()
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: prod1.ID, LocatorID: loc.ID,
		BatchNumber: "BAT-WH1", ReceivedAt: now, QtyOnHand: 10, UpdatedAt: now,
	})
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: prod1.ID, LocatorID: loc2.ID,
		BatchNumber: "BAT-WH2", ReceivedAt: now, QtyOnHand: 20, UpdatedAt: now,
	})

	// Filter to wh1 only — should return 1 row
	rows, _, err := FetchFIFOValuation(ValuationFilter{WarehouseID: wh.ID})
	if err != nil {
		t.Fatalf("filter warehouse: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row for wh1, got %d", len(rows))
	}
	if rows[0].WarehouseCode != wh.Code {
		t.Errorf("expected warehouse %s, got %s", wh.Code, rows[0].WarehouseCode)
	}
}

func TestFetchFIFOValuation_FilterByCategory(t *testing.T) {
	setupTestDB(t)
	prod1, prod2, _, loc := seedValuationFixtures(t)

	now := time.Now()
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: prod1.ID, LocatorID: loc.ID,
		BatchNumber: "BAT-CAT1", ReceivedAt: now, QtyOnHand: 5, UpdatedAt: now,
	})
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: prod2.ID, LocatorID: loc.ID,
		BatchNumber: "BAT-CAT2", ReceivedAt: now, QtyOnHand: 3, UpdatedAt: now,
	})

	rows, _, err := FetchFIFOValuation(ValuationFilter{Category: "Electronics"})
	if err != nil {
		t.Fatalf("filter category: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 Electronics row, got %d", len(rows))
	}
	if rows[0].Category != "Electronics" {
		t.Errorf("expected category Electronics, got %s", rows[0].Category)
	}
}

func TestFetchFIFOValuation_SkipsZeroQty(t *testing.T) {
	setupTestDB(t)
	prod1, _, _, loc := seedValuationFixtures(t)

	now := time.Now()
	// Lot with stock
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: prod1.ID, LocatorID: loc.ID,
		BatchNumber: "BAT-HAS-STOCK", ReceivedAt: now, QtyOnHand: 7, UpdatedAt: now,
	})
	// Lot with zero qty — should be excluded from valuation
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: prod1.ID, LocatorID: loc.ID,
		BatchNumber: "BAT-ZERO-QTY", ReceivedAt: now, QtyOnHand: 0, UpdatedAt: now,
	})

	rows, _, err := FetchFIFOValuation(ValuationFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row (zero-qty lot excluded), got %d", len(rows))
	}
}

func TestFetchFIFOValuation_UniqueProductCount(t *testing.T) {
	setupTestDB(t)
	prod1, prod2, _, loc := seedValuationFixtures(t)

	now := time.Now()
	// Two lots for prod1, one lot for prod2
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: prod1.ID, LocatorID: loc.ID,
		BatchNumber: "BAT-P1-A", ReceivedAt: now, QtyOnHand: 5, UpdatedAt: now,
	})
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: prod1.ID, LocatorID: loc.ID,
		BatchNumber: "BAT-P1-B", ReceivedAt: now, QtyOnHand: 3, UpdatedAt: now,
	})
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: prod2.ID, LocatorID: loc.ID,
		BatchNumber: "BAT-P2-A", ReceivedAt: now, QtyOnHand: 2, UpdatedAt: now,
	})

	_, summary, err := FetchFIFOValuation(ValuationFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.UniqueProducts != 2 {
		t.Errorf("expected UniqueProducts=2, got %d", summary.UniqueProducts)
	}
}

func TestFetchValuationWarehouses(t *testing.T) {
	setupTestDB(t)
	_, _, wh, loc := seedValuationFixtures(t)

	// Warehouse with no stock should not appear
	emptyWH := &models.Warehouse{
		ID: uuid.New().String(), Code: "WH-EMPTY", Name: "Empty WH", IsActive: true,
	}
	database.DB.Create(emptyWH)

	// Add stock to the main warehouse locator
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: uuid.New().String(), LocatorID: loc.ID,
		BatchNumber: "BAT-WH-TEST", ReceivedAt: time.Now(), QtyOnHand: 1, UpdatedAt: time.Now(),
	})

	warehouses, err := FetchValuationWarehouses()
	if err != nil {
		t.Fatalf("FetchValuationWarehouses: %v", err)
	}

	found := false
	for _, w := range warehouses {
		if w.ID == wh.ID {
			found = true
		}
		if w.ID == emptyWH.ID {
			t.Errorf("empty warehouse %s should not appear in results", emptyWH.Code)
		}
	}
	if !found {
		t.Errorf("warehouse %s with stock should appear in results", wh.Code)
	}
}

func TestFetchValuationCategories(t *testing.T) {
	setupTestDB(t)
	prod1, prod2, _, loc := seedValuationFixtures(t)

	now := time.Now()
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: prod1.ID, LocatorID: loc.ID,
		BatchNumber: "BAT-C1", ReceivedAt: now, QtyOnHand: 5, UpdatedAt: now,
	})
	database.DB.Create(&models.Storage{
		ID: uuid.New().String(), ProductID: prod2.ID, LocatorID: loc.ID,
		BatchNumber: "BAT-C2", ReceivedAt: now, QtyOnHand: 3, UpdatedAt: now,
	})

	categories, err := FetchValuationCategories()
	if err != nil {
		t.Fatalf("FetchValuationCategories: %v", err)
	}
	if len(categories) < 2 {
		t.Errorf("expected at least 2 categories, got %d: %v", len(categories), categories)
	}

	catSet := map[string]bool{}
	for _, c := range categories {
		catSet[c] = true
	}
	if !catSet["Electronics"] {
		t.Error("expected category 'Electronics' to be returned")
	}
	if !catSet["Accessories"] {
		t.Error("expected category 'Accessories' to be returned")
	}
}

func TestAgingBucketBoundaries(t *testing.T) {
	cases := []struct {
		days   int
		expect string
	}{
		{0, "0-30"},
		{30, "0-30"},
		{31, "31-90"},
		{90, "31-90"},
		{91, "91+"},
		{365, "91+"},
	}
	for _, tc := range cases {
		got := agingBucket(tc.days)
		if got != tc.expect {
			t.Errorf("agingBucket(%d): expected %q, got %q", tc.days, tc.expect, got)
		}
	}
}
