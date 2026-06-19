package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
	"wms_dashboard/internal/repository"
)

// ==========================================
// 1. PRODUCTS MASTER HANDLERS
// ==========================================

// GET /wms/masters/products
func ServeProductsMaster(c *fiber.Ctx) error {
	products, err := repository.FetchAllProducts()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	// If HTMX fragment load request, render only the table body rows
	if c.Get("HX-Request") == "true" && c.Query("rows_only") == "true" {
		return renderPartial(c, "partials/product_row.html", "product_rows", fiber.Map{
			"Products": products,
		})
	}

	return renderPage(c, "products_master.html", fiber.Map{
		"Products": products,
	})
}

// GET /wms/masters/products/new
func ServeNewProductForm(c *fiber.Ctx) error {
	uoms, _ := repository.FetchAllUoMs()
	return renderPartial(c, "partials/product_modal.html", "product_modal_new", fiber.Map{
		"UoMs": uoms,
	})
}

// POST /wms/masters/products
func CreateProduct(c *fiber.Ctx) error {
	sku := c.FormValue("sku")
	name := c.FormValue("name")
	description := c.FormValue("description")
	category := c.FormValue("category")
	priceStr := c.FormValue("price")
	isBundle := c.FormValue("is_bundle") == "true"

	var price float64
	if isBundle {
		price = 0.0
	} else {
		_, _ = fmt.Sscanf(priceStr, "%f", &price)
	}

	if sku == "" || name == "" {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "SKU and Product Name are required.",
		})
	}

	var unitWeight, unitVolume float64
	_, _ = fmt.Sscanf(c.FormValue("unit_weight"), "%f", &unitWeight)
	_, _ = fmt.Sscanf(c.FormValue("unit_volume"), "%f", &unitVolume)

	product := models.Product{
		SKU:         sku,
		Name:        name,
		Description: description,
		Category:    category,
		Price:       price,
		UnitWeight:  unitWeight,
		UnitVolume:  unitVolume,
		IsBundle:    isBundle,
		UoMID:       c.FormValue("uom_id"),
	}

	if err := repository.CreateProduct(&product); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Failed to create product: " + err.Error(),
		})
	}

	// Trigger dynamic client table reload
	c.Set("HX-Trigger", "refreshProductsList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": fmt.Sprintf("Product SKU %s registered successfully!", sku),
	})
}

// GET /wms/masters/products/:id/edit
func ServeEditProductForm(c *fiber.Ctx) error {
	id := c.Params("id")
	product, err := repository.FetchProductByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Product not found")
	}

	uoms, _ := repository.FetchAllUoMs()

	return renderPartial(c, "partials/product_modal.html", "product_modal_edit", fiber.Map{
		"Product": product,
		"UoMs":    uoms,
	})
}

// PUT /wms/masters/products/:id
func UpdateProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	product, err := repository.FetchProductByID(id)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Product not found.",
		})
	}

	product.SKU = c.FormValue("sku")
	product.Name = c.FormValue("name")
	product.Description = c.FormValue("description")
	product.Category = c.FormValue("category")
	product.UoMID = c.FormValue("uom_id")

	product.IsBundle = c.FormValue("is_bundle") == "true"

	var price float64
	if product.IsBundle {
		price = 0.0
	} else {
		_, _ = fmt.Sscanf(c.FormValue("price"), "%f", &price)
	}
	product.Price = price
	_, _ = fmt.Sscanf(c.FormValue("unit_weight"), "%f", &product.UnitWeight)
	_, _ = fmt.Sscanf(c.FormValue("unit_volume"), "%f", &product.UnitVolume)

	if err := repository.UpdateProduct(&product); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Failed to update: " + err.Error(),
		})
	}

	c.Set("HX-Trigger", "refreshProductsList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": fmt.Sprintf("Product %s updated successfully!", product.SKU),
	})
}

// DELETE /wms/masters/products/:id
func DeleteProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := repository.DeleteProduct(id); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	c.Set("HX-Trigger", "refreshProductsList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": "Product deleted successfully.",
	})
}

// ==========================================
// 2. WAREHOUSES MASTER HANDLERS
// ==========================================

// GET /wms/masters/warehouses
func ServeWarehousesMaster(c *fiber.Ctx) error {
	warehouses, err := repository.FetchAllWarehouses()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	if c.Get("HX-Request") == "true" && c.Query("rows_only") == "true" {
		return renderPartial(c, "partials/warehouse_row.html", "warehouse_rows", fiber.Map{
			"Warehouses": warehouses,
		})
	}

	return renderPage(c, "warehouses_master.html", fiber.Map{
		"Warehouses": warehouses,
	})
}

// GET /wms/masters/warehouses/new
func ServeNewWarehouseForm(c *fiber.Ctx) error {
	return renderPartial(c, "partials/warehouse_modal.html", "warehouse_modal_new", nil)
}

// POST /wms/masters/warehouses
func CreateWarehouse(c *fiber.Ctx) error {
	code := c.FormValue("code")
	name := c.FormValue("name")
	address := c.FormValue("address")

	if code == "" || name == "" {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Warehouse Code and Name are required.",
		})
	}

	warehouse := models.Warehouse{Code: code, Name: name, Address: address}
	if err := repository.CreateWarehouse(&warehouse); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Failed to create warehouse: " + err.Error(),
		})
	}

	c.Set("HX-Trigger", "refreshWarehousesList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": fmt.Sprintf("Warehouse %s registered successfully!", code),
	})
}

// GET /wms/masters/warehouses/:id/edit
func ServeEditWarehouseForm(c *fiber.Ctx) error {
	id := c.Params("id")
	warehouse, err := repository.FetchWarehouseByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Warehouse not found")
	}

	return renderPartial(c, "partials/warehouse_modal.html", "warehouse_modal_edit", fiber.Map{
		"Warehouse": warehouse,
	})
}

// PUT /wms/masters/warehouses/:id
func UpdateWarehouse(c *fiber.Ctx) error {
	id := c.Params("id")
	warehouse, err := repository.FetchWarehouseByID(id)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Warehouse not found.",
		})
	}

	warehouse.Code = c.FormValue("code")
	warehouse.Name = c.FormValue("name")
	warehouse.Address = c.FormValue("address")
	warehouse.IsActive = c.FormValue("is_active") == "true"

	if err := repository.UpdateWarehouse(&warehouse); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Failed to update: " + err.Error(),
		})
	}

	c.Set("HX-Trigger", "refreshWarehousesList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": fmt.Sprintf("Warehouse %s updated successfully!", warehouse.Code),
	})
}

// DELETE /wms/masters/warehouses/:id
func DeleteWarehouse(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := repository.DeleteWarehouse(id); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	c.Set("HX-Trigger", "refreshWarehousesList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": "Warehouse deleted successfully.",
	})
}

// ==========================================
// 3. LOCATORS MASTER HANDLERS
// ==========================================

// GET /wms/masters/locators
func ServeLocatorsMaster(c *fiber.Ctx) error {
	locators, err := repository.FetchAllLocators()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	occupancies, _ := repository.FetchLocatorOccupancies()
	occMap := make(map[string]repository.LocatorOccupancy, len(occupancies))
	var greenCount, amberCount, redCount int
	var totalUtil float64
	for _, o := range occupancies {
		occMap[o.LocatorID] = o
		totalUtil += o.UtilPct
		switch o.ColorBand() {
		case "red":
			redCount++
		case "amber":
			amberCount++
		default:
			greenCount++
		}
	}
	var avgUtil float64
	if len(occupancies) > 0 {
		avgUtil = totalUtil / float64(len(occupancies))
	}

	if c.Get("HX-Request") == "true" && c.Query("rows_only") == "true" {
		return renderPartial(c, "partials/locator_row.html", "locator_rows", fiber.Map{
			"Locators":     locators,
			"OccupancyMap": occMap,
		})
	}

	return renderPage(c, "locators_master.html", fiber.Map{
		"Locators":     locators,
		"Occupancies":  occupancies,
		"OccupancyMap": occMap,
		"GreenCount":   greenCount,
		"AmberCount":   amberCount,
		"RedCount":     redCount,
		"AvgUtil":      avgUtil,
	})
}

// GET /wms/masters/locators/new
func ServeNewLocatorForm(c *fiber.Ctx) error {
	var warehouses []models.Warehouse
	_ = database.DB.Find(&warehouses).Error

	return renderPartial(c, "partials/locator_modal.html", "locator_modal_new", fiber.Map{
		"Warehouses": warehouses,
	})
}

// POST /wms/masters/locators
func CreateLocator(c *fiber.Ctx) error {
	whID := c.FormValue("warehouse_id")
	zone := c.FormValue("zone")
	aisle := c.FormValue("aisle")
	shelf := c.FormValue("shelf")
	level := c.FormValue("level")

	if whID == "" || zone == "" || aisle == "" || shelf == "" || level == "" {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "All shelf locator coordinates are required.",
		})
	}

	// Fetch warehouse code to compile dynamic combined Locator Code
	var wh models.Warehouse
	if err := database.DB.First(&wh, "id = ?", whID).Error; err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Selected Warehouse not found.",
		})
	}

	code := fmt.Sprintf("%s-%s-%s-%s-%s", wh.Code, zone, aisle, shelf, level)

	var maxWeight, maxVolume float64
	_, _ = fmt.Sscanf(c.FormValue("max_weight"), "%f", &maxWeight)
	_, _ = fmt.Sscanf(c.FormValue("max_volume"), "%f", &maxVolume)

	locator := models.Locator{
		WarehouseID: whID,
		Zone:        zone,
		Aisle:       aisle,
		Shelf:       shelf,
		Level:       level,
		Code:        code,
		MaxWeight:   maxWeight,
		MaxVolume:   maxVolume,
	}

	if err := repository.CreateLocator(&locator); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Failed to create locator: " + err.Error(),
		})
	}

	c.Set("HX-Trigger", "refreshLocatorsList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": fmt.Sprintf("Locator %s registered successfully!", code),
	})
}

// GET /wms/masters/locators/:id/edit
func ServeEditLocatorForm(c *fiber.Ctx) error {
	id := c.Params("id")
	locator, err := repository.FetchLocatorByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Locator not found")
	}

	var warehouses []models.Warehouse
	_ = database.DB.Find(&warehouses).Error

	return renderPartial(c, "partials/locator_modal.html", "locator_modal_edit", fiber.Map{
		"Locator":    locator,
		"Warehouses": warehouses,
	})
}

// PUT /wms/masters/locators/:id
func UpdateLocator(c *fiber.Ctx) error {
	id := c.Params("id")
	locator, err := repository.FetchLocatorByID(id)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Locator not found.",
		})
	}

	locator.WarehouseID = c.FormValue("warehouse_id")
	locator.Zone = c.FormValue("zone")
	locator.Aisle = c.FormValue("aisle")
	locator.Shelf = c.FormValue("shelf")
	locator.Level = c.FormValue("level")
	locator.IsActive = c.FormValue("is_active") == "true"
	_, _ = fmt.Sscanf(c.FormValue("max_weight"), "%f", &locator.MaxWeight)
	_, _ = fmt.Sscanf(c.FormValue("max_volume"), "%f", &locator.MaxVolume)

	// Fetch warehouse code to update code
	var wh models.Warehouse
	if err := database.DB.First(&wh, "id = ?", locator.WarehouseID).Error; err == nil {
		locator.Code = fmt.Sprintf("%s-%s-%s-%s-%s", wh.Code, locator.Zone, locator.Aisle, locator.Shelf, locator.Level)
	}

	if err := repository.UpdateLocator(&locator); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Failed to update: " + err.Error(),
		})
	}

	c.Set("HX-Trigger", "refreshLocatorsList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": fmt.Sprintf("Locator %s updated successfully!", locator.Code),
	})
}

// DELETE /wms/masters/locators/:id
func DeleteLocator(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := repository.DeleteLocator(id); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	c.Set("HX-Trigger", "refreshLocatorsList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": "Locator deleted successfully.",
	})
}

// ==========================================
// 4. UOM MASTER HANDLERS
// ==========================================

// GET /wms/masters/uoms
func ServeUoMsMaster(c *fiber.Ctx) error {
	uoms, err := repository.FetchAllUoMs()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	conversions, err := repository.FetchAllConversions()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	if c.Get("HX-Request") == "true" && c.Query("rows_only") == "true" {
		return renderPartial(c, "partials/uom_row.html", "uom_rows", fiber.Map{
			"UoMs": uoms,
		})
	}

	if c.Get("HX-Request") == "true" && c.Query("conversions_only") == "true" {
		return renderPartial(c, "partials/uom_row.html", "conversion_rows", fiber.Map{
			"Conversions": conversions,
		})
	}

	return renderPage(c, "uoms_master.html", fiber.Map{
		"UoMs":        uoms,
		"Conversions": conversions,
	})
}

// GET /wms/masters/uoms/new
func ServeNewUoMForm(c *fiber.Ctx) error {
	return renderPartial(c, "partials/uom_modal.html", "uom_modal_new", nil)
}

// POST /wms/masters/uoms
func CreateUoM(c *fiber.Ctx) error {
	code := c.FormValue("code")
	name := c.FormValue("name")
	description := c.FormValue("description")

	if code == "" || name == "" {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Unit Code and Name are required.",
		})
	}

	uom := models.UoM{
		Code:        code,
		Name:        name,
		Description: description,
	}

	if err := repository.CreateUoM(&uom); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Failed to create UoM: " + err.Error(),
		})
	}

	c.Set("HX-Trigger", "refreshUoMsList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": fmt.Sprintf("Unit %s registered successfully!", code),
	})
}

// GET /wms/masters/uoms/:id/edit
func ServeEditUoMForm(c *fiber.Ctx) error {
	id := c.Params("id")
	uom, err := repository.FetchUoMByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Unit of measure not found")
	}

	return renderPartial(c, "partials/uom_modal.html", "uom_modal_edit", fiber.Map{
		"UoM": uom,
	})
}

// PUT /wms/masters/uoms/:id
func UpdateUoM(c *fiber.Ctx) error {
	id := c.Params("id")
	uom, err := repository.FetchUoMByID(id)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Unit of measure not found.",
		})
	}

	uom.Code = c.FormValue("code")
	uom.Name = c.FormValue("name")
	uom.Description = c.FormValue("description")

	if err := repository.UpdateUoM(&uom); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Failed to update: " + err.Error(),
		})
	}

	c.Set("HX-Trigger", "refreshUoMsList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": fmt.Sprintf("Unit %s updated successfully!", uom.Code),
	})
}

// DELETE /wms/masters/uoms/:id
func DeleteUoM(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := repository.DeleteUoM(id); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	c.Set("HX-Trigger", "refreshUoMsList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": "Unit of measure deleted successfully.",
	})
}

// ==========================================
// 5. UOM CONVERSIONS HANDLERS
// ==========================================

// GET /wms/masters/conversions/new
func ServeNewConversionForm(c *fiber.Ctx) error {
	uoms, _ := repository.FetchAllUoMs()
	products, _ := repository.FetchAllProducts()

	return renderPartial(c, "partials/uom_modal.html", "conversion_modal_new", fiber.Map{
		"UoMs":     uoms,
		"Products": products,
	})
}

// POST /wms/masters/conversions
func CreateConversion(c *fiber.Ctx) error {
	prodID := c.FormValue("product_id")
	fromUoMID := c.FormValue("from_uom_id")
	toUoMID := c.FormValue("to_uom_id")
	factorStr := c.FormValue("multiply_factor")

	var factor float64
	_, _ = fmt.Sscanf(factorStr, "%f", &factor)

	if fromUoMID == "" || toUoMID == "" || factor <= 0 {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Source, Target Units and a positive conversion factor are required.",
		})
	}

	if fromUoMID == toUoMID {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Source and Target Units must be different.",
		})
	}

	conv := models.UoMConversion{
		ProductID:      prodID,
		FromUoMID:      fromUoMID,
		ToUoMID:        toUoMID,
		MultiplyFactor: factor,
	}

	if err := repository.CreateConversion(&conv); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Failed to create conversion rule: " + err.Error(),
		})
	}

	// Dynamic triggers
	c.Set("HX-Trigger", "refreshUoMsList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": "Unit conversion rule registered successfully!",
	})
}

// DELETE /wms/masters/conversions/:id
func DeleteConversion(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := repository.DeleteConversion(id); err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	c.Set("HX-Trigger", "refreshUoMsList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
		"Success": true,
		"Message": "Conversion rule deleted successfully.",
	})
}
