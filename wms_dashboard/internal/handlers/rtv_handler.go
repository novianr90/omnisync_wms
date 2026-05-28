package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
	"wms_dashboard/internal/repository"
)

// GET /wms/rtv
func ServeRTV(c *fiber.Ctx) error {
	var movements []models.InventoryMovement
	_ = database.DB.Preload("Lines.Product.UoM").Preload("Lines.FromLocator.Warehouse").
		Where("movement_type = ?", "RTV").
		Order("updated_at DESC").Find(&movements).Error

	var products []models.Product
	_ = database.DB.Preload("UoM").Find(&products).Error

	// Also we can calculate summary stats for RTV
	var stats struct {
		TotalCount    int64
		PendingCount  int64
		FromHoldCount int64
	}
	_ = database.DB.Model(&models.InventoryMovement{}).Where("movement_type = ?", "RTV").Count(&stats.TotalCount).Error
	_ = database.DB.Model(&models.InventoryMovement{}).Where("movement_type = ? AND status IN ('OPEN', 'IN_PROGRESS')", "RTV").Count(&stats.PendingCount).Error
	
	// Sourced from hold count: movements where at least one line is from hold and status is JOURNALED/COMPLETED
	var fromHoldCount int64
	_ = database.DB.Table("inventory_movements").
		Joins("JOIN inventory_movement_lines ON inventory_movements.id = inventory_movement_lines.movement_id").
		Where("inventory_movements.movement_type = ? AND inventory_movement_lines.is_from_hold = ?", "RTV", true).
		Distinct("inventory_movements.id").
		Count(&fromHoldCount).Error
	stats.FromHoldCount = fromHoldCount

	return renderPage(c, "rtv.html", fiber.Map{
		"Movements":     movements,
		"Products":      products,
		"TotalReturns":  stats.TotalCount,
		"PendingCount":  stats.PendingCount,
		"FromHoldCount": stats.FromHoldCount,
	})
}

// GET /wms/rtv/storages?product_id=...
func ServeRTVStoragesByProduct(c *fiber.Ctx) error {
	productID := c.Query("product_id")
	if productID == "" {
		return c.SendString(`<option value="">Select a storage lot...</option>`)
	}

	var lots []models.Storage
	err := database.DB.Preload("Locator.Warehouse").
		Where("product_id = ? AND qty_on_hand > 0", productID).
		Find(&lots).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	html := `<option value="">Select a storage lot...</option>`
	for _, lot := range lots {
		available := lot.QtyOnHand - lot.QtyReserved - lot.QtyOnHold
		html += fmt.Sprintf(
			`<option value="%s|%s" data-locator-id="%s" data-batch-number="%s" data-available="%d" data-on-hold="%d">`+
				`Locator: %s/%s | Batch: %s | Available: %d | On Hold: %d`+
				`</option>`,
			lot.LocatorID, lot.BatchNumber,
			lot.LocatorID, lot.BatchNumber, available, lot.QtyOnHold,
			lot.Locator.Warehouse.Code, lot.Locator.Code, lot.BatchNumber, available, lot.QtyOnHold,
		)
	}

	return c.SendString(html)
}

// POST /wms/rtv/new
func CreateRTV(c *fiber.Ctx) error {
	productID := c.FormValue("product_id")
	lotKey := c.FormValue("lot_key") // "locator_id|batch_number"
	isFromHoldStr := c.FormValue("is_from_hold") // "true" or "false"
	quantityStr := c.FormValue("quantity")
	remarks := c.FormValue("remarks")

	var qty int
	_, _ = fmt.Sscanf(quantityStr, "%d", &qty)

	if productID == "" || lotKey == "" || qty <= 0 {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Please select a product, storage lot, and enter a valid quantity.",
		})
	}

	parts := strings.Split(lotKey, "|")
	if len(parts) != 2 {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Invalid storage lot selected.",
		})
	}
	locatorID := parts[0]
	batchNumber := parts[1]

	isFromHold := isFromHoldStr == "true"

	// Fetch product to verify base UoM
	var product models.Product
	if err := database.DB.Preload("UoM").First(&product, "id = ?", productID).Error; err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Product not found.",
		})
	}

	userID := c.Locals("user_id").(string)

	docNo := fmt.Sprintf("RTV-%d", time.Now().UnixNano()%100000)

	movement := models.InventoryMovement{
		DocumentNo:   docNo,
		MovementType: "RTV",
		Status:       "OPEN",
		CreatedBy:    userID,
		Remarks:      remarks,
	}

	line := models.InventoryMovementLine{
		ProductID:         productID,
		FromLocatorID:     locatorID,
		BatchNumber:       batchNumber,
		RequestedQuantity: qty,
		IsFromHold:        isFromHold,
	}

	// Trigger repository creation
	err := repository.CreateInventoryMovement(&movement, []models.InventoryMovementLine{line})
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	setReloadToast(c, fmt.Sprintf("Return to Vendor ticket %s created successfully.", docNo), true)
	return c.SendStatus(fiber.StatusCreated)
}
