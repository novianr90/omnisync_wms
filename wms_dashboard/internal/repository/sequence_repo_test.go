package repository

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

func setupSequenceTestDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite db: %v", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(1)
	}

	err = db.AutoMigrate(
		&models.SequenceGenerator{},
	)
	if err != nil {
		t.Fatalf("failed to migrate sequence generator: %v", err)
	}

	database.DB = db
}

func TestGetNextSequence_Sequential(t *testing.T) {
	setupSequenceTestDB(t)

	// Seed generator
	generator := models.SequenceGenerator{
		ID:            "seq-mov-test",
		UsageTable:    "inventory_movements",
		Prefix:        "MOV",
		FiscalYear:    time.Now().Year(),
		CurrentNumber: 1,
		NumberLength:  5,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := database.DB.Create(&generator).Error; err != nil {
		t.Fatalf("failed to seed sequence generator: %v", err)
	}

	// Fetch 1st
	num1, err := GetNextSequence(database.DB, "inventory_movements")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected1 := fmt.Sprintf("MOV-%d-00001", time.Now().Year())
	if num1 != expected1 {
		t.Errorf("expected %s, got %s", expected1, num1)
	}

	// Fetch 2nd
	num2, err := GetNextSequence(database.DB, "inventory_movements")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected2 := fmt.Sprintf("MOV-%d-00002", time.Now().Year())
	if num2 != expected2 {
		t.Errorf("expected %s, got %s", expected2, num2)
	}
}

func TestGetNextSequence_YearReset(t *testing.T) {
	setupSequenceTestDB(t)

	// Seed generator with past year (2024)
	generator := models.SequenceGenerator{
		ID:            "seq-mov-test",
		UsageTable:    "inventory_movements",
		Prefix:        "MOV",
		FiscalYear:    2024,
		CurrentNumber: 15,
		NumberLength:  5,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := database.DB.Create(&generator).Error; err != nil {
		t.Fatalf("failed to seed sequence generator: %v", err)
	}

	// Fetch next sequence
	num, err := GetNextSequence(database.DB, "inventory_movements")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	currentYear := time.Now().Year()
	expected := fmt.Sprintf("MOV-%d-00001", currentYear)
	if num != expected {
		t.Errorf("expected reset sequence %s, got %s", expected, num)
	}

	// Check if updated in database
	var updated models.SequenceGenerator
	if err := database.DB.First(&updated, "usage_table = ?", "inventory_movements").Error; err != nil {
		t.Fatalf("failed to fetch updated sequence generator: %v", err)
	}

	if updated.FiscalYear != currentYear {
		t.Errorf("expected updated year to be %d, got %d", currentYear, updated.FiscalYear)
	}
	if updated.CurrentNumber != 2 {
		t.Errorf("expected updated current_number to be 2, got %d", updated.CurrentNumber)
	}
}

func TestGetNextSequence_Concurrency(t *testing.T) {
	setupSequenceTestDB(t)

	// Seed generator
	generator := models.SequenceGenerator{
		ID:            "seq-mov-test",
		UsageTable:    "inventory_movements",
		Prefix:        "MOV",
		FiscalYear:    time.Now().Year(),
		CurrentNumber: 1,
		NumberLength:  5,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := database.DB.Create(&generator).Error; err != nil {
		t.Fatalf("failed to seed sequence generator: %v", err)
	}

	const concurrencyCount = 50
	var wg sync.WaitGroup
	results := make(chan string, concurrencyCount)
	errorsChan := make(chan error, concurrencyCount)

	for i := 0; i < concurrencyCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Run GetNextSequence in its own GORM Transaction context to simulate real-world concurrent handlers
			err := database.DB.Transaction(func(tx *gorm.DB) error {
				num, err := GetNextSequence(tx, "inventory_movements")
				if err != nil {
					return err
				}
				results <- num
				return nil
			})
			if err != nil {
				errorsChan <- err
			}
		}()
	}

	wg.Wait()
	close(results)
	close(errorsChan)

	// Assert no errors occurred
	for err := range errorsChan {
		t.Errorf("concurrency error: %v", err)
	}

	// Verify all returned sequences are completely unique
	uniqueSet := make(map[string]bool)
	count := 0
	for res := range results {
		count++
		if uniqueSet[res] {
			t.Errorf("duplicate sequence number found: %s", res)
		}
		uniqueSet[res] = true
	}

	if count != concurrencyCount {
		t.Errorf("expected %d results, got %d", concurrencyCount, count)
	}

	// Verify database state at the end
	var finalSeq models.SequenceGenerator
	if err := database.DB.First(&finalSeq, "usage_table = ?", "inventory_movements").Error; err != nil {
		t.Fatalf("failed to fetch final sequence generator: %v", err)
	}

	expectedNextVal := concurrencyCount + 1
	if finalSeq.CurrentNumber != expectedNextVal {
		t.Errorf("expected final current_number to be %d, got %d", expectedNextVal, finalSeq.CurrentNumber)
	}
}
