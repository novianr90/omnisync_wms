package handlers

import (
	"bytes"
	"html/template"
	"math"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"wms_dashboard/internal/repository"
)

// ServeLedger renders the main Inventory Ledger page
func ServeLedger(c *fiber.Ctx) error {
	// Parse filters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit := 50
	offset := (page - 1) * limit

	filter := repository.LedgerFilter{
		Search:     c.Query("search", ""),
		ProductSKU: c.Query("sku", ""),
		DocumentNo: c.Query("doc_no", ""),
		StartDate:  c.Query("start_date", ""),
		EndDate:    c.Query("end_date", ""),
		Limit:      limit,
		Offset:     offset,
	}

	ledgers, total, err := repository.FetchInventoryLedger(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching ledger: " + err.Error())
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	data := fiber.Map{
		"Role":        c.Locals("user_role"),
		"Ledgers":     ledgers,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Filter":      filter,
	}

	if c.Get("HX-Request") == "true" {
		// HTMX requested just the table partial
		if c.Get("HX-Target") == "ledger-table-container" {
			fp := filepath.Join("web", "templates", "pages", "ledger.html")
			tmpl, err := template.ParseFiles(fp)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
			}
			var buf bytes.Buffer
			if err := tmpl.ExecuteTemplate(&buf, "ledger_table", data); err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
			}
			return c.Type("html").Send(buf.Bytes())
		}
	}

	// For full page or HTMX full content swap, use renderPage
	return renderPage(c, "ledger.html", data)
}
