package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"wms_dashboard/internal/models"
)

// GetNextSequence generates the next document or batch number atomically inside a transaction.
// It resets the sequence to 1 if the current calendar year is greater than the stored fiscal year.
func GetNextSequence(tx *gorm.DB, usageTable string) (string, error) {
	var seq models.SequenceGenerator

	// Query with row-level transaction lock (SELECT ... FOR UPDATE) to guarantee atomic counter updates
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&seq, "usage_table = ?", usageTable).Error
	if err != nil {
		return "", fmt.Errorf("failed to retrieve sequence generator for '%s': %w", usageTable, err)
	}

	currentYear := time.Now().Year()

	// If a new fiscal year has started, reset current_number back to 1 and update the fiscal_year
	if currentYear > seq.FiscalYear {
		seq.FiscalYear = currentYear
		seq.CurrentNumber = 1
	}

	// Format final string: {PREFIX}-{FY}-{PADDED_NUMBER}
	formattedNo := fmt.Sprintf("%s-%d-%0*d", seq.Prefix, seq.FiscalYear, seq.NumberLength, seq.CurrentNumber)

	// Increment the counter
	seq.CurrentNumber++
	seq.UpdatedAt = time.Now()

	// Save back to sequence_generators table
	if err := tx.Save(&seq).Error; err != nil {
		return "", fmt.Errorf("failed to update sequence generator counter for '%s': %w", usageTable, err)
	}

	return formattedNo, nil
}
