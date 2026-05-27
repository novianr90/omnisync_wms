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

// ProductInventory represents an aggregated catalog row for the main dashboard view
type ProductInventory struct {
	ID          string  `json:"id"`
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	QtyOnHand   int     `json:"qty_on_hand"`
	QtyReserved int     `json:"qty_reserved"`
	QtyAvailable int    `json:"qty_available"`
	UoMCode     string  `json:"uom_code"`
}

// FetchInventoryCatalog returns all products along with their calculated storage quantities
func FetchInventoryCatalog(search string) ([]ProductInventory, error) {
	var results []ProductInventory

	query := database.DB.Model(&models.Product{}).
		Select("products.id, products.sku, products.name, products.category, products.price, uoms.code as uom_code, " +
			"COALESCE(SUM(storages.qty_on_hand), 0) as qty_on_hand, " +
			"COALESCE(SUM(storages.qty_reserved), 0) as qty_reserved, " +
			"COALESCE(SUM(storages.qty_on_hand - storages.qty_reserved - storages.qty_on_hold), 0) as qty_available")

	if search != "" {
		query = query.Where("products.name LIKE ? OR products.sku LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	err := query.
		Joins("LEFT JOIN storages ON products.id = storages.product_id").
		Joins("LEFT JOIN uoms ON products.uom_id = uoms.id").
		Group("products.id").
		Scan(&results).Error

	return results, err
}

// CreateInventoryMovement handles creating movement headers and lines, allocating stock for OUTBOUND FIFO
func CreateInventoryMovement(movement *models.InventoryMovement, lines []models.InventoryMovementLine) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		movement.ID = uuid.New().String()
		movement.CreatedAt = time.Now()
		movement.UpdatedAt = time.Now()

		// 1. Save movement header
		if err := tx.Create(movement).Error; err != nil {
			return err
		}

		// 2. Process each line
		for i := range lines {
			line := &lines[i]
			line.ID = uuid.New().String()
			line.MovementID = movement.ID

			if movement.MovementType == "OUTBOUND" {
				// FIFO stock allocation
				requiredQty := line.RequestedQuantity
				allocatedQty := 0

				// Query active storage lots sorted by received_at ASC (FIFO)
				var lots []models.Storage
				err := tx.Where("product_id = ? AND (qty_on_hand - qty_reserved - qty_on_hold) > 0", line.ProductID).
					Order("received_at ASC").
					Find(&lots).Error

				if err != nil {
					return err
				}

				for _, lot := range lots {
					available := lot.QtyOnHand - lot.QtyReserved - lot.QtyOnHold
					take := requiredQty - allocatedQty
					if take <= 0 {
						break
					}

					if available < take {
						take = available
					}

					// Update reservation on this lot
					lot.QtyReserved += take
					if err := tx.Save(&lot).Error; err != nil {
						return err
					}

					// Since an outbound line might span multiple lots/locators,
					// for simplicity in this dashboard we link it to the first matching lot.
					// A more complex system would create multiple sub-lines, but we associate the primary source locator.
					if allocatedQty == 0 {
						line.FromLocatorID = lot.LocatorID
						line.BatchNumber = lot.BatchNumber
					}

					allocatedQty += take
				}

				if allocatedQty < requiredQty {
					return fmt.Errorf("insufficient stock for SKU product ID: %s. Requested: %d, Available: %d", line.ProductID, requiredQty, allocatedQty)
				}
			}

			// Save line item
			if err := tx.Create(line).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// JournalizeInventoryMovement commits the physical stock updates
func JournalizeInventoryMovement(movementID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var movement models.InventoryMovement
		if err := tx.Preload("Lines").First(&movement, "id = ?", movementID).Error; err != nil {
			return err
		}

		if movement.Status == "JOURNALED" || movement.Status == "COMPLETED" || movement.Status == "REJECTED" {
			return errors.New("movement has already been finalized or closed")
		}

		for i := range movement.Lines {
			line := &movement.Lines[i]
			actualQty := line.RequestedQuantity // Assume perfect operations for now
			line.ActualQuantity = actualQty

			if movement.MovementType == "INBOUND" {
				// Generate a FIFO batch number for the inbound lot
				batchNo := fmt.Sprintf("BAT-%s-%s", movement.DocumentNo, line.ID[:8])
				line.BatchNumber = batchNo

				// Create new storage balance record
				storage := models.Storage{
					ID:          uuid.New().String(),
					ProductID:   line.ProductID,
					LocatorID:   line.ToLocatorID,
					BatchNumber: batchNo,
					ReceivedAt:  time.Now(),
					QtyOnHand:   actualQty,
					QtyReserved: 0,
					QtyOnHold:   0,
					UpdatedAt:   time.Now(),
				}

				if err := tx.Create(&storage).Error; err != nil {
					return err
				}
			} else if movement.MovementType == "OUTBOUND" {
				// Deduct stock from the reserved storage lots
				remainingDeduct := actualQty
				
				// Fetch lots that match the product and have reservations
				var lots []models.Storage
				err := tx.Where("product_id = ? AND qty_reserved > 0", line.ProductID).
					Order("received_at ASC").
					Find(&lots).Error

				if err != nil {
					return err
				}

				for _, lot := range lots {
					if remainingDeduct <= 0 {
						break
					}

					deduct := remainingDeduct
					if lot.QtyReserved < deduct {
						deduct = lot.QtyReserved
					}

					lot.QtyOnHand -= deduct
					lot.QtyReserved -= deduct
					lot.UpdatedAt = time.Now()

					if err := tx.Save(&lot).Error; err != nil {
						return err
					}

					remainingDeduct -= deduct
				}
			}

			// Update line's actual quantity and batch details
			if err := tx.Save(line).Error; err != nil {
				return err
			}
		}

		// Update movement status to JOURNALED
		movement.Status = "JOURNALED"
		movement.UpdatedAt = time.Now()
		return tx.Save(&movement).Error
	})
}

// RejectInventoryMovement cancels the movement and releases reservations
func RejectInventoryMovement(movementID string, reason string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var movement models.InventoryMovement
		if err := tx.Preload("Lines").First(&movement, "id = ?", movementID).Error; err != nil {
			return err
		}

		if movement.Status == "JOURNALED" || movement.Status == "COMPLETED" || movement.Status == "REJECTED" {
			return errors.New("movement has already been finalized or closed")
		}

		if movement.MovementType == "OUTBOUND" {
			// Release allocations
			for i := range movement.Lines {
				line := &movement.Lines[i]
				releasedQty := line.RequestedQuantity

				var lots []models.Storage
				err := tx.Where("product_id = ? AND qty_reserved > 0", line.ProductID).
					Order("received_at DESC"). // Release from newest first
					Find(&lots).Error

				if err != nil {
					return err
				}

				for _, lot := range lots {
					if releasedQty <= 0 {
						break
					}

					release := releasedQty
					if lot.QtyReserved < release {
						release = lot.QtyReserved
					}

					lot.QtyReserved -= release
					lot.UpdatedAt = time.Now()

					if err := tx.Save(&lot).Error; err != nil {
						return err
					}

					releasedQty -= release
				}
			}
		}

		// Set status to REJECTED
		movement.Status = "REJECTED"
		movement.RejectionReason = reason
		movement.UpdatedAt = time.Now()
		return tx.Save(&movement).Error
	})
}

// UpdateMovementStatus moves a movement through minor workflow stages (e.g. IN_PROGRESS, RECEIPT, COMPLETED)
func UpdateMovementStatus(movementID string, status string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var movement models.InventoryMovement
		if err := tx.First(&movement, "id = ?", movementID).Error; err != nil {
			return err
		}

		// Pre-journal validation
		if status == "COMPLETED" && movement.Status != "JOURNALED" {
			return errors.New("cannot complete movement before it is journaled")
		}

		movement.Status = status
		movement.UpdatedAt = time.Now()
		return tx.Save(&movement).Error
	})
}
