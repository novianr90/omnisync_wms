package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
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
	data["UserPermissions"] = c.Locals("user_permissions")

	lp := filepath.Join("web", "templates", "layouts", "base.html")
	fp := filepath.Join("web", "templates", "pages", pageTemplate)

	var tmpl *template.Template
	var err error
	var execTmpl string

	funcMap := template.FuncMap{
		"hasPermission": func(perms interface{}, role interface{}, required string) bool {
			isEmpty := true
			if perms != nil {
				switch slice := perms.(type) {
				case []string:
					if len(slice) > 0 {
						isEmpty = false
						for _, p := range slice {
							if p == required {
								return true
							}
						}
					}
				case []interface{}:
					if len(slice) > 0 {
						isEmpty = false
						for _, item := range slice {
							if s, ok := item.(string); ok && s == required {
								return true
							}
						}
					}
				}
			}
			if isEmpty {
				roleStr := ""
				if role != nil {
					if s, ok := role.(string); ok {
						roleStr = s
					}
				}
				if roleStr == "System Admin" {
					return true
				}
				if roleStr == "Admin WMS" && (required == "modify_masters" || required == "manage_system") {
					return true
				}
			}
			return false
		},
	}

	if c.Get("HX-Request") == "true" {
		tmpl = template.New(pageTemplate).Funcs(funcMap)
		tmpl, err = tmpl.ParseFiles(fp)
		execTmpl = "content"
	} else {
		tmpl = template.New("base").Funcs(funcMap)
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
	data["UserPermissions"] = c.Locals("user_permissions")

	partialFuncMap := template.FuncMap{
		"hasPermission": func(perms interface{}, role interface{}, required string) bool {
			isEmpty := true
			if perms != nil {
				switch slice := perms.(type) {
				case []string:
					if len(slice) > 0 {
						isEmpty = false
						for _, p := range slice {
							if p == required {
								return true
							}
						}
					}
				case []interface{}:
					if len(slice) > 0 {
						isEmpty = false
						for _, item := range slice {
							if s, ok := item.(string); ok && s == required {
								return true
							}
						}
					}
				}
			}
			if isEmpty {
				roleStr := ""
				if role != nil {
					if s, ok := role.(string); ok {
						roleStr = s
					}
				}
				if roleStr == "System Admin" {
					return true
				}
				if roleStr == "Admin WMS" && (required == "modify_masters" || required == "manage_system") {
					return true
				}
			}
			return false
		},
	}

	fp := filepath.Join("web", "templates", partialPath)
	tmpl, err := template.New(templateName).Funcs(partialFuncMap).ParseFiles(fp)
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
		"TotalReserved":  totalReserved,
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

// POST /wms/movements - Handles multi-line Inbound/Outbound ticket creation (JSON input)
func CreateMovement(c *fiber.Ctx) error {
	var req struct {
		MovementType string `json:"movement_type"`
		IsCrossDock  bool   `json:"is_cross_dock"`
		Remarks      string `json:"remarks"`
		Lines        []struct {
			ProductID string `json:"product_id"`
			Quantity  int    `json:"quantity"`
			UoMID     string `json:"uom_id"`
			LocatorID string `json:"locator_id"`
		} `json:"lines"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(req.Lines) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one item line must be provided.",
		})
	}

	userID := c.Locals("user_id").(string)

	// Base document setup
	docNo := fmt.Sprintf("MOV-%s-%d", req.MovementType[:3], time.Now().UnixNano()%100000)
	movement := models.InventoryMovement{
		DocumentNo:   docNo,
		MovementType: req.MovementType,
		IsCrossDock:  req.IsCrossDock,
		Status:       "OPEN",
		CreatedBy:    userID,
		Remarks:      req.Remarks,
	}

	var movementLines []models.InventoryMovementLine

	for idx, l := range req.Lines {
		if l.ProductID == "" || l.Quantity <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Line %d: Please select a product and enter a valid quantity.", idx+1),
			})
		}

		// Fetch product to verify base UoM
		var product models.Product
		if err := database.DB.Preload("UoM").First(&product, "id = ?", l.ProductID).Error; err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Line %d: Product not found.", idx+1),
			})
		}

		qty := l.Quantity
		remarksLine := ""
		// Apply UoM conversion if necessary
		if l.UoMID != "" && l.UoMID != product.UoMID {
			var transUoM models.UoM
			_ = database.DB.First(&transUoM, "id = ?", l.UoMID).Error

			var conv models.UoMConversion
			err := database.DB.First(&conv, "product_id = ? AND from_uom_id = ? AND to_uom_id = ?", l.ProductID, l.UoMID, product.UoMID).Error
			if err != nil {
				err = database.DB.First(&conv, "(product_id = '' OR product_id IS NULL) AND from_uom_id = ? AND to_uom_id = ?", l.UoMID, product.UoMID).Error
			}

			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Line %d: No conversion formula registered to convert %s to product base unit %s.", idx+1, transUoM.Code, product.UoM.Code),
				})
			}

			convertedQty := float64(qty) * conv.MultiplyFactor
			qty = int(convertedQty)

			remarksLine = fmt.Sprintf("Line %d: %d %s converted to %d %s", idx+1, l.Quantity, transUoM.Code, qty, product.UoM.Code)
		}

		line := models.InventoryMovementLine{
			ProductID:         l.ProductID,
			RequestedQuantity: qty,
		}

		if req.MovementType == "INBOUND" {
			if !req.IsCrossDock && l.LocatorID == "" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Line %d: Target shelf locator is required for Inbound movements.", idx+1),
				})
			}
			line.ToLocatorID = l.LocatorID
		} else {
			// For OUTBOUND, locator is optional for auto FIFO, but user can specify source locator zone
			line.FromLocatorID = l.LocatorID
		}

		movementLines = append(movementLines, line)

		if remarksLine != "" {
			if movement.Remarks != "" {
				movement.Remarks += "\n" + remarksLine
			} else {
				movement.Remarks = remarksLine
			}
		}
	}

	// Trigger repository creation (this does FIFO allocation and hold checks inside transaction)
	err := repository.CreateInventoryMovement(&movement, movementLines)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Set success cookies so redirect page picks up toast
	toastType := "success"
	message := fmt.Sprintf("Movement %s registered successfully.", movement.DocumentNo)
	c.Cookie(&fiber.Cookie{
		Name:     "toast_msg",
		Value:    message,
		HTTPOnly: false,
		Path:     "/",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "toast_type",
		Value:    toastType,
		HTTPOnly: false,
		Path:     "/",
	})

	return c.JSON(fiber.Map{
		"success":  true,
		"redirect": "/wms/movements",
	})
}

// GET /wms/movements
func ServeMovementsPage(c *fiber.Ctx) error {
	search := c.Query("search")
	mType := c.Query("type")
	status := c.Query("status")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	query := database.DB.Model(&models.InventoryMovement{}).
		Preload("Lines.Product.UoM").
		Preload("Lines.FromLocator").
		Preload("Lines.ToLocator")

	if search != "" {
		// Search by document_no or by product SKU/name in lines
		query = query.Where("document_no LIKE ? OR id IN (SELECT movement_id FROM inventory_movement_lines JOIN products ON inventory_movement_lines.product_id = products.id WHERE products.sku LIKE ? OR products.name LIKE ?)", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if mType != "" {
		query = query.Where("movement_type = ?", mType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}

	var movements []models.InventoryMovement
	if err := query.Order("updated_at DESC").Find(&movements).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	// For HTMX partial table replacement
	if c.Get("HX-Request") == "true" && c.Get("HX-Target") == "movements-tbody" {
		return renderPartial(c, "partials/movements_table.html", "movements_table", fiber.Map{
			"Movements": movements,
		})
	}

	return renderPage(c, "movements.html", fiber.Map{
		"Movements": movements,
	})
}

// GET /wms/movements/new
func ServeNewMovementPage(c *fiber.Ctx) error {
	var products []models.Product
	_ = database.DB.Preload("UoM").Find(&products).Error

	var locators []models.Locator
	_ = database.DB.Preload("Warehouse").Find(&locators).Error

	var uoms []models.UoM
	_ = database.DB.Find(&uoms).Error

	return renderPage(c, "movement_new.html", fiber.Map{
		"Products": products,
		"Locators": locators,
		"UoMs":     uoms,
	})
}

// GET /wms/movements/:id
func ServeMovementDetailPage(c *fiber.Ctx) error {
	id := c.Params("id")

	var movement models.InventoryMovement
	err := database.DB.Preload("Lines.Product.UoM").
		Preload("Lines.FromLocator").
		Preload("Lines.ToLocator").
		First(&movement, "id = ?", id).Error

	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("<h1>Document Not Found</h1>")
	}

	return renderPage(c, "movement_detail.html", fiber.Map{
		"Movement": movement,
	})
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
		if mov.Status != "OPEN" {
			return errors.New("movement is already claimed or not available")
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

// POST /wms/movements/:id/crossdock/inbound
func ConfirmCrossDockInbound(c *fiber.Ctx) error {
	id := c.Params("id")
	err := repository.ProcessCrossDockInbound(id)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}
	setReloadToast(c, "Cross-dock inbound receipt confirmed.", true)
	return c.SendStatus(fiber.StatusOK)
}

// POST /wms/movements/:id/crossdock/shipping
func ConfirmCrossDockShipping(c *fiber.Ctx) error {
	id := c.Params("id")
	err := repository.ProcessCrossDockShipping(id)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}
	setReloadToast(c, "Cross-dock loading initiated.", true)
	return c.SendStatus(fiber.StatusOK)
}

// POST /wms/movements/:id/crossdock/outbound
func ConfirmCrossDockOutbound(c *fiber.Ctx) error {
	id := c.Params("id")
	err := repository.ProcessCrossDockOutbound(id)
	if err != nil {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{
			"Success": false,
			"Message": err.Error(),
		})
	}
	setReloadToast(c, "Cross-dock outbound dispatch confirmed.", true)
	return c.SendStatus(fiber.StatusOK)
}
