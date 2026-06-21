package handlers

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"wms_dashboard/internal/repository"
)

func ServeCycleCounts(c *fiber.Ctx) error {
	counts, err := repository.FetchCycleCounts()
	if err != nil {
		slog.Error("Failed to fetch cycle counts", slog.Any("error", err))
		return c.Status(500).SendString("Internal Server Error")
	}

	return renderPage(c, "wms_cycle_counts.html", fiber.Map{
		"Title":       "Cycle Counting",
		"Counts":      counts,
		"CurrentPath": c.Path(),
	})
}

func ServeNewCycleCountForm(c *fiber.Ctx) error {
	locators, err := repository.FetchLocatorsWithStock()
	if err != nil {
		slog.Error("Failed to fetch locators", slog.Any("error", err))
		return c.Status(500).SendString("Internal Server Error")
	}

	return renderPage(c, "wms_cycle_count_new.html", fiber.Map{
		"Title":       "New Cycle Count",
		"Locators":    locators,
		"CurrentPath": "/wms/cycle-counts",
	})
}

func CreateCycleCount(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	type Request struct {
		LocatorIDs []string `json:"locator_ids"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	if len(req.LocatorIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "At least one locator must be selected"})
	}

	count, err := repository.CreateCycleCount(userID, req.LocatorIDs)
	if err != nil {
		slog.Error("Failed to create cycle count", slog.Any("error", err))
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"id":      count.ID,
		"message": "Cycle count created and stock frozen",
	})
}

func ServeCycleCountDetail(c *fiber.Ctx) error {
	id := c.Params("id")
	count, err := repository.GetCycleCountByID(id)
	if err != nil {
		slog.Error("Failed to fetch cycle count", slog.Any("error", err))
		return c.Status(404).SendString("Cycle count not found")
	}

	// Check if user is admin
	isAdmin := false
	userRole, ok := c.Locals("user_role").(string)
	if ok && strings.Contains(strings.ToLower(userRole), "admin") {
		isAdmin = true
	}

	return renderPage(c, "wms_cycle_count_detail.html", fiber.Map{
		"Title":       "Cycle Count " + count.DocumentNo,
		"Count":       count,
		"CurrentPath": "/wms/cycle-counts",
		"IsAdmin":     isAdmin,
	})
}

func UpdateCountSheet(c *fiber.Ctx) error {
	id := c.Params("id")

	type Request struct {
		LineID     string `json:"line_id"`
		CountedQty *int   `json:"counted_qty"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	if req.CountedQty == nil {
		return c.Status(400).JSON(fiber.Map{"error": "counted_qty is required"})
	}

	// Make sure the line belongs to this count sheet
	count, err := repository.GetCycleCountByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Cycle count not found"})
	}

	if count.Status != "IN_PROGRESS" {
		return c.Status(400).JSON(fiber.Map{"error": "Only IN_PROGRESS counts can be updated"})
	}

	err = repository.UpdateCycleCountLine(req.LineID, *req.CountedQty)
	if err != nil {
		slog.Error("Failed to update cycle count line", slog.Any("error", err))
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Line updated",
	})
}

func ReconcileCycleCount(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	id := c.Params("id")

	// First verify all lines are counted
	count, err := repository.GetCycleCountByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Cycle count not found"})
	}

	for _, line := range count.Lines {
		if line.CountedQty == nil {
			return c.Status(400).JSON(fiber.Map{"error": "All lines must be counted before reconciling"})
		}
	}

	adjID, err := repository.ReconcileCycleCount(id, userID)
	if err != nil {
		slog.Error("Failed to reconcile cycle count", slog.Any("error", err))
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// If an adjustment was generated, journal it
	if adjID != "" {
		if err := repository.JournalizeInventoryAdjustment(adjID); err != nil {
			slog.Error("Failed to journal generated adjustment", slog.Any("error", err))
			return c.Status(500).JSON(fiber.Map{"error": "Count reconciled, but adjustment failed to journal: " + err.Error()})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Cycle count reconciled successfully",
	})
}

func CancelCycleCount(c *fiber.Ctx) error {
	id := c.Params("id")

	err := repository.CancelCycleCount(id)
	if err != nil {
		slog.Error("Failed to cancel cycle count", slog.Any("error", err))
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Cycle count canceled successfully",
	})
}

func UpdateCycleCountStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	type Request struct {
		Status string `json:"status"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	if req.Status == "" {
		return c.Status(400).JSON(fiber.Map{"error": "status is required"})
	}

	// Verify the cycle count exists
	count, err := repository.GetCycleCountByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Cycle count not found"})
	}

	// Basic validation of valid transitions
	if req.Status == "IN_PROGRESS" && count.Status != "CREATED" {
		return c.Status(400).JSON(fiber.Map{"error": "Only CREATED counts can be started"})
	}
	if req.Status == "COMPLETED" && count.Status != "RECONCILED" {
		return c.Status(400).JSON(fiber.Map{"error": "Only RECONCILED counts can be completed"})
	}

	err = repository.UpdateCycleCountStatus(id, req.Status)
	if err != nil {
		slog.Error("Failed to update cycle count status", slog.Any("error", err))
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Cycle count status updated",
	})
}
