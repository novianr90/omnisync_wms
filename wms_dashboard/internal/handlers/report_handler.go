package handlers

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
	"wms_dashboard/internal/repository"
)

// ExportLedgerExcel generates a beautifully formatted Excel sheet of the filtered ledger
func ExportLedgerExcel(c *fiber.Ctx) error {
	filter := repository.LedgerFilter{
		Search:     c.Query("search", ""),
		ProductSKU: c.Query("sku", ""),
		StartDate:  c.Query("start_date", ""),
		EndDate:    c.Query("end_date", ""),
		Limit:      0, // Retrieve all records
		Offset:     0,
	}

	ledgers, _, err := repository.FetchInventoryLedger(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching ledger data: " + err.Error())
	}

	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()

	sheetName := "Inventory Ledger"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error creating sheet: " + err.Error())
	}
	_ = f.DeleteSheet("Sheet1")

	// Set Page Title Block
	_ = f.SetCellValue(sheetName, "A1", "OMNISYNC WMS - INVENTORY LEDGER REPORT")
	_ = f.MergeCell(sheetName, "A1", "K1")

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4f46e5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	_ = f.SetRowHeight(sheetName, 1, 40)
	_ = f.SetCellStyle(sheetName, "A1", "K1", titleStyle)

	// Filter metadata info row
	metadataText := fmt.Sprintf("Export Date: %s | Active Filters: ", time.Now().Format("2006-01-02 15:04:05"))
	if filter.StartDate != "" || filter.EndDate != "" {
		metadataText += fmt.Sprintf("Date Range [%s to %s] ", filter.StartDate, filter.EndDate)
	}
	if filter.ProductSKU != "" {
		metadataText += fmt.Sprintf("SKU [%s] ", filter.ProductSKU)
	}
	if filter.Search != "" {
		metadataText += fmt.Sprintf("Search [%s] ", filter.Search)
	}
	if filter.StartDate == "" && filter.EndDate == "" && filter.ProductSKU == "" && filter.Search == "" {
		metadataText += "None"
	}
	_ = f.SetCellValue(sheetName, "A2", metadataText)
	_ = f.MergeCell(sheetName, "A2", "K2")
	metaStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Italic: true, Size: 9, Color: "475569"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"f1f5f9"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	_ = f.SetRowHeight(sheetName, 2, 22)
	_ = f.SetCellStyle(sheetName, "A2", "K2", metaStyle)

	// Table Headers
	headers := []string{
		"Transaction Date", "Type", "Document No", "Batch Number", 
		"SKU", "Product Name", "Locator Code", "Quantity Change", 
		"Batch Balance", "Account No", "Contra Account No",
	}
	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 4)
		_ = f.SetCellValue(sheetName, cell, h)
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"1e293b"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "bottom", Color: "475569", Style: 1},
			{Type: "top", Color: "475569", Style: 1},
		},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	_ = f.SetRowHeight(sheetName, 4, 25)
	_ = f.SetCellStyle(sheetName, "A4", "K4", headerStyle)

	// Data Rows
	rowStyleEven, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 9.5},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"ffffff"}, Pattern: 1},
		Border: []excelize.Border{{Type: "bottom", Color: "e2e8f0", Style: 1}},
	})
	rowStyleOdd, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 9.5},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"f8fafc"}, Pattern: 1},
		Border: []excelize.Border{{Type: "bottom", Color: "e2e8f0", Style: 1}},
	})

	for i, l := range ledgers {
		rowIdx := i + 5
		_ = f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIdx), l.TransactionDate.Format("2006-01-02 15:04:05"))
		_ = f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIdx), l.TransactionType)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIdx), l.DocumentNo)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIdx), l.BatchNumber)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIdx), l.Product.SKU)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIdx), l.Product.Name)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIdx), l.Locator.Code)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowIdx), l.QtyChange)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowIdx), l.BatchBalance)
		
		accNo := ""
		if l.AccountNo != nil {
			accNo = *l.AccountNo
		}
		contraAccNo := ""
		if l.ContraAccountNo != nil {
			contraAccNo = *l.ContraAccountNo
		}
		_ = f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowIdx), accNo)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowIdx), contraAccNo)

		_ = f.SetRowHeight(sheetName, rowIdx, 20)
		targetStyle := rowStyleEven
		if i%2 != 0 {
			targetStyle = rowStyleOdd
		}
		_ = f.SetCellStyle(sheetName, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("K%d", rowIdx), targetStyle)
	}

	// Auto-fit Columns
	cols, _ := f.GetCols(sheetName)
	for i, col := range cols {
		maxLen := 0
		for _, cellVal := range col {
			if len(cellVal) > maxLen {
				maxLen = len(cellVal)
			}
		}
		colName, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheetName, colName, colName, float64(maxLen+4))
	}

	f.SetActiveSheet(index)

	c.Set("Content-Disposition", "attachment; filename=inventory_ledger_"+time.Now().Format("20060102")+".xlsx")
	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error writing excel file: " + err.Error())
	}
	return c.Send(buf.Bytes())
}

// ExportLedgerPDF generates a styled PDF report of the filtered ledger
func ExportLedgerPDF(c *fiber.Ctx) error {
	filter := repository.LedgerFilter{
		Search:     c.Query("search", ""),
		ProductSKU: c.Query("sku", ""),
		StartDate:  c.Query("start_date", ""),
		EndDate:    c.Query("end_date", ""),
		Limit:      0,
		Offset:     0,
	}

	ledgers, _, err := repository.FetchInventoryLedger(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching ledger data: " + err.Error())
	}

	// Create new Landscape PDF on A4
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 15)

	// Define Header and Footer
	pdf.SetHeaderFunc(func() {
		// Omnisync Brand Header Block
		pdf.SetFillColor(79, 70, 229) // Indigo Accent
		pdf.Rect(10, 10, 277, 20, "F")
		
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Arial", "B", 14)
		pdf.CellFormat(277, 20, "   OMNISYNC WMS - INVENTORY LEDGER REPORT", "", 0, "L", false, 0, "")
		pdf.Ln(25)
	})

	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(100, 116, 139)
		pdf.CellFormat(0, 10, fmt.Sprintf("Page %d of {nb}", pdf.PageNo()), "", 0, "C", false, 0, "")
		pdf.CellFormat(0, 10, "Omnisync WMS Suite", "", 0, "R", false, 0, "")
	})

	pdf.AliasNbPages("")
	pdf.AddPage()

	// Metadata Panel
	pdf.SetFillColor(241, 245, 249) // Light Slate
	pdf.Rect(10, 32, 277, 18, "F")
	
	pdf.SetTextColor(71, 85, 105)
	pdf.SetFont("Arial", "", 8.5)
	
	metaLeft := fmt.Sprintf("Export Time: %s", time.Now().Format("2006-01-02 15:04:05"))
	metaRight := "Active Filters: "
	if filter.StartDate != "" || filter.EndDate != "" {
		metaRight += fmt.Sprintf("Dates [%s to %s] ", filter.StartDate, filter.EndDate)
	}
	if filter.ProductSKU != "" {
		metaRight += fmt.Sprintf("SKU [%s] ", filter.ProductSKU)
	}
	if filter.Search != "" {
		metaRight += fmt.Sprintf("Search [%s] ", filter.Search)
	}
	if filter.StartDate == "" && filter.EndDate == "" && filter.ProductSKU == "" && filter.Search == "" {
		metaRight += "None"
	}

	pdf.SetXY(12, 33)
	pdf.Cell(130, 8, metaLeft)
	pdf.SetXY(12, 39)
	pdf.Cell(270, 8, metaRight)
	pdf.Ln(15)

	// Table Headers
	headers := []struct {
		Name  string
		Width float64
	}{
		{"Date/Time", 35},
		{"Type", 22},
		{"Document No", 32},
		{"Batch No", 28},
		{"SKU", 30},
		{"Product Name", 42},
		{"Locator", 30},
		{"Qty", 15},
		{"Balance", 15},
		{"Acc", 14},
		{"Contra", 14},
	}

	pdf.SetFillColor(30, 41, 59) // Slate-900 Header Fill
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8)
	
	pdf.SetX(10)
	for _, h := range headers {
		pdf.CellFormat(h.Width, 7, h.Name, "1", 0, "L", true, 0, "")
	}
	pdf.Ln(7)

	// Render Data Rows
	pdf.SetFont("Arial", "", 7.5)
	pdf.SetTextColor(51, 65, 85)

	for i, l := range ledgers {
		// Alternating row background colors
		if i%2 == 0 {
			pdf.SetFillColor(255, 255, 255)
		} else {
			pdf.SetFillColor(248, 250, 252) // slate-50
		}

		accNo := ""
		if l.AccountNo != nil {
			accNo = *l.AccountNo
		}
		contraAcc := ""
		if l.ContraAccountNo != nil {
			contraAcc = *l.ContraAccountNo
		}

		// Pre-process variables to prevent overflow or truncation
		qtyStr := fmt.Sprintf("%+d", l.QtyChange)
		balStr := strconv.Itoa(l.BatchBalance)

		pdf.SetX(10)
		pdf.CellFormat(35, 6.5, l.TransactionDate.Format("2006-01-02 15:04:05"), "1", 0, "L", true, 0, "")
		pdf.CellFormat(22, 6.5, l.TransactionType, "1", 0, "L", true, 0, "")
		pdf.CellFormat(32, 6.5, l.DocumentNo, "1", 0, "L", true, 0, "")
		pdf.CellFormat(28, 6.5, l.BatchNumber, "1", 0, "L", true, 0, "")
		pdf.CellFormat(30, 6.5, l.Product.SKU, "1", 0, "L", true, 0, "")
		
		// Clip or shorten product name to avoid wrapping layout failures
		prodName := l.Product.Name
		if len(prodName) > 22 {
			prodName = prodName[:19] + "..."
		}
		pdf.CellFormat(42, 6.5, prodName, "1", 0, "L", true, 0, "")
		pdf.CellFormat(30, 6.5, l.Locator.Code, "1", 0, "L", true, 0, "")
		pdf.CellFormat(15, 6.5, qtyStr, "1", 0, "R", true, 0, "")
		pdf.CellFormat(15, 6.5, balStr, "1", 0, "R", true, 0, "")
		pdf.CellFormat(14, 6.5, accNo, "1", 0, "C", true, 0, "")
		pdf.CellFormat(14, 6.5, contraAcc, "1", 0, "C", true, 0, "")
		pdf.Ln(6.5)
	}

	c.Set("Content-Disposition", "attachment; filename=inventory_ledger_"+time.Now().Format("20060102")+".pdf")
	c.Set("Content-Type", "application/pdf")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error generating PDF: " + err.Error())
	}
	return c.Send(buf.Bytes())
}
