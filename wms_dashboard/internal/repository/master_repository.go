package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

// ==========================================
// 1. PRODUCTS MASTER CRUD
// ==========================================

func FetchAllProducts() ([]models.Product, error) {
	var products []models.Product
	err := database.DB.Preload("UoM").Order("sku ASC").Find(&products).Error
	return products, err
}

func FetchProductByID(id string) (models.Product, error) {
	var product models.Product
	err := database.DB.Preload("UoM").First(&product, "id = ?", id).Error
	return product, err
}

func CreateProduct(product *models.Product) error {
	product.ID = uuid.New().String()
	product.CreatedAt = time.Now()
	return database.DB.Create(product).Error
}

func UpdateProduct(product *models.Product) error {
	return database.DB.Save(product).Error
}

func DeleteProduct(id string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Safeguard A: Check if product has active stock on hand in the warehouse
		var stockCount int64
		err := tx.Model(&models.Storage{}).
			Where("product_id = ? AND qty_on_hand > 0", id).
			Count(&stockCount).Error
		if err != nil {
			return err
		}
		if stockCount > 0 {
			return errors.New("cannot delete product: physical stock exists on warehouse shelves")
		}

		// Safeguard B: Check if product is referenced in any historical movement lines
		var movementCount int64
		err = tx.Model(&models.InventoryMovementLine{}).
			Where("product_id = ?", id).
			Count(&movementCount).Error
		if err != nil {
			return err
		}
		if movementCount > 0 {
			return errors.New("cannot delete product: referenced in historical movement documents")
		}

		// Perform Soft Delete
		return tx.Delete(&models.Product{}, "id = ?", id).Error
	})
}

// ==========================================
// 2. WAREHOUSES MASTER CRUD
// ==========================================

func FetchAllWarehouses() ([]models.Warehouse, error) {
	var warehouses []models.Warehouse
	err := database.DB.Order("code ASC").Find(&warehouses).Error
	return warehouses, err
}

func FetchWarehouseByID(id string) (models.Warehouse, error) {
	var warehouse models.Warehouse
	err := database.DB.First(&warehouse, "id = ?", id).Error
	return warehouse, err
}

func CreateWarehouse(warehouse *models.Warehouse) error {
	warehouse.ID = uuid.New().String()
	warehouse.CreatedAt = time.Now()
	warehouse.IsActive = true
	return database.DB.Create(warehouse).Error
}

func UpdateWarehouse(warehouse *models.Warehouse) error {
	return database.DB.Save(warehouse).Error
}

func DeleteWarehouse(id string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Safeguard: Check if warehouse has child locators
		var locatorCount int64
		err := tx.Model(&models.Locator{}).
			Where("warehouse_id = ?", id).
			Count(&locatorCount).Error
		if err != nil {
			return err
		}
		if locatorCount > 0 {
			return errors.New("cannot delete warehouse: active shelf locators exist inside it")
		}

		// Perform Soft Delete
		return tx.Delete(&models.Warehouse{}, "id = ?", id).Error
	})
}

// ==========================================
// 3. LOCATORS MASTER CRUD
// ==========================================

func FetchAllLocators() ([]models.Locator, error) {
	var locators []models.Locator
	err := database.DB.Preload("Warehouse").Order("code ASC").Find(&locators).Error
	return locators, err
}

func FetchLocatorByID(id string) (models.Locator, error) {
	var locator models.Locator
	err := database.DB.Preload("Warehouse").First(&locator, "id = ?", id).Error
	return locator, err
}

func CreateLocator(locator *models.Locator) error {
	locator.ID = uuid.New().String()
	locator.CreatedAt = time.Now()
	locator.IsActive = true
	return database.DB.Create(locator).Error
}

func UpdateLocator(locator *models.Locator) error {
	return database.DB.Save(locator).Error
}

func DeleteLocator(id string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Safeguard: Check if shelf locator currently contains active stock
		var stockCount int64
		err := tx.Model(&models.Storage{}).
			Where("locator_id = ? AND qty_on_hand > 0", id).
			Count(&stockCount).Error
		if err != nil {
			return err
		}
		if stockCount > 0 {
			return errors.New("cannot delete locator: active physical stock exists on this shelf")
		}

		// Perform Soft Delete
		return tx.Delete(&models.Locator{}, "id = ?", id).Error
	})
}

// ==========================================
// 4. UOM MASTER CRUD
// ==========================================

func FetchAllUoMs() ([]models.UoM, error) {
	var uoms []models.UoM
	err := database.DB.Order("code ASC").Find(&uoms).Error
	return uoms, err
}

func FetchUoMByID(id string) (models.UoM, error) {
	var uom models.UoM
	err := database.DB.First(&uom, "id = ?", id).Error
	return uom, err
}

func CreateUoM(uom *models.UoM) error {
	uom.ID = uuid.New().String()
	uom.CreatedAt = time.Now()
	return database.DB.Create(uom).Error
}

func UpdateUoM(uom *models.UoM) error {
	return database.DB.Save(uom).Error
}

func DeleteUoM(id string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Safeguard A: Check if UoM is used as base UoM for any active product
		var prodCount int64
		err := tx.Model(&models.Product{}).
			Where("uom_id = ?", id).
			Count(&prodCount).Error
		if err != nil {
			return err
		}
		if prodCount > 0 {
			return errors.New("cannot delete unit of measure: referenced by active products in the master catalog")
		}

		// Safeguard B: Check if UoM is referenced in any conversion formulas
		var convCount int64
		err = tx.Model(&models.UoMConversion{}).
			Where("from_uom_id = ? OR to_uom_id = ?", id, id).
			Count(&convCount).Error
		if err != nil {
			return err
		}
		if convCount > 0 {
			return errors.New("cannot delete unit of measure: referenced in active conversion rules")
		}

		// Perform Soft Delete
		return tx.Delete(&models.UoM{}, "id = ?", id).Error
	})
}

// ==========================================
// 5. UOM CONVERSIONS CRUD
// ==========================================

func FetchAllConversions() ([]models.UoMConversion, error) {
	var conversions []models.UoMConversion
	err := database.DB.Preload("Product").Preload("FromUo").Preload("ToUo").
		Order("product_id ASC, from_uom_id ASC").Find(&conversions).Error
	return conversions, err
}

func FetchConversionsByProduct(productID string) ([]models.UoMConversion, error) {
	var conversions []models.UoMConversion
	err := database.DB.Preload("Product").Preload("FromUo").Preload("ToUo").
		Where("product_id = ? OR product_id = '' OR product_id IS NULL", productID).
		Find(&conversions).Error
	return conversions, err
}

func CreateConversion(conv *models.UoMConversion) error {
	conv.ID = uuid.New().String()
	conv.CreatedAt = time.Now()
	return database.DB.Create(conv).Error
}

func DeleteConversion(id string) error {
	return database.DB.Delete(&models.UoMConversion{}, "id = ?", id).Error
}
