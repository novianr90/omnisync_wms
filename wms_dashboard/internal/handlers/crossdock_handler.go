package handlers

import (
	"github.com/gofiber/fiber/v2"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

func ServeCrossDock(c *fiber.Ctx) error {
	var movements []models.InventoryMovement
	
	// Fetch inbound movements that are flagged for cross-docking and are not yet completed
	err := database.DB.Preload("Lines.Product").
		Where("is_cross_dock = ? AND movement_type = ? AND status != ?", true, "INBOUND", "COMPLETED").
		Order("created_at DESC").
		Find(&movements).Error

	if err != nil {
		return c.Status(500).SendString("Error fetching cross dock activities")
	}

	return renderPage(c, "crossdock.html", fiber.Map{
		"Title": "Cross Docking Dashboard",
		"ActiveTransactions": movements,
	})
}
