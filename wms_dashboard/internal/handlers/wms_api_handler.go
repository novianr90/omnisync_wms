package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
	"wms_dashboard/internal/repository"
)

// APILogin - Mobile API for authenticating and obtaining a JWT token
func APILogin(c *fiber.Ctx) error {
	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	authReqBody, _ := json.Marshal(map[string]string{
		"email":    req.Email,
		"password": req.Password,
	})

	authAPIUrl := os.Getenv("AUTH_API_URL")
	if authAPIUrl == "" {
		authAPIUrl = "http://localhost:8000"
	}
	loginUrl := fmt.Sprintf("%s/auth/login", authAPIUrl)

	resp, err := http.Post(loginUrl, "application/json", bytes.NewBuffer(authReqBody))
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Cannot reach Auth Service",
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": msg,
		})
	}

	var authSuccess map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&authSuccess)

	return c.Status(fiber.StatusOK).JSON(authSuccess)
}

// APIGetProductBySKU - Look up product by scanning barcode (SKU)
func APIGetProductBySKU(c *fiber.Ctx) error {
	sku := c.Params("sku")

	var product models.Product
	if err := database.DB.Preload("UoM").First(&product, "sku = ?", sku).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Product not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(product)
}

// APIGetLocatorByCode - Look up shelf locator by scanning barcode (Code)
func APIGetLocatorByCode(c *fiber.Ctx) error {
	code := c.Params("code")

	var locator models.Locator
	if err := database.DB.Preload("Warehouse").First(&locator, "code = ?", code).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Locator not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(locator)
}

// APICreateMovementRequest represents the expected JSON payload for creating movements
type APICreateMovementRequest struct {
	MovementType string `json:"movement_type"` // INBOUND or OUTBOUND
	ProductID    string `json:"product_id"`
	LocatorID    string `json:"locator_id"`    // FromLocator for Outbound, ToLocator for Inbound
	Quantity     int    `json:"quantity"`
	UoMID        string `json:"uom_id"`        // Optional transaction unit of measure
	Remarks      string `json:"remarks"`
	IsCrossDock  bool   `json:"is_cross_dock"` // Optional cross-dock flag
}

// APICreateMovement - Create movement via JSON API
func APICreateMovement(c *fiber.Ctx) error {
	var req APICreateMovementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	if req.ProductID == "" || req.Quantity <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Product ID and a valid quantity (>0) are required",
		})
	}

	// Fetch product to verify base UoM
	var product models.Product
	if err := database.DB.Preload("UoM").First(&product, "id = ?", req.ProductID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Product not found",
		})
	}

	qty := req.Quantity
	originalQty := qty
	remarks := req.Remarks

	// Apply UoM Conversion if different from base UoM
	if req.UoMID != "" && req.UoMID != product.UoMID {
		var transUoM models.UoM
		if err := database.DB.First(&transUoM, "id = ?", req.UoMID).Error; err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Transaction UoM not found",
			})
		}

		var conv models.UoMConversion
		err := database.DB.First(&conv, "product_id = ? AND from_uom_id = ? AND to_uom_id = ?", req.ProductID, req.UoMID, product.UoMID).Error
		if err != nil {
			err = database.DB.First(&conv, "(product_id = '' OR product_id IS NULL) AND from_uom_id = ? AND to_uom_id = ?", req.UoMID, product.UoMID).Error
		}

		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("No conversion formula registered to convert %s to %s", transUoM.Code, product.UoM.Code),
			})
		}

		convertedQty := float64(qty) * conv.MultiplyFactor
		qty = int(convertedQty)

		remarks = fmt.Sprintf("%s (Converted from %d %s to %d %s using rule: 1 %s = %.2f %s)", 
			remarks, originalQty, transUoM.Code, qty, product.UoM.Code, transUoM.Code, conv.MultiplyFactor, product.UoM.Code)
	}

	// Extract user ID from token middleware
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	docNo := fmt.Sprintf("MOV-%s-%d", req.MovementType[:3], time.Now().UnixNano()%100000)

	movement := models.InventoryMovement{
		DocumentNo:   docNo,
		MovementType: req.MovementType,
		IsCrossDock:  req.IsCrossDock,
		Status:       "OPEN",
		CreatedBy:    userID.(string),
		Remarks:      remarks,
	}

	line := models.InventoryMovementLine{
		ProductID:         req.ProductID,
		RequestedQuantity: qty,
	}

	if req.MovementType == "INBOUND" {
		line.ToLocatorID = req.LocatorID
	} else if req.MovementType == "OUTBOUND" {
		line.FromLocatorID = req.LocatorID
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid movement_type. Must be INBOUND or OUTBOUND",
		})
	}

	if err := repository.CreateInventoryMovement(&movement, []models.InventoryMovementLine{line}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Movement created successfully",
		"document_no": docNo,
	})
}
