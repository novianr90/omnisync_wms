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

// CreateQCHold freezes a specific quantity of stock in a given storage lot
func CreateQCHold(storageID string, qty int, reason, notes, createdBy string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Fetch the storage lot
		var storage models.Storage
		if err := tx.Preload("Product").Preload("Locator.Warehouse").First(&storage, "id = ?", storageID).Error; err != nil {
			return fmt.Errorf("storage lot not found: %w", err)
		}

		// 2. Check available (not reserved, not already on hold)
		available := storage.QtyOnHand - storage.QtyReserved - storage.QtyOnHold
		if qty <= 0 {
			return errors.New("hold quantity must be greater than zero")
		}
		if qty > available {
			return fmt.Errorf("cannot hold %d items, only %d available (unreserved & unfrozen)", qty, available)
		}

		// 3. Increment qty_on_hold on the storage lot
		storage.QtyOnHold += qty
		storage.UpdatedAt = time.Now()
		if err := tx.Save(&storage).Error; err != nil {
			return err
		}

		// 4. Create the QC Hold record
		hold := models.QCHold{
			ID:        uuid.New().String(),
			StorageID: storageID,
			Qty:       qty,
			Reason:    reason,
			Status:    "ACTIVE",
			Notes:     notes,
			CreatedBy: createdBy,
			CreatedAt: time.Now(),
		}
		return tx.Create(&hold).Error
	})
}

// ReleaseQCHold unfreezes the stock quantity from a specific hold
func ReleaseQCHold(holdID string, releasedBy string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Fetch the hold
		var hold models.QCHold
		if err := tx.First(&hold, "id = ?", holdID).Error; err != nil {
			return fmt.Errorf("QC hold not found: %w", err)
		}

		if hold.Status != "ACTIVE" {
			return errors.New("QC hold is already released")
		}

		// 2. Decrement qty_on_hold on the storage lot
		var storage models.Storage
		if err := tx.First(&storage, "id = ?", hold.StorageID).Error; err != nil {
			return fmt.Errorf("storage lot not found: %w", err)
		}

		release := hold.Qty
		if storage.QtyOnHold < release {
			release = storage.QtyOnHold
		}
		storage.QtyOnHold -= release
		storage.UpdatedAt = time.Now()
		if err := tx.Save(&storage).Error; err != nil {
			return err
		}

		// 3. Update the hold record to RELEASED
		now := time.Now()
		hold.Status = "RELEASED"
		hold.ReleasedBy = releasedBy
		hold.ReleasedAt = &now
		return tx.Save(&hold).Error
	})
}

// FetchQCHolds returns all QC holds with preloaded storage info
func FetchQCHolds() ([]models.QCHold, error) {
	var holds []models.QCHold
	err := database.DB.
		Preload("Storage.Product.UoM").
		Preload("Storage.Locator.Warehouse").
		Order("created_at DESC").
		Find(&holds).Error
	return holds, err
}

// FetchStoragesWithAvailableStock returns all storage lots that have available stock for hold
func FetchStoragesWithAvailableStock() ([]models.Storage, error) {
	var storages []models.Storage
	err := database.DB.
		Preload("Product.UoM").
		Preload("Locator.Warehouse").
		Where("(qty_on_hand - qty_reserved - qty_on_hold) > 0").
		Order("product_id, received_at ASC").
		Find(&storages).Error
	return storages, err
}
