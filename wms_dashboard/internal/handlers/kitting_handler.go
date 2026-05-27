package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
	"wms_dashboard/internal/repository"
)

// GET /wms/kitting
func ServeKittingOrders(c *fiber.Ctx) error {
	var kittings []models.InventoryKitting
	_ = database.DB.Preload("FinishedProduct.UoM").Preload("FinishedLocator.Warehouse").
		Preload("ComponentLines.Product.UoM").Preload("ComponentLines.Locator").
		Order("updated_at DESC").Find(&kittings).Error

	var locators []models.Locator
	_ = database.DB.Preload("Warehouse").Find(&locators).Error

	var products []models.Product
	_ = database.DB.Preload("UoM").Find(&products).Error

	return renderPage(c, "kitting.html", fiber.Map{
		"Kittings": kittings,
		"Locators": locators,
		"Products": products,
	})
}

// GET /wms/kitting/locators?product_id=...
func ServeKittingLocatorsByProduct(c *fiber.Ctx) error {
	productID := c.Query("product_id")
	if productID == "" {
		return c.SendString(`<option value="">Select source locator...</option>`)
	}

	var lots []models.Storage
	_ = database.DB.Preload("Locator.Warehouse").
		Where("product_id = ? AND (qty_on_hand - qty_reserved - qty_on_hold) > 0", productID).
		Find(&lots).Error

	if len(lots) == 0 {
		return c.SendString(`<option value="">No stock available for this product</option>`)
	}

	// Distinct locators
	locatorMap := make(map[string]models.Locator)
	for _, lot := range lots {
		locatorMap[lot.LocatorID] = lot.Locator
	}

	html := `<option value="">Select source locator...</option>`
	for _, loc := range locatorMap {
		html += fmt.Sprintf(`<option value="%s">%s / %s</option>`, loc.ID, loc.Warehouse.Code, loc.Code)
	}

	return c.SendString(html)
}

type KittingReq struct {
	FinishedProductID string   `form:"finished_product_id"`
	FinishedLocatorID string   `form:"finished_locator_id"`
	FinishedQty       int      `form:"finished_qty"`
	Remarks           string   `form:"remarks"`
	CompProductID     []string `form:"comp_product_id[]"`
	CompLocatorID     []string `form:"comp_locator_id[]"`
	CompQty           []int    `form:"comp_qty[]"`
}

// POST /wms/kitting/new
func CreateKittingOrder(c *fiber.Ctx) error {
	req := KittingReq{
		FinishedProductID: c.FormValue("finished_product_id"),
		FinishedLocatorID: c.FormValue("finished_locator_id"),
		Remarks:           c.FormValue("remarks"),
	}
	
	fmt.Sscanf(c.FormValue("finished_qty"), "%d", &req.FinishedQty)

	// Manual parsing for array inputs
	args := c.Request().PostArgs()
	if args != nil {
		for _, arg := range args.PeekMulti("comp_product_id[]") {
			req.CompProductID = append(req.CompProductID, string(arg))
		}
		for _, arg := range args.PeekMulti("comp_locator_id[]") {
			req.CompLocatorID = append(req.CompLocatorID, string(arg))
		}
		for _, arg := range args.PeekMulti("comp_qty[]") {
			var qty int
			fmt.Sscanf(string(arg), "%d", &qty)
			req.CompQty = append(req.CompQty, qty)
		}
	}

	if req.FinishedProductID == "" || req.FinishedLocatorID == "" || req.FinishedQty <= 0 {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Please select a finished product, locator, and valid quantity.",
		})
	}

	if len(req.CompProductID) == 0 {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "You must add at least one component.",
		})
	}

	userID := c.Locals("user_id").(string)
	docNo := fmt.Sprintf("KIT-%d", time.Now().UnixNano()%1000000)

	kit := models.InventoryKitting{
		DocumentNo:        docNo,
		Status:            "OPEN",
		FinishedProductID: req.FinishedProductID,
		FinishedLocatorID: req.FinishedLocatorID,
		FinishedQty:       req.FinishedQty,
		Remarks:           req.Remarks,
		CreatedBy:         userID,
	}

	var lines []models.InventoryKittingLine
	for i := range req.CompProductID {
		if req.CompProductID[i] == "" || req.CompLocatorID[i] == "" || req.CompQty[i] <= 0 {
			return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
				"Success": false,
				"Message": "Component details are incomplete or quantity is zero.",
			})
		}
		lines = append(lines, models.InventoryKittingLine{
			ProductID:   req.CompProductID[i],
			LocatorID:   req.CompLocatorID[i],
			ConsumedQty: req.CompQty[i],
		})
	}

	err := repository.CreateKittingOrder(&kit, lines)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	c.Set("HX-Refresh", "true")
	return c.SendStatus(fiber.StatusCreated)
}

// POST /wms/kitting/:id/journal
func JournalKittingOrder(c *fiber.Ctx) error {
	id := c.Params("id")
	err := repository.JournalizeKittingOrder(id)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	c.Set("HX-Refresh", "true")
	return c.SendStatus(fiber.StatusOK)
}

// POST /wms/kitting/:id/reject
func RejectKittingOrder(c *fiber.Ctx) error {
	id := c.Params("id")
	reason := c.FormValue("rejection_reason")
	if reason == "" {
		reason = "Cancelled by user"
	}

	err := repository.RejectKittingOrder(id, reason)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	c.Set("HX-Refresh", "true")
	return c.SendStatus(fiber.StatusOK)
}
