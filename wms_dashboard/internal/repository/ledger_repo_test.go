package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

func TestFetchInventoryLedger(t *testing.T) {
	setupTestDB(t)

	// Seed some base requirements
	prod := &models.Product{
		SKU:  "LEDGER-TEST",
		Name: "Ledger Test Product",
	}
	_ = CreateProduct(prod)

	now := time.Now()

	// Seed Ledger Entries manually for testing
	ledgers := []models.InventoryLedger{
		{
			ID:              uuid.New().String(),
			TransactionDate: now.Add(-48 * time.Hour),
			ProductID:       prod.ID,
			TransactionType: "INBOUND",
			DocumentNo:      "DOC-001",
			BatchNumber:     "BAT-001",
			QtyChange:       10,
			RunningBalance:  10,
		},
		{
			ID:              uuid.New().String(),
			TransactionDate: now.Add(-24 * time.Hour),
			ProductID:       prod.ID,
			TransactionType: "OUTBOUND",
			DocumentNo:      "DOC-002",
			BatchNumber:     "BAT-001",
			QtyChange:       -5,
			RunningBalance:  5,
		},
	}
	for _, l := range ledgers {
		if err := database.DB.Create(&l).Error; err != nil {
			t.Fatalf("failed to seed ledger: %v", err)
		}
	}

	// Test 1: Fetch without filters
	results, total, err := FetchInventoryLedger(LedgerFilter{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("FetchInventoryLedger error: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	// Check descending order
	if results[0].DocumentNo != "DOC-002" {
		t.Errorf("Expected latest transaction DOC-002 first, got %s", results[0].DocumentNo)
	}

	// Test 2: Filter by SKU
	results, total, err = FetchInventoryLedger(LedgerFilter{ProductSKU: "LEDGER-TEST", Limit: 10})
	if err != nil || total != 2 {
		t.Errorf("SKU Filter failed. Total: %d, Err: %v", total, err)
	}

	results, total, err = FetchInventoryLedger(LedgerFilter{ProductSKU: "NON-EXISTENT", Limit: 10})
	if err != nil || total != 0 {
		t.Errorf("SKU Filter should return 0 for non-existent. Total: %d", total)
	}

	// Test 3: Search text
	results, total, err = FetchInventoryLedger(LedgerFilter{Search: "DOC-001", Limit: 10})
	if err != nil || total != 1 {
		t.Errorf("Search text failed. Total: %d, Err: %v", total, err)
	}
}
