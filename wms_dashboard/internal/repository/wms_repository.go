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

// InsertInventoryLedger creates an atomic stock mutation record in the ledger
func InsertInventoryLedger(tx *gorm.DB, date time.Time, productID, locatorID, batchNo, txnType, docNo string, qtyChange, batchBalance int, accountNo, contraAccountNo, createdBy string) error {
	var accPtr *string
	if accountNo != "" {
		accPtr = &accountNo
	}
	var contraAccPtr *string
	if contraAccountNo != "" {
		contraAccPtr = &contraAccountNo
	}

	ledger := models.InventoryLedger{
		ID:              uuid.New().String(),
		TransactionDate: date,
		ProductID:       productID,
		LocatorID:       locatorID,
		BatchNumber:     batchNo,
		TransactionType: txnType,
		DocumentNo:      docNo,
		QtyChange:       qtyChange,
		BatchBalance:    batchBalance,
		AccountNo:       accPtr,
		ContraAccountNo: contraAccPtr,
		CreatedBy:       createdBy,
	}
	return tx.Create(&ledger).Error
}

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
		query = query.Where("LOWER(products.name) LIKE LOWER(?) OR LOWER(products.sku) LIKE LOWER(?)", "%"+search+"%", "%"+search+"%")
	}

	err := query.
		Joins("LEFT JOIN storages ON products.id = storages.product_id").
		Joins("LEFT JOIN uoms ON products.uom_id = uoms.id").
		Group("products.id, products.sku, products.name, products.category, products.price, uoms.code").
		Scan(&results).Error

	return results, err
}

// CreateInventoryMovement handles creating movement headers and lines, allocating stock for OUTBOUND FIFO
func CreateInventoryMovement(movement *models.InventoryMovement, lines []models.InventoryMovementLine) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var docNo string
		var err error
		if movement.MovementType == models.MvtTypeTransfer {
			docNo, err = GetNextSequence(tx, "inventory_transfers")
		} else {
			docNo, err = GetNextSequence(tx, "inventory_movements")
		}
		if err != nil {
			return err
		}
		movement.DocumentNo = docNo
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

			if movement.IsCrossDock && movement.MovementType == models.MvtTypeInbound {
				line.ToLocatorID = "loc-crossdock-01"
			}

			if movement.MovementType == models.MvtTypeOutbound {
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
			} else if movement.MovementType == models.MvtTypeRTV {
				if line.IsFromHold {
					if line.FromLocatorID == "" || line.BatchNumber == "" {
						return errors.New("locator and batch number are required to return from QC Hold")
					}
					var lot models.Storage
					err := tx.Where("product_id = ? AND locator_id = ? AND batch_number = ?", line.ProductID, line.FromLocatorID, line.BatchNumber).First(&lot).Error
					if err != nil {
						return fmt.Errorf("matching storage lot not found for return: %w", err)
					}
					if lot.QtyOnHold < line.RequestedQuantity {
						return fmt.Errorf("insufficient hold stock in selected lot. Requested return: %d, On Hold: %d", line.RequestedQuantity, lot.QtyOnHold)
					}
				} else {
					// Return from regular stock
					if line.FromLocatorID != "" && line.BatchNumber != "" {
						// Sourced from a specific lot
						var lot models.Storage
						err := tx.Where("product_id = ? AND locator_id = ? AND batch_number = ?", line.ProductID, line.FromLocatorID, line.BatchNumber).First(&lot).Error
						if err != nil {
							return fmt.Errorf("matching storage lot not found for return: %w", err)
						}
						available := lot.QtyOnHand - lot.QtyReserved - lot.QtyOnHold
						if available < line.RequestedQuantity {
							return fmt.Errorf("insufficient available stock in selected lot. Requested return: %d, Available: %d", line.RequestedQuantity, available)
						}
						lot.QtyReserved += line.RequestedQuantity
						if err := tx.Save(&lot).Error; err != nil {
							return err
						}
					} else {
						// FIFO allocation fallback
						requiredQty := line.RequestedQuantity
						allocatedQty := 0

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

							lot.QtyReserved += take
							if err := tx.Save(&lot).Error; err != nil {
								return err
							}

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
				}
			} else if movement.MovementType == models.MvtTypeTransfer {
				if line.FromLocatorID == "" || line.ToLocatorID == "" {
					return errors.New("both source and destination locators are required for transfers")
				}
				if line.FromLocatorID == line.ToLocatorID {
					return errors.New("source and destination locators must be different")
				}

				requiredQty := line.RequestedQuantity
				allocatedQty := 0

				var lots []models.Storage
				err := tx.Where("product_id = ? AND locator_id = ? AND (qty_on_hand - qty_reserved - qty_on_hold) > 0", line.ProductID, line.FromLocatorID).
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

					lot.QtyReserved += take
					if err := tx.Save(&lot).Error; err != nil {
						return err
					}

					if allocatedQty == 0 {
						line.BatchNumber = lot.BatchNumber
					}

					allocatedQty += take
				}

				if allocatedQty < requiredQty {
					return fmt.Errorf("insufficient stock in source locator for SKU product ID: %s. Requested: %d, Available: %d", line.ProductID, requiredQty, allocatedQty)
				}
			}

			// Save line item, omitting empty FK locator fields so Postgres stores NULL not ''
			q := tx
			if line.FromLocatorID == "" {
				q = q.Omit("FromLocatorID")
			}
			if line.ToLocatorID == "" {
				q = q.Omit("ToLocatorID")
			}
			if err := q.Create(line).Error; err != nil {
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

		if movement.Status == models.MvtStatusJournaled || movement.Status == models.MvtStatusCompleted || movement.Status == models.MvtStatusRejected {
			return errors.New("movement has already been finalized or closed")
		}

		if movement.IsCrossDock {
			return errors.New("cannot journal cross-dock movement directly; use cross-dock operational workflow")
		}

		for i := range movement.Lines {
			line := &movement.Lines[i]
			actualQty := line.RequestedQuantity // Assume perfect operations for now
			line.ActualQuantity = actualQty

			if movement.MovementType == models.MvtTypeInbound {
				// Generate a FIFO batch number for the inbound lot
				batchNo, err := GetNextSequence(tx, "storages")
				if err != nil {
					return err
				}
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

				// Insert Ledger (INBOUND: Debit Inventory 11000, Credit AP 21000)
				err = InsertInventoryLedger(tx, time.Now(), line.ProductID, line.ToLocatorID, batchNo, "INBOUND", movement.DocumentNo, actualQty, storage.QtyOnHand, models.AccInventoryAsset, models.AccAccountsPayable, movement.CreatedBy)
				if err != nil {
					return err
				}
			} else if movement.MovementType == models.MvtTypeOutbound {
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

					// Insert Ledger (OUTBOUND: Debit COGS 51000, Credit Inventory 11000)
					err = InsertInventoryLedger(tx, time.Now(), line.ProductID, lot.LocatorID, lot.BatchNumber, "OUTBOUND", movement.DocumentNo, -deduct, lot.QtyOnHand, models.AccCOGS, models.AccInventoryAsset, movement.CreatedBy)
					if err != nil {
						return err
					}

					remainingDeduct -= deduct
				}
			} else if movement.MovementType == models.MvtTypeRTV {
				if line.IsFromHold {
					if line.FromLocatorID == "" || line.BatchNumber == "" {
						return errors.New("locator and batch number are required to return from QC Hold")
					}
					var lot models.Storage
					err := tx.Where("product_id = ? AND locator_id = ? AND batch_number = ?", line.ProductID, line.FromLocatorID, line.BatchNumber).First(&lot).Error
					if err != nil {
						return fmt.Errorf("matching storage lot not found for return: %w", err)
					}
					if lot.QtyOnHold < actualQty {
						return fmt.Errorf("insufficient hold stock in selected lot. Requested return: %d, On Hold: %d", actualQty, lot.QtyOnHold)
					}
					if lot.QtyOnHand < actualQty {
						return fmt.Errorf("insufficient physical stock in selected lot. Requested return: %d, On Hand: %d", actualQty, lot.QtyOnHand)
					}

					lot.QtyOnHand -= actualQty
					lot.QtyOnHold -= actualQty
					lot.UpdatedAt = time.Now()
					if err := tx.Save(&lot).Error; err != nil {
						return err
					}

					// Insert Ledger (RTV from Hold: Debit AP 21000, Credit Inventory 11000)
					err = InsertInventoryLedger(tx, time.Now(), line.ProductID, lot.LocatorID, lot.BatchNumber, "RTV", movement.DocumentNo, -actualQty, lot.QtyOnHand, models.AccAccountsPayable, models.AccInventoryAsset, movement.CreatedBy)
					if err != nil {
						return err
					}

					// Fetch active QC hold records and consume them
					var qcHolds []models.QCHold
					err = tx.Where("storage_id = ? AND status = 'ACTIVE'", lot.ID).Order("created_at ASC").Find(&qcHolds).Error
					if err != nil {
						return err
					}

					remainingToRelease := actualQty
					for j := range qcHolds {
						qcHold := &qcHolds[j]
						if remainingToRelease <= 0 {
							break
						}
						release := remainingToRelease
						if qcHold.Qty < release {
							release = qcHold.Qty
						}
						qcHold.Qty -= release
						if qcHold.Qty == 0 {
							qcHold.Status = "RELEASED"
							now := time.Now()
							qcHold.ReleasedAt = &now
							qcHold.ReleasedBy = movement.CreatedBy
						}
						if err := tx.Save(qcHold).Error; err != nil {
							return err
						}
						remainingToRelease -= release
					}
				} else {
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

						// Insert Ledger (RTV Normal: Debit AP 21000, Credit Inventory 11000)
						err = InsertInventoryLedger(tx, time.Now(), line.ProductID, lot.LocatorID, lot.BatchNumber, "RTV", movement.DocumentNo, -deduct, lot.QtyOnHand, models.AccAccountsPayable, models.AccInventoryAsset, movement.CreatedBy)
						if err != nil {
							return err
						}

						remainingDeduct -= deduct
					}

					if remainingDeduct > 0 {
						// Fallback to match specific locator and batch if still remaining
						var lot models.Storage
						err := tx.Where("product_id = ? AND locator_id = ? AND batch_number = ?", line.ProductID, line.FromLocatorID, line.BatchNumber).First(&lot).Error
						if err == nil {
							deduct := remainingDeduct
							if lot.QtyOnHand < deduct {
								deduct = lot.QtyOnHand
							}
							lot.QtyOnHand -= deduct
							if lot.QtyReserved >= deduct {
								lot.QtyReserved -= deduct
							} else {
								lot.QtyReserved = 0
							}
							lot.UpdatedAt = time.Now()
							if err := tx.Save(&lot).Error; err != nil {
								return err
							}

							// Insert Ledger (RTV Fallback: Debit AP 21000, Credit Inventory 11000)
							err = InsertInventoryLedger(tx, time.Now(), line.ProductID, lot.LocatorID, lot.BatchNumber, "RTV", movement.DocumentNo, -deduct, lot.QtyOnHand, models.AccAccountsPayable, models.AccInventoryAsset, movement.CreatedBy)
							if err != nil {
								return err
							}

							remainingDeduct -= deduct
						}
					}
				}
			} else if movement.MovementType == models.MvtTypeTransfer {
				if line.FromLocatorID == "" || line.ToLocatorID == "" {
					return errors.New("both source and destination locators are required for transfers")
				}

				remainingDeduct := actualQty

				var lots []models.Storage
				err := tx.Where("product_id = ? AND locator_id = ? AND qty_reserved > 0", line.ProductID, line.FromLocatorID).
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

					var targetLot models.Storage
					err = tx.Where("product_id = ? AND locator_id = ? AND batch_number = ?", line.ProductID, line.ToLocatorID, lot.BatchNumber).
						First(&targetLot).Error

					if err == nil {
						targetLot.QtyOnHand += deduct
						targetLot.UpdatedAt = time.Now()
						if err := tx.Save(&targetLot).Error; err != nil {
							return err
						}
					} else if errors.Is(err, gorm.ErrRecordNotFound) {
						targetLot = models.Storage{
							ID:          uuid.New().String(),
							ProductID:   line.ProductID,
							LocatorID:   line.ToLocatorID,
							BatchNumber: lot.BatchNumber,
							ReceivedAt:  lot.ReceivedAt,
							QtyOnHand:   deduct,
							QtyReserved: 0,
							QtyOnHold:   0,
							UpdatedAt:   time.Now(),
						}
						if err := tx.Create(&targetLot).Error; err != nil {
							return err
						}
					} else {
						return err
					}

					err = InsertInventoryLedger(tx, time.Now(), line.ProductID, lot.LocatorID, lot.BatchNumber, "TRANSFER", movement.DocumentNo, -deduct, lot.QtyOnHand, models.AccInventoryAsset, models.AccInventoryAsset, movement.CreatedBy)
					if err != nil {
						return err
					}

					err = InsertInventoryLedger(tx, time.Now(), line.ProductID, targetLot.LocatorID, targetLot.BatchNumber, "TRANSFER", movement.DocumentNo, deduct, targetLot.QtyOnHand, models.AccInventoryAsset, models.AccInventoryAsset, movement.CreatedBy)
					if err != nil {
						return err
					}

					remainingDeduct -= deduct
				}
			}

			// Update line's actual quantity and batch details
			// Omit empty FK fields so Postgres stores NULL not ''
			qSave := tx
			if line.FromLocatorID == "" {
				qSave = qSave.Omit("FromLocatorID")
			}
			if line.ToLocatorID == "" {
				qSave = qSave.Omit("ToLocatorID")
			}
			if err := qSave.Save(line).Error; err != nil {
				return err
			}
		}

		// Update movement status to JOURNALED
		movement.Status = models.MvtStatusJournaled
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

		if movement.Status == models.MvtStatusJournaled || movement.Status == models.MvtStatusCompleted || movement.Status == models.MvtStatusRejected {
			return errors.New("movement has already been finalized or closed")
		}

		if models.HasReservation(movement.MovementType) {
			// Release allocations
			for i := range movement.Lines {
				line := &movement.Lines[i]
				if movement.MovementType == models.MvtTypeRTV && line.IsFromHold {
					continue // No reservation was made for QC Hold returns
				}
				releasedQty := line.RequestedQuantity

				var lots []models.Storage
				var err error
				if movement.MovementType == models.MvtTypeTransfer {
					err = tx.Where("product_id = ? AND locator_id = ? AND qty_reserved > 0", line.ProductID, line.FromLocatorID).
						Order("received_at DESC").
						Find(&lots).Error
				} else {
					err = tx.Where("product_id = ? AND qty_reserved > 0", line.ProductID).
						Order("received_at DESC"). // Release from newest first
						Find(&lots).Error
				}

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
		movement.Status = models.MvtStatusRejected
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
		if status == models.MvtStatusCompleted {
			if movement.IsCrossDock {
				if movement.Status != models.MvtStatusOutbound {
					return errors.New("cannot complete cross-dock movement before it is outbound dispatched")
				}
			} else {
				if movement.Status != models.MvtStatusJournaled {
					return errors.New("cannot complete movement before it is journaled")
				}
			}
		}

		movement.Status = status
		movement.UpdatedAt = time.Now()
		return tx.Save(&movement).Error
	})
}

// ProcessCrossDockInbound handles the inbound stage of cross docking
func ProcessCrossDockInbound(movementID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var movement models.InventoryMovement
		if err := tx.Preload("Lines").First(&movement, "id = ?", movementID).Error; err != nil {
			return err
		}

		if !movement.IsCrossDock {
			return errors.New("movement is not marked for cross-docking")
		}
		if movement.Status != "IN_PROGRESS" {
			return fmt.Errorf("invalid status transition: cannot move to INBOUND from %s", movement.Status)
		}

		for i := range movement.Lines {
			line := &movement.Lines[i]
			actualQty := line.RequestedQuantity
			line.ActualQuantity = actualQty

			// Generate a FIFO batch number for the transit lot
			batchNo, err := GetNextSequence(tx, "storages")
			if err != nil {
				return err
			}
			line.BatchNumber = batchNo

			// Create new storage balance record in the cross-dock staging area
			storage := models.Storage{
				ID:          uuid.New().String(),
				ProductID:   line.ProductID,
				LocatorID:   "loc-crossdock-01",
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

			// Update the line item with the generated batch number
			// Omit empty FK fields so Postgres stores NULL not ''
			qSave := tx
			if line.FromLocatorID == "" {
				qSave = qSave.Omit("FromLocatorID")
			}
			if line.ToLocatorID == "" {
				qSave = qSave.Omit("ToLocatorID")
			}
			if err := qSave.Save(line).Error; err != nil {
				return err
			}

			// Insert Ledger (INBOUND: Debit Inventory 11000, Credit AP 21000)
			err = InsertInventoryLedger(tx, time.Now(), line.ProductID, "loc-crossdock-01", batchNo, "INBOUND", movement.DocumentNo, actualQty, storage.QtyOnHand, models.AccInventoryAsset, models.AccAccountsPayable, movement.CreatedBy)
			if err != nil {
				return err
			}
		}

		movement.Status = "INBOUND"
		movement.UpdatedAt = time.Now()
		return tx.Save(&movement).Error
	})
}

// ProcessCrossDockShipping handles initiating loading for cross docking
func ProcessCrossDockShipping(movementID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var movement models.InventoryMovement
		if err := tx.First(&movement, "id = ?", movementID).Error; err != nil {
			return err
		}

		if !movement.IsCrossDock {
			return errors.New("movement is not marked for cross-docking")
		}
		if movement.Status != "INBOUND" {
			return fmt.Errorf("invalid status transition: cannot move to SHIPPING from %s", movement.Status)
		}

		movement.Status = "SHIPPING"
		movement.UpdatedAt = time.Now()
		return tx.Save(&movement).Error
	})
}

// ProcessCrossDockOutbound handles dispatching/releasing cross-docked goods
func ProcessCrossDockOutbound(movementID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var movement models.InventoryMovement
		if err := tx.Preload("Lines").First(&movement, "id = ?", movementID).Error; err != nil {
			return err
		}

		if !movement.IsCrossDock {
			return errors.New("movement is not marked for cross-docking")
		}
		if movement.Status != "SHIPPING" {
			return fmt.Errorf("invalid status transition: cannot move to OUTBOUND from %s", movement.Status)
		}

		for i := range movement.Lines {
			line := &movement.Lines[i]
			actualQty := line.RequestedQuantity

			// Find the staging balance record
			var lot models.Storage
			err := tx.Where("product_id = ? AND locator_id = ? AND batch_number = ?", line.ProductID, "loc-crossdock-01", line.BatchNumber).First(&lot).Error
			if err != nil {
				return fmt.Errorf("cross-dock transit stock not found for batch %s: %w", line.BatchNumber, err)
			}

			if lot.QtyOnHand < actualQty {
				return fmt.Errorf("insufficient cross-dock transit stock: need %d, on hand %d", actualQty, lot.QtyOnHand)
			}

			// Deduct transit stock
			lot.QtyOnHand -= actualQty
			lot.UpdatedAt = time.Now()
			if err := tx.Save(&lot).Error; err != nil {
				return err
			}

			// Insert Ledger (OUTBOUND: Debit COGS 51000, Credit Inventory 11000)
			err = InsertInventoryLedger(tx, time.Now(), line.ProductID, lot.LocatorID, lot.BatchNumber, "OUTBOUND", movement.DocumentNo, -actualQty, lot.QtyOnHand, models.AccCOGS, models.AccInventoryAsset, movement.CreatedBy)
			if err != nil {
				return err
			}
		}

		movement.Status = "OUTBOUND"
		movement.UpdatedAt = time.Now()
		return tx.Save(&movement).Error
	})
}
