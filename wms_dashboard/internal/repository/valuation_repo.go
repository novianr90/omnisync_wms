package repository

import (
	"time"

	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

// FIFOValuationRow represents one storage lot with its FIFO cost valuation
type FIFOValuationRow struct {
	ProductID      string
	SKU            string
	ProductName    string
	Category       string
	WarehouseCode  string
	WarehouseName  string
	LocatorCode    string
	BatchNumber    string
	ReceivedAt     time.Time
	QtyOnHand      int
	UnitCost       float64
	TotalValue     float64
	AgingDays      int
	AgingBucket    string // "0-30", "31-90", "91+"
}

// ValuationSummary aggregates totals for the summary stats bar
type ValuationSummary struct {
	TotalValue    float64
	TotalQty      int
	Bucket0_30    float64
	Bucket31_90   float64
	Bucket91Plus  float64
	UniqueProducts int
}

// ValuationFilter controls which records to return
type ValuationFilter struct {
	WarehouseID string
	Category    string
	AsBucket    string // "0-30", "31-90", "91+", or empty = all
}

// FetchFIFOValuation returns per-batch valuation rows. Unit cost comes from the
// first (oldest) INBOUND ledger entry for that product+batch combination.
// If no ledger cost exists we fall back to products.price.
func FetchFIFOValuation(filter ValuationFilter) ([]FIFOValuationRow, ValuationSummary, error) {
	type rawRow struct {
		ProductID     string
		SKU           string
		ProductName   string
		Category      string
		WarehouseCode string
		WarehouseName string
		LocatorCode   string
		BatchNumber   string
		ReceivedAt    time.Time
		QtyOnHand     int
		ProductPrice  float64
		// earliest inbound unit cost from ledger (may be 0 if no ledger row)
		LedgerUnitCost float64
	}

	var rows []rawRow

	// Pull storages joined with locator/warehouse/product.
	// LEFT JOIN to the earliest INBOUND ledger entry to get the receipt unit cost.
	// SQLite compatible sub-select pattern.
	query := database.DB.
		Table("storages s").
		Select(`
			p.id          AS product_id,
			p.sku         AS sku,
			p.name        AS product_name,
			p.category    AS category,
			w.code        AS warehouse_code,
			w.name        AS warehouse_name,
			l.code        AS locator_code,
			s.batch_number,
			s.received_at,
			s.qty_on_hand,
			p.price       AS product_price,
			COALESCE(lc.unit_cost, 0) AS ledger_unit_cost
		`).
		Joins("JOIN products p ON p.id = s.product_id AND p.deleted_at IS NULL").
		Joins("JOIN locators l ON l.id = s.locator_id").
		Joins("JOIN warehouses w ON w.id = l.warehouse_id").
		Joins(`LEFT JOIN (
			SELECT product_id, batch_number,
			       CAST(SUM(qty_change) AS REAL) as inbound_qty,
			       0.0 as unit_cost
			FROM inventory_ledgers
			WHERE transaction_type = 'INBOUND'
			GROUP BY product_id, batch_number
		) lc ON lc.product_id = s.product_id AND lc.batch_number = s.batch_number`).
		Where("s.qty_on_hand > 0")

	if filter.WarehouseID != "" {
		query = query.Where("w.id = ?", filter.WarehouseID)
	}
	if filter.Category != "" {
		query = query.Where("p.category = ?", filter.Category)
	}

	if err := query.Order("p.name, s.received_at ASC").Scan(&rows).Error; err != nil {
		return nil, ValuationSummary{}, err
	}

	now := time.Now()
	result := make([]FIFOValuationRow, 0, len(rows))
	var summary ValuationSummary
	summary.UniqueProducts = 0
	seenProducts := map[string]bool{}

	for _, r := range rows {
		days := int(now.Sub(r.ReceivedAt).Hours() / 24)
		bucket := agingBucket(days)

		if filter.AsBucket != "" && filter.AsBucket != bucket {
			continue
		}

		// Prefer ledger cost; fall back to product price
		unitCost := r.LedgerUnitCost
		if unitCost == 0 {
			unitCost = r.ProductPrice
		}
		totalValue := unitCost * float64(r.QtyOnHand)

		row := FIFOValuationRow{
			ProductID:     r.ProductID,
			SKU:           r.SKU,
			ProductName:   r.ProductName,
			Category:      r.Category,
			WarehouseCode: r.WarehouseCode,
			WarehouseName: r.WarehouseName,
			LocatorCode:   r.LocatorCode,
			BatchNumber:   r.BatchNumber,
			ReceivedAt:    r.ReceivedAt,
			QtyOnHand:     r.QtyOnHand,
			UnitCost:      unitCost,
			TotalValue:    totalValue,
			AgingDays:     days,
			AgingBucket:   bucket,
		}
		result = append(result, row)

		summary.TotalValue += totalValue
		summary.TotalQty += r.QtyOnHand
		switch bucket {
		case "0-30":
			summary.Bucket0_30 += totalValue
		case "31-90":
			summary.Bucket31_90 += totalValue
		case "91+":
			summary.Bucket91Plus += totalValue
		}
		if !seenProducts[r.ProductID] {
			seenProducts[r.ProductID] = true
			summary.UniqueProducts++
		}
	}

	return result, summary, nil
}

func agingBucket(days int) string {
	switch {
	case days <= 30:
		return "0-30"
	case days <= 90:
		return "31-90"
	default:
		return "91+"
	}
}

// FetchValuationWarehouses returns distinct warehouses that have stock
func FetchValuationWarehouses() ([]models.Warehouse, error) {
	var warehouses []models.Warehouse
	err := database.DB.
		Joins("JOIN locators l ON l.warehouse_id = warehouses.id").
		Joins("JOIN storages s ON s.locator_id = l.id AND s.qty_on_hand > 0").
		Where("warehouses.deleted_at IS NULL").
		Group("warehouses.id").
		Find(&warehouses).Error
	return warehouses, err
}

// FetchValuationCategories returns distinct product categories with stock
func FetchValuationCategories() ([]string, error) {
	var categories []string
	err := database.DB.
		Model(&models.Product{}).
		Select("DISTINCT category").
		Joins("JOIN storages s ON s.product_id = products.id AND s.qty_on_hand > 0").
		Where("products.deleted_at IS NULL AND products.category != ''").
		Pluck("category", &categories).Error
	return categories, err
}
