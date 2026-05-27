package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
	"wms_dashboard/internal/repository"
)

// GET /wms/adjustments
func ServeAdjustments(c *fiber.Ctx) error {
	var adjustments []models.InventoryAdjustment
	_ = database.DB.Preload("Lines.Product.UoM").Preload("Lines.Locator.Warehouse").
		Order("updated_at DESC").Find(&adjustments).Error

	var locators []models.Locator
	_ = database.DB.Preload("Warehouse").Find(&locators).Error

	var products []models.Product
	_ = database.DB.Preload("UoM").Find(&products).Error

	return renderPage(c, "adjustments.html", fiber.Map{
		"Adjustments": adjustments,
		"Locators":    locators,
		"Products":    products,
	})
}

// GET /wms/adjustments/locators?product_id=...
func ServeAdjustmentLocatorsByProduct(c *fiber.Ctx) error {
	productID := c.Query("product_id")
	if productID == "" {
		return c.SendString(`<option value="">Select a locator...</option>`)
	}

	// 1. Get locators with stock for this product
	var lots []models.Storage
	_ = database.DB.Preload("Locator.Warehouse").
		Where("product_id = ? AND qty_on_hand > 0", productID).
		Find(&lots).Error

	stockLocators := make(map[string]models.Locator)
	for _, lot := range lots {
		stockLocators[lot.LocatorID] = lot.Locator
	}

	// 2. Get ALL locators
	var allLocators []models.Locator
	_ = database.DB.Preload("Warehouse").Find(&allLocators).Error

	html := `<option value="">Select a locator...</option>`
	
	if len(stockLocators) > 0 {
		html += `<optgroup label="Locations with Stock">`
		for _, loc := range stockLocators {
			html += fmt.Sprintf(`<option value="%s">%s / %s</option>`, loc.ID, loc.Warehouse.Code, loc.Code)
		}
		html += `</optgroup>`
	}

	html += `<optgroup label="All Locations">`
	for _, loc := range allLocators {
		if _, exists := stockLocators[loc.ID]; !exists {
			html += fmt.Sprintf(`<option value="%s">%s / %s</option>`, loc.ID, loc.Warehouse.Code, loc.Code)
		}
	}
	html += `</optgroup>`

	return c.SendString(html)
}

// POST /wms/adjustments/new
func CreateAdjustment(c *fiber.Ctx) error {
	productID := c.FormValue("product_id")
	locatorID := c.FormValue("locator_id")
	qtyDeltaStr := c.FormValue("qty_delta")
	reasonCode := c.FormValue("reason_code")
	remarks := c.FormValue("remarks")

	var qtyDelta int
	_, _ = fmt.Sscanf(qtyDeltaStr, "%d", &qtyDelta)

	if productID == "" || locatorID == "" || qtyDelta == 0 {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Please select a product, locator, and enter a non-zero quantity.",
		})
	}

	// Auto-correct the sign based on the reason code
	switch reasonCode {
	case "DAMAGED", "LOST", "EXPIRED":
		if qtyDelta > 0 {
			qtyDelta = -qtyDelta
		}
	case "FOUND":
		if qtyDelta < 0 {
			qtyDelta = -qtyDelta
		}
	case "STOCK_TAKE":
		// Leave as is (could be + or -)
	}

	userID := c.Locals("user_id").(string)
	docNo := fmt.Sprintf("ADJ-%d", time.Now().UnixNano()%1000000)

	adj := models.InventoryAdjustment{
		DocumentNo: docNo,
		Status:     "OPEN",
		ReasonCode: reasonCode,
		Remarks:    remarks,
		CreatedBy:  userID,
	}

	line := models.InventoryAdjustmentLine{
		ProductID: productID,
		LocatorID: locatorID,
		QtyDelta:  qtyDelta,
	}

	err := repository.CreateInventoryAdjustment(&adj, []models.InventoryAdjustmentLine{line})
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	c.Set("HX-Refresh", "true")
	return c.SendStatus(fiber.StatusCreated)
}

// POST /wms/adjustments/:id/journal
func JournalAdjustment(c *fiber.Ctx) error {
	id := c.Params("id")
	err := repository.JournalizeInventoryAdjustment(id)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	c.Set("HX-Refresh", "true")
	return c.SendStatus(fiber.StatusOK)
}

// POST /wms/adjustments/:id/reject
func RejectAdjustment(c *fiber.Ctx) error {
	id := c.Params("id")
	reason := c.FormValue("rejection_reason")
	if reason == "" {
		reason = "Cancelled by user"
	}

	err := repository.RejectInventoryAdjustment(id, reason)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	c.Set("HX-Refresh", "true")
	return c.SendStatus(fiber.StatusOK)
}
