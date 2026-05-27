package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"wms_dashboard/internal/repository"
)

// GET /wms/qc-holds
func ServeQCHolds(c *fiber.Ctx) error {
	holds, err := repository.FetchQCHolds()
	if err != nil {
		holds = nil
	}

	storages, err := repository.FetchStoragesWithAvailableStock()
	if err != nil {
		storages = nil
	}

	activeCount := 0
	totalHeldQty := 0
	for _, h := range holds {
		if h.Status == "ACTIVE" {
			activeCount++
			totalHeldQty += h.Qty
		}
	}

	reasons := []string{"DAMAGED", "INVESTIGATION", "EXPIRED", "OTHER"}

	return renderPage(c, "qc_holds.html", fiber.Map{
		"Holds":        holds,
		"Storages":     storages,
		"Reasons":      reasons,
		"ActiveCount":  activeCount,
		"TotalHeldQty": totalHeldQty,
	})
}

// POST /wms/qc-holds
func CreateQCHold(c *fiber.Ctx) error {
	storageID := c.FormValue("storage_id")
	reason := c.FormValue("reason")
	notes := c.FormValue("notes")
	userID := c.Locals("user_id").(string)

	var qty int
	fmt.Sscanf(c.FormValue("qty"), "%d", &qty)

	if storageID == "" || reason == "" || qty <= 0 {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Please select a storage lot, reason, and a valid quantity.",
		})
	}

	err := repository.CreateQCHold(storageID, qty, reason, notes, userID)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	c.Set("HX-Refresh", "true")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": "Stock successfully placed on QC Hold.",
	})
}

// POST /wms/qc-holds/:id/release
func ReleaseQCHold(c *fiber.Ctx) error {
	holdID := c.Params("id")
	userID := c.Locals("user_id").(string)

	err := repository.ReleaseQCHold(holdID, userID)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	c.Set("HX-Refresh", "true")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": "QC Hold released. Stock is now available.",
	})
}
