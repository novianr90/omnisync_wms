package repository

import (
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

// LedgerFilter defines the parameters for querying the ledger
type LedgerFilter struct {
	Search     string
	ProductSKU string
	DocumentNo string
	StartDate  string
	EndDate    string
	Limit      int
	Offset     int
}

// FetchInventoryLedger retrieves a paginated list of ledger entries based on filters
func FetchInventoryLedger(filter LedgerFilter) ([]models.InventoryLedger, int64, error) {
	var ledgers []models.InventoryLedger
	var total int64

	query := database.DB.Model(&models.InventoryLedger{}).
		Preload("Product").
		Preload("Locator").
		Preload("Account").
		Preload("ContraAccount")

	// Base search joins Product if necessary
	if filter.Search != "" || filter.ProductSKU != "" {
		query = query.Joins("LEFT JOIN products ON products.id = inventory_ledgers.product_id")
	}

	if filter.Search != "" {
		query = query.Where("inventory_ledgers.document_no LIKE ? OR inventory_ledgers.batch_number LIKE ? OR products.sku LIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}
	if filter.ProductSKU != "" {
		query = query.Where("products.sku = ?", filter.ProductSKU)
	}
	if filter.DocumentNo != "" {
		query = query.Where("inventory_ledgers.document_no = ?", filter.DocumentNo)
	}
	if filter.StartDate != "" {
		query = query.Where("inventory_ledgers.transaction_date >= ?", filter.StartDate)
	}
	if filter.EndDate != "" {
		query = query.Where("inventory_ledgers.transaction_date <= ?", filter.EndDate+" 23:59:59")
	}

	// Count total records matching filter
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and ordering
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	query = query.Order("inventory_ledgers.transaction_date DESC")

	err := query.Find(&ledgers).Error
	return ledgers, total, err
}
