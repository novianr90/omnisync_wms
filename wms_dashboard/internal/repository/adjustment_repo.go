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

// CreateInventoryAdjustment creates a new adjustment ticket
func CreateInventoryAdjustment(adjustment *models.InventoryAdjustment, lines []models.InventoryAdjustmentLine) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		adjustment.ID = uuid.New().String()
		adjustment.CreatedAt = time.Now()
		adjustment.UpdatedAt = time.Now()

		if err := tx.Create(adjustment).Error; err != nil {
			return err
		}

		for i := range lines {
			line := &lines[i]
			line.ID = uuid.New().String()
			line.AdjustmentID = adjustment.ID

			// Validate if negative, we have enough physically unreserved stock
			if line.QtyDelta < 0 {
				var available int
				err := tx.Model(&models.Storage{}).
					Where("product_id = ? AND locator_id = ?", line.ProductID, line.LocatorID).
					Select("COALESCE(SUM(qty_on_hand - qty_reserved), 0)").
					Scan(&available).Error

				if err != nil {
					return err
				}

				if available < -line.QtyDelta {
					return fmt.Errorf("cannot deduct %d items, only %d available (unreserved) in this locator", -line.QtyDelta, available)
				}
			}

			if err := tx.Create(line).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// JournalizeInventoryAdjustment commits the physical stock changes
func JournalizeInventoryAdjustment(adjustmentID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var adj models.InventoryAdjustment
		if err := tx.Preload("Lines").First(&adj, "id = ?", adjustmentID).Error; err != nil {
			return err
		}

		if adj.Status != "OPEN" {
			return errors.New("adjustment is already finalized or rejected")
		}

		for _, line := range adj.Lines {
			if line.QtyDelta == 0 {
				continue
			}

			if line.QtyDelta > 0 {
				// Positive adjustment -> create new storage lot (found stock)
				batchNo := fmt.Sprintf("ADJ-%s-%s", adj.DocumentNo, line.ID[:8])
				storage := models.Storage{
					ID:          uuid.New().String(),
					ProductID:   line.ProductID,
					LocatorID:   line.LocatorID,
					BatchNumber: batchNo,
					ReceivedAt:  time.Now(),
					QtyOnHand:   line.QtyDelta,
					QtyReserved: 0,
					UpdatedAt:   time.Now(),
				}
				if err := tx.Create(&storage).Error; err != nil {
					return err
				}
			} else {
				// Negative adjustment -> deduct from oldest batches in locator
				deductAmount := -line.QtyDelta
				var lots []models.Storage
				err := tx.Where("product_id = ? AND locator_id = ? AND (qty_on_hand - qty_reserved) > 0", line.ProductID, line.LocatorID).
					Order("received_at ASC").
					Find(&lots).Error

				if err != nil {
					return err
				}

				for _, lot := range lots {
					if deductAmount <= 0 {
						break
					}

					available := lot.QtyOnHand - lot.QtyReserved
					take := deductAmount
					if available < take {
						take = available
					}

					lot.QtyOnHand -= take
					lot.UpdatedAt = time.Now()

					if err := tx.Save(&lot).Error; err != nil {
						return err
					}

					deductAmount -= take
				}

				if deductAmount > 0 {
					return fmt.Errorf("insufficient unreserved stock to deduct. Some stock might be reserved for outbound")
				}
			}
		}

		adj.Status = "JOURNALED"
		adj.UpdatedAt = time.Now()
		return tx.Save(&adj).Error
	})
}

// RejectInventoryAdjustment cancels the adjustment
func RejectInventoryAdjustment(adjustmentID string, reason string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var adj models.InventoryAdjustment
		if err := tx.First(&adj, "id = ?", adjustmentID).Error; err != nil {
			return err
		}

		if adj.Status != "OPEN" {
			return errors.New("adjustment is already finalized or rejected")
		}

		adj.Status = "REJECTED"
		adj.Remarks = fmt.Sprintf("%s (Rejected: %s)", adj.Remarks, reason)
		adj.UpdatedAt = time.Now()
		return tx.Save(&adj).Error
	})
}
