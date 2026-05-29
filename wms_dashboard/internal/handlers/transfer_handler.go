package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
	"wms_dashboard/internal/repository"
)

// GET /wms/transfers
func ServeTransfers(c *fiber.Ctx) error {
	var movements []models.InventoryMovement
	_ = database.DB.Preload("Lines.Product.UoM").
		Preload("Lines.FromLocator.Warehouse").
		Preload("Lines.ToLocator.Warehouse").
		Where("movement_type = ?", models.MvtTypeTransfer).
		Order("updated_at DESC").Find(&movements).Error

	var products []models.Product
	_ = database.DB.Preload("UoM").Find(&products).Error

	var stats struct {
		TotalCount     int64
		PendingCount   int64
		CompletedCount int64
	}
	_ = database.DB.Model(&models.InventoryMovement{}).Where("movement_type = ?", models.MvtTypeTransfer).Count(&stats.TotalCount).Error
	_ = database.DB.Model(&models.InventoryMovement{}).Where("movement_type = ? AND status IN (?, ?)", models.MvtTypeTransfer, models.MvtStatusOpen, models.MvtStatusInProgress).Count(&stats.PendingCount).Error
	_ = database.DB.Model(&models.InventoryMovement{}).Where("movement_type = ? AND status = ?", models.MvtTypeTransfer, models.MvtStatusCompleted).Count(&stats.CompletedCount).Error

	return renderPage(c, "transfers.html", fiber.Map{
		"Movements":      movements,
		"Products":       products,
		"TotalTransfers": stats.TotalCount,
		"PendingCount":   stats.PendingCount,
		"CompletedCount": stats.CompletedCount,
	})
}

// GET /wms/transfers/source-storages?product_id=...
func ServeTransferSourceStorages(c *fiber.Ctx) error {
	productID := c.Query("product_id")
	if productID == "" {
		return c.SendString(`<option value="">Select a source locator...</option>`)
	}

	var lots []models.Storage
	err := database.DB.Preload("Locator.Warehouse").
		Where("product_id = ? AND qty_on_hand > 0", productID).
		Find(&lots).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	type LocatorStock struct {
		LocatorID   string
		Warehouse   string
		LocatorCode string
		Available   int
	}
	locStocks := make(map[string]*LocatorStock)
	for _, lot := range lots {
		available := lot.QtyOnHand - lot.QtyReserved - lot.QtyOnHold
		if available <= 0 {
			continue
		}
		if item, exists := locStocks[lot.LocatorID]; exists {
			item.Available += available
		} else {
			locStocks[lot.LocatorID] = &LocatorStock{
				LocatorID:   lot.LocatorID,
				Warehouse:   lot.Locator.Warehouse.Code,
				LocatorCode: lot.Locator.Code,
				Available:   available,
			}
		}
	}

	html := `<option value="">Select a source locator...</option>`
	for _, item := range locStocks {
		html += fmt.Sprintf(
			`<option value="%s" data-available="%d">`+
				`Locator: %s/%s | Available: %d`+
				`</option>`,
			item.LocatorID, item.Available,
			item.Warehouse, item.LocatorCode, item.Available,
		)
	}

	return c.SendString(html)
}

// GET /wms/transfers/destinations?exclude_locator_id=...
func ServeTransferDestinations(c *fiber.Ctx) error {
	excludeLocatorID := c.Query("exclude_locator_id")

	var locators []models.Locator
	query := database.DB.Preload("Warehouse").Where("is_active = ?", true)
	if excludeLocatorID != "" {
		query = query.Where("id != ?", excludeLocatorID)
	}
	err := query.Find(&locators).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	html := `<option value="">Select a destination locator...</option>`
	for _, loc := range locators {
		html += fmt.Sprintf(
			`<option value="%s">Locator: %s/%s</option>`,
			loc.ID, loc.Warehouse.Code, loc.Code,
		)
	}

	return c.SendString(html)
}

// POST /wms/transfers/new
func CreateTransfer(c *fiber.Ctx) error {
	productID := c.FormValue("product_id")
	fromLocatorID := c.FormValue("from_locator_id")
	toLocatorID := c.FormValue("to_locator_id")
	quantityStr := c.FormValue("quantity")
	remarks := c.FormValue("remarks")

	var qty int
	_, _ = fmt.Sscanf(quantityStr, "%d", &qty)

	if productID == "" || fromLocatorID == "" || toLocatorID == "" || qty <= 0 {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Please select a product, source locator, destination locator, and enter a valid quantity.",
		})
	}

	if fromLocatorID == toLocatorID {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Source and destination locators must be different.",
		})
	}

	var product models.Product
	if err := database.DB.Preload("UoM").First(&product, "id = ?", productID).Error; err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Product not found.",
		})
	}

	userID := c.Locals("user_id").(string)

	movement := models.InventoryMovement{
		MovementType: models.MvtTypeTransfer,
		Status:       models.MvtStatusOpen,
		CreatedBy:    userID,
		Remarks:      remarks,
	}

	line := models.InventoryMovementLine{
		ProductID:         productID,
		FromLocatorID:     fromLocatorID,
		ToLocatorID:       toLocatorID,
		RequestedQuantity: qty,
	}

	err := repository.CreateInventoryMovement(&movement, []models.InventoryMovementLine{line})
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	setReloadToast(c, fmt.Sprintf("Inventory Transfer ticket %s created successfully.", movement.DocumentNo), true)
	return c.SendStatus(fiber.StatusOK)
}
