package handlers

import (
	"github.com/gofiber/fiber/v2"
	"wms_dashboard/internal/repository"
)

// ServeInProgressDocs renders the In-Progress Documents page
func ServeInProgressDocs(c *fiber.Ctx) error {
	docType := c.Query("type", "All")
	search := c.Query("search", "")

	docs, err := repository.FetchInProgressDocs(docType, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching in-progress documents: " + err.Error())
	}

	data := fiber.Map{
		"Docs":    docs,
		"DocType": docType,
		"Search":  search,
	}

	if c.Get("HX-Request") == "true" && c.Query("rows_only") == "true" {
		return renderPartial(c, "partials/in_progress_rows.html", "in_progress_rows", data)
	}

	return renderPage(c, "in_progress.html", data)
}
