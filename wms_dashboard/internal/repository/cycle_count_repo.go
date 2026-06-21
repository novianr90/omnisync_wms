package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

// CreateCycleCount generates a new cycle count sheet
func CreateCycleCount(userID string, locatorIDs []string) (*models.CycleCount, error) {
	var count models.CycleCount
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		docNo, err := GetNextSequence(tx, "cycle_counts")
		if err != nil {
			return err
		}

		count = models.CycleCount{
			ID:         uuid.New().String(),
			DocumentNo: docNo,
			Status:     "CREATED",
			CreatedBy:  userID,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if err := tx.Create(&count).Error; err != nil {
			return err
		}

		// Generate lines based on current storage for the selected locators
		// Only active products in those locators
		var storages []models.Storage
		if len(locatorIDs) > 0 {
			if err := tx.Where("locator_id IN ?", locatorIDs).Find(&storages).Error; err != nil {
				return err
			}

			// Freeze the locators
			if err := tx.Model(&models.Locator{}).Where("id IN ?", locatorIDs).Update("is_frozen", true).Error; err != nil {
				return err
			}
		} else {
			// If no locators selected, maybe count everything? Too big. 
			return errors.New("at least one locator must be selected")
		}

		// Group by product and locator to sum up quantities across batches
		type prodLoc struct {
			ProductID string
			LocatorID string
		}
		summary := make(map[prodLoc]int)
		for _, s := range storages {
			pl := prodLoc{ProductID: s.ProductID, LocatorID: s.LocatorID}
			summary[pl] += s.QtyOnHand
		}

		for pl, qty := range summary {
			line := models.CycleCountLine{
				ID:           uuid.New().String(),
				CycleCountID: count.ID,
				LocatorID:    pl.LocatorID,
				ProductID:    pl.ProductID,
				ExpectedQty:  qty,
				IsFrozen:     true, // Mark the line as responsible for freezing
			}
			if err := tx.Create(&line).Error; err != nil {
				return err
			}
		}

		return nil
	})

	return &count, err
}

// FetchCycleCounts lists all cycle counts
func FetchCycleCounts() ([]models.CycleCount, error) {
	var counts []models.CycleCount
	err := database.DB.Order("created_at DESC").Find(&counts).Error
	return counts, err
}

// GetCycleCountByID retrieves a cycle count with its lines and preloaded relations
func GetCycleCountByID(id string) (*models.CycleCount, error) {
	var count models.CycleCount
	err := database.DB.
		Preload("Lines").
		Preload("Lines.Product").
		Preload("Lines.Locator").
		Where("id = ?", id).First(&count).Error
	return &count, err
}

// UpdateCycleCountLine saves physical counts
func UpdateCycleCountLine(lineID string, countedQty int) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var line models.CycleCountLine
		if err := tx.Where("id = ?", lineID).First(&line).Error; err != nil {
			return err
		}

		line.CountedQty = &countedQty
		line.Variance = countedQty - line.ExpectedQty

		return tx.Save(&line).Error
	})
}

// UpdateCycleCountStatus safely transitions a count document's status
// ponytail: single method handles generic transitions without bloated logic
func UpdateCycleCountStatus(countID, newStatus string) error {
	return database.DB.Model(&models.CycleCount{}).Where("id = ?", countID).Update("status", newStatus).Error
}

// ReconcileCycleCount approves the count, generates an adjustment for variances, and unfreezes locators.
// It returns the generated AdjustmentID (if any) so the caller can journal it.
func ReconcileCycleCount(countID string, userID string) (string, error) {
	var createdAdjID string
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var count models.CycleCount
		if err := tx.Preload("Lines").Where("id = ?", countID).First(&count).Error; err != nil {
			return err
		}

		if count.Status != "IN_PROGRESS" {
			return errors.New("only IN_PROGRESS counts can be reconciled")
		}

		// Collect locators to unfreeze
		locatorsToUnfreeze := make(map[string]bool)
		
		var adjustmentLines []models.InventoryAdjustmentLine
		for _, line := range count.Lines {
			locatorsToUnfreeze[line.LocatorID] = true
			if line.CountedQty != nil && line.Variance != 0 {
				adjLine := models.InventoryAdjustmentLine{
					ProductID: line.ProductID,
					LocatorID: line.LocatorID,
					QtyDelta:  line.Variance,
				}
				adjustmentLines = append(adjustmentLines, adjLine)
			}
		}

		// If variances exist, create an adjustment
		if len(adjustmentLines) > 0 {
			adjDocNo, err := GetNextSequence(tx, "inventory_adjustments")
			if err != nil {
				return err
			}

			adj := models.InventoryAdjustment{
				ID:         uuid.New().String(),
				DocumentNo: adjDocNo,
				Status:     "OPEN",
				ReasonCode: "CYCLE_COUNT",
				Remarks:    fmt.Sprintf("Auto-generated for Cycle Count %s", count.DocumentNo),
				CreatedBy:  userID,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			if err := tx.Create(&adj).Error; err != nil {
				return err
			}

			for i := range adjustmentLines {
				adjustmentLines[i].ID = uuid.New().String()
				adjustmentLines[i].AdjustmentID = adj.ID
				if err := tx.Create(&adjustmentLines[i]).Error; err != nil {
					return err
				}
			}

			createdAdjID = adj.ID
		}

		// Unfreeze locators
		var locIDs []string
		for locID := range locatorsToUnfreeze {
			locIDs = append(locIDs, locID)
		}
		if len(locIDs) > 0 {
			if err := tx.Model(&models.Locator{}).Where("id IN ?", locIDs).Update("is_frozen", false).Error; err != nil {
				return err
			}
		}

		now := time.Now()
		count.Status = "RECONCILED"
		count.AdjustedAt = &now
		return tx.Save(&count).Error
	})

	return createdAdjID, err
}

// CancelCycleCount cancels the count sheet and unfreezes locators
func CancelCycleCount(countID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var count models.CycleCount
		if err := tx.Preload("Lines").Where("id = ?", countID).First(&count).Error; err != nil {
			return err
		}

		if count.Status != "IN_PROGRESS" && count.Status != "CREATED" {
			return errors.New("only IN_PROGRESS or CREATED counts can be canceled")
		}

		// Collect locators to unfreeze
		locatorsToUnfreeze := make(map[string]bool)
		for _, line := range count.Lines {
			locatorsToUnfreeze[line.LocatorID] = true
		}

		var locIDs []string
		for locID := range locatorsToUnfreeze {
			locIDs = append(locIDs, locID)
		}
		if len(locIDs) > 0 {
			if err := tx.Model(&models.Locator{}).Where("id IN ?", locIDs).Update("is_frozen", false).Error; err != nil {
				return err
			}
		}

		count.Status = "CANCELED"
		return tx.Save(&count).Error
	})
}
