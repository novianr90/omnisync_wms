package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
	"wms_dashboard/internal/repository"
)

// Helper to render templates with layout
func renderPage(c *fiber.Ctx, pageTemplate string, data fiber.Map) error {
	// Add user locals context to template variables
	if data == nil {
		data = fiber.Map{}
	}
	data["Username"] = c.Locals("user_name")
	data["UserRole"] = c.Locals("user_role")
	data["UserEmail"] = c.Locals("user_email")

	lp := filepath.Join("web", "templates", "layouts", "base.html")
	fp := filepath.Join("web", "templates", "pages", pageTemplate)

	var tmpl *template.Template
	var err error
	var execTmpl string

	if c.Get("HX-Request") == "true" {
		tmpl = template.New(pageTemplate)
		tmpl, err = tmpl.ParseFiles(fp)
		execTmpl = "content"
	} else {
		tmpl = template.New("base")
		tmpl, err = tmpl.ParseFiles(lp, fp)
		execTmpl = "base"
	}

	if err != nil {
		log.Printf("Template parsing error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading template: " + err.Error())
	}

	// Dynamically parse all HTML partials in the partials folder
	partialsGlob := filepath.Join("web", "templates", "partials", "*.html")
	tmpl, err = tmpl.ParseGlob(partialsGlob)
	if err != nil {
		log.Printf("Glob parsing warning: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, execTmpl, data); err != nil {
		log.Printf("Template execution error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error rendering page: " + err.Error())
	}

	c.Set("Content-Type", "text/html")
	return c.Send(buf.Bytes())
}

// Helper to render partial html files (no base layout)
func renderPartial(c *fiber.Ctx, partialPath string, templateName string, data fiber.Map) error {
	if data == nil {
		data = fiber.Map{}
	}
	data["Username"] = c.Locals("user_name")
	data["UserRole"] = c.Locals("user_role")
	data["UserEmail"] = c.Locals("user_email")

	fp := filepath.Join("web", "templates", partialPath)
	tmpl, err := template.ParseFiles(fp)
	if err != nil {
		log.Printf("Partial parsing error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error: " + err.Error())
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		log.Printf("Partial execution error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error: " + err.Error())
	}

	c.Set("Content-Type", "text/html")
	return c.Send(buf.Bytes())
}

// Helper to set a cookie toast for page reloads (HX-Refresh)
func setReloadToast(c *fiber.Ctx, message string, isSuccess bool) {
	toastType := "error"
	if isSuccess {
		toastType = "success"
	}
	c.Cookie(&fiber.Cookie{
		Name:     "toast_msg",
		Value:    message,
		HTTPOnly: false, // JS needs to read it
		Path:     "/",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "toast_type",
		Value:    toastType,
		HTTPOnly: false, // JS needs to read it
		Path:     "/",
	})
	c.Set("HX-Refresh", "true")
}

// GET /login
func ServeLogin(c *fiber.Ctx) error {
	// If already authenticated, redirect to dashboard
	if c.Cookies("jwt_token") != "" {
		return c.Redirect("/")
	}

	fp := filepath.Join("web", "templates", "pages", "login.html")
	tmpl, err := template.ParseFiles(fp)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	var buf bytes.Buffer
	_ = tmpl.Execute(&buf, nil)
	c.Set("Content-Type", "text/html")
	return c.Send(buf.Bytes())
}

// POST /auth/login-submit - Bridges credentials to auth_services (Port 8000)
func HandleLogin(c *fiber.Ctx) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	// Call separate auth_services backend
	authReqBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})

	authAPIUrl := os.Getenv("AUTH_API_URL")
	if authAPIUrl == "" {
		authAPIUrl = "http://localhost:8000"
	}
	loginUrl := fmt.Sprintf("%s/auth/login", authAPIUrl)
	resp, err := http.Post(loginUrl, "application/json", bytes.NewBuffer(authReqBody))
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Cannot reach Auth Service, please confirm it is running.",
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		msg := "Invalid credentials"
		if val, ok := errResp["error"]; ok {
			msg = val
		}
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": msg,
		})
	}

	// Decode successful JWT response
	var authSuccess map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&authSuccess)
	tokenString := authSuccess["token"].(string)

	// Set HttpOnly cookie locally on WMS domain
	c.Cookie(&fiber.Cookie{
		Name:     "jwt_token",
		Value:    tokenString,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Path:     "/",
	})

	// Tell HTMX client to redirect to dashboard
	c.Set("HX-Redirect", "/")
	return c.SendStatus(fiber.StatusOK)
}

// GET /logout
func HandleLogout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "jwt_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Path:     "/",
	})
	return c.Redirect("/login")
}

// GET / (Main Dashboard)
func ServeDashboard(c *fiber.Ctx) error {
	// Fetch catalog list
	catalog, err := repository.FetchInventoryCatalog("")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	// Fetch physical locators and products for modal dropdowns
	var locators []models.Locator
	_ = database.DB.Preload("Warehouse").Find(&locators).Error

	var products []models.Product
	_ = database.DB.Preload("UoM").Find(&products).Error

	var uoms []models.UoM
	_ = database.DB.Find(&uoms).Error

	// Fetch active movements
	var movements []models.InventoryMovement
	_ = database.DB.Preload("Lines.Product.UoM").Preload("Lines.FromLocator").Preload("Lines.ToLocator").
		Order("updated_at DESC").Find(&movements).Error

	// Compute stock totals for header stats
	totalOnHand := 0
	totalReserved := 0
	totalAvailable := 0
	for _, item := range catalog {
		totalOnHand += item.QtyOnHand
		totalReserved += item.QtyReserved
		totalAvailable += item.QtyAvailable
	}

	return renderPage(c, "dashboard.html", fiber.Map{
		"Catalog":        catalog,
		"Locators":       locators,
		"Products":       products,
		"UoMs":           uoms,
		"Movements":      movements,
		"TotalOnHand":    totalOnHand,
		"TotalReserved":   totalReserved,
		"TotalAvailable": totalAvailable,
	})
}

// GET /wms/inventory (Dynamic HTMX table search)
func GetInventoryList(c *fiber.Ctx) error {
	search := c.Query("search")
	catalog, err := repository.FetchInventoryCatalog(search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return renderPartial(c, "partials/inventory_list.html", "inventory_list", fiber.Map{
		"Catalog": catalog,
	})
}

// POST /wms/movements/new - Handles Inbound/Outbound ticket creations
func CreateMovement(c *fiber.Ctx) error {
	moveType := c.FormValue("movement_type")
	productID := c.FormValue("product_id")
	quantity := c.FormValue("quantity")
	locatorID := c.FormValue("locator_id") // FromLocator for Outbound, ToLocator for Inbound
	remarks := c.FormValue("remarks")
	transUoMID := c.FormValue("uom_id")

	var qty int
	_, _ = fmt.Sscanf(quantity, "%d", &qty)

	if productID == "" || qty <= 0 {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Please select a product and enter a valid quantity.",
		})
	}

	// Fetch product to verify base UoM
	var product models.Product
	if err := database.DB.Preload("UoM").First(&product, "id = ?", productID).Error; err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": "Product not found.",
		})
	}

	originalQty := qty
	// If a custom transaction UoM is selected, apply conversion
	if transUoMID != "" && transUoMID != product.UoMID {
		var transUoM models.UoM
		_ = database.DB.First(&transUoM, "id = ?", transUoMID).Error

		var conv models.UoMConversion
		// Try product-specific conversion first
		err := database.DB.First(&conv, "product_id = ? AND from_uom_id = ? AND to_uom_id = ?", productID, transUoMID, product.UoMID).Error
		if err != nil {
			// Try global conversion next
			err = database.DB.First(&conv, "(product_id = '' OR product_id IS NULL) AND from_uom_id = ? AND to_uom_id = ?", transUoMID, product.UoMID).Error
		}

		if err != nil {
			return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
				"Success": false,
				"Message": fmt.Sprintf("No conversion formula registered to convert %s to product base unit %s.", transUoM.Code, product.UoM.Code),
			})
		}

		// Apply multiplier factor
		convertedQty := float64(qty) * conv.MultiplyFactor
		qty = int(convertedQty)

		remarks = fmt.Sprintf("%s (Converted from %d %s to %d %s using rule: 1 %s = %.2f %s)", 
			remarks, originalQty, transUoM.Code, qty, product.UoM.Code, transUoM.Code, conv.MultiplyFactor, product.UoM.Code)
	}

	userID := c.Locals("user_id").(string)

	// Document Number Generation
	docNo := fmt.Sprintf("MOV-%s-%d", moveType[:3], time.Now().UnixNano()%100000)

	isCrossDock := c.FormValue("is_cross_dock") == "on" || c.FormValue("is_cross_dock") == "true"

	movement := models.InventoryMovement{
		DocumentNo:   docNo,
		MovementType: moveType,
		IsCrossDock:  isCrossDock,
		Status:       "OPEN",
		CreatedBy:    userID,
		Remarks:      remarks,
	}

	line := models.InventoryMovementLine{
		ProductID:         productID,
		RequestedQuantity: qty,
	}

	if moveType == "INBOUND" {
		line.ToLocatorID = locatorID
	} else {
		line.FromLocatorID = locatorID
	}

	// Trigger repo creation
	err := repository.CreateInventoryMovement(&movement, []models.InventoryMovementLine{line})
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	// Refresh client to reload dashboard
	setReloadToast(c, fmt.Sprintf("Movement %s registered successfully.", docNo), true)
	return c.SendStatus(fiber.StatusCreated)
}

// POST /wms/movements/:id/claim
func ClaimMovement(c *fiber.Ctx) error {
	id := c.Params("id")
	operatorID := c.Locals("user_id").(string)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var mov models.InventoryMovement
		if err := tx.First(&mov, "id = ?", id).Error; err != nil {
			return err
		}
		mov.Status = "IN_PROGRESS"
		mov.AssignedOperatorID = operatorID
		mov.UpdatedAt = time.Now()
		return tx.Save(&mov).Error
	})

	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	setReloadToast(c, "Task claimed successfully.", true)
	return c.SendStatus(fiber.StatusOK)
}

// POST /wms/movements/:id/journal
func JournalMovement(c *fiber.Ctx) error {
	id := c.Params("id")
	err := repository.JournalizeInventoryMovement(id)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	setReloadToast(c, "Movement successfully journaled.", true)
	return c.SendStatus(fiber.StatusOK)
}

// POST /wms/movements/:id/complete
func CompleteMovement(c *fiber.Ctx) error {
	id := c.Params("id")
	err := repository.UpdateMovementStatus(id, "COMPLETED")
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	setReloadToast(c, "Task completed successfully.", true)
	return c.SendStatus(fiber.StatusOK)
}

// POST /wms/movements/:id/reject
func RejectMovement(c *fiber.Ctx) error {
	id := c.Params("id")
	reason := c.FormValue("rejection_reason")
	if reason == "" {
		reason = "Cancelled by warehouse manager"
	}

	err := repository.RejectInventoryMovement(id, reason)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}

	setReloadToast(c, "Movement successfully rejected.", true)
	return c.SendStatus(fiber.StatusOK)
}
