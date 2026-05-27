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

// CreateKittingOrder creates a new kitting order
func CreateKittingOrder(kitting *models.InventoryKitting, lines []models.InventoryKittingLine) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		docNo, err := GetNextSequence(tx, "inventory_kittings")
		if err != nil {
			return err
		}
		kitting.DocumentNo = docNo
		kitting.ID = uuid.New().String()
		kitting.CreatedAt = time.Now()
		kitting.UpdatedAt = time.Now()

		if err := tx.Create(kitting).Error; err != nil {
			return err
		}

		for i := range lines {
			line := &lines[i]
			line.ID = uuid.New().String()
			line.KittingID = kitting.ID

			if line.ConsumedQty <= 0 {
				return errors.New("consumed quantity must be greater than zero")
			}

			// Validate if we have enough physically unreserved stock for the component
			var available int
			err := tx.Model(&models.Storage{}).
				Where("product_id = ? AND locator_id = ?", line.ProductID, line.LocatorID).
				Select("COALESCE(SUM(qty_on_hand - qty_reserved - qty_on_hold), 0)").
				Scan(&available).Error

			if err != nil {
				return err
			}

			if available < line.ConsumedQty {
				return fmt.Errorf("cannot consume %d items, only %d available (unreserved) in this locator", line.ConsumedQty, available)
			}

			if err := tx.Create(line).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// JournalizeKittingOrder commits the kitting order: deducts components, adds finished goods
func JournalizeKittingOrder(kittingID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var kitting models.InventoryKitting
		if err := tx.Preload("FinishedProduct").Preload("ComponentLines").Preload("ComponentLines.Product").First(&kitting, "id = ?", kittingID).Error; err != nil {
			return err
		}

		if kitting.Status != "OPEN" {
			return errors.New("kitting order is already finalized or rejected")
		}

		var totalComponentCost float64 = 0

		// 1. Deduct component lines
		for _, line := range kitting.ComponentLines {
			totalComponentCost += line.Product.Price * float64(line.ConsumedQty)
			
			deductAmount := line.ConsumedQty
			var lots []models.Storage
			err := tx.Where("product_id = ? AND locator_id = ? AND (qty_on_hand - qty_reserved - qty_on_hold) > 0", line.ProductID, line.LocatorID).
				Order("received_at ASC").
				Find(&lots).Error

			if err != nil {
				return err
			}

			for _, lot := range lots {
				if deductAmount <= 0 {
					break
				}

				available := lot.QtyOnHand - lot.QtyReserved - lot.QtyOnHold
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
				return fmt.Errorf("insufficient unreserved stock to consume component. Some stock might have been reserved elsewhere")
			}
		}

		// Update Bundle Price if applicable
		if kitting.FinishedProduct.IsBundle && kitting.FinishedQty > 0 {
			unitPrice := totalComponentCost / float64(kitting.FinishedQty)
			if err := tx.Model(&models.Product{}).Where("id = ?", kitting.FinishedProductID).Update("price", unitPrice).Error; err != nil {
				return err
			}
		}

		// 2. Add finished product
		batchNo, err := GetNextSequence(tx, "storages")
		if err != nil {
			return err
		}
		storage := models.Storage{
			ID:          uuid.New().String(),
			ProductID:   kitting.FinishedProductID,
			LocatorID:   kitting.FinishedLocatorID,
			BatchNumber: batchNo,
			ReceivedAt:  time.Now(),
			QtyOnHand:   kitting.FinishedQty,
			QtyReserved: 0,
			QtyOnHold:   0,
			UpdatedAt:   time.Now(),
		}
		if err := tx.Create(&storage).Error; err != nil {
			return err
		}

		kitting.Status = "JOURNALED"
		kitting.UpdatedAt = time.Now()
		return tx.Save(&kitting).Error
	})
}

// RejectKittingOrder cancels the kitting order
func RejectKittingOrder(kittingID string, reason string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var kit models.InventoryKitting
		if err := tx.First(&kit, "id = ?", kittingID).Error; err != nil {
			return err
		}

		if kit.Status != "OPEN" {
			return errors.New("kitting order is already finalized or rejected")
		}

		kit.Status = "REJECTED"
		kit.Remarks = fmt.Sprintf("%s (Rejected: %s)", kit.Remarks, reason)
		kit.UpdatedAt = time.Now()
		return tx.Save(&kit).Error
	})
}
