package middleware

import (
	"bytes"
	"html/template"
	"log"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
)

// Helper to render partial warning toasts in middleware
func renderWarningToast(c *fiber.Ctx, msg string) error {
	fp := filepath.Join("web", "templates", "partials", "notification.html")
	tmpl, err := template.ParseFiles(fp)
	if err != nil {
		log.Printf("Warning toast parsing error: %v", err)
		return c.Status(fiber.StatusForbidden).SendString("Access Denied: Admin role required.")
	}

	var buf bytes.Buffer
	data := fiber.Map{
		"Success": false,
		"Message": msg,
	}
	if err := tmpl.ExecuteTemplate(&buf, "notification", data); err != nil {
		log.Printf("Warning toast execution error: %v", err)
		return c.Status(fiber.StatusForbidden).SendString("Access Denied: Admin role required.")
	}

	c.Set("Content-Type", "text/html")
	return c.Send(buf.Bytes())
}

func hasPermission(c *fiber.Ctx, required string) bool {
	permsVal := c.Locals("user_permissions")
	if permsVal == nil {
		role, ok := c.Locals("user_role").(string)
		if !ok {
			return false
		}
		if role == "System Admin" {
			return true
		}
		if role == "Admin WMS" && (required == "modify_masters" || required == "manage_system") {
			return true
		}
		return false
	}

	switch slice := permsVal.(type) {
	case []string:
		for _, p := range slice {
			if p == required {
				return true
			}
		}
	case []interface{}:
		for _, item := range slice {
			if s, ok := item.(string); ok && s == required {
				return true
			}
		}
	}
	return false
}

// RequireAdmin verifies that the logged-in user has permission to modify master data
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !hasPermission(c, "modify_masters") {
			if c.Get("HX-Request") == "true" {
				return renderWarningToast(c, "Access Denied: You do not have permission to modify master data.")
			}
			return c.Status(fiber.StatusForbidden).SendString("<h1>Access Denied</h1><p>You do not have permission to modify master data.</p>")
		}
		return c.Next()
	}
}

// RequireSystemAdmin verifies that the logged-in user has permission to view the ledger
func RequireSystemAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !hasPermission(c, "view_ledger") {
			if c.Get("HX-Request") == "true" {
				return renderWarningToast(c, "Access Denied: You do not have permission to view the ledger.")
			}
			return c.Status(fiber.StatusForbidden).SendString("<h1>Access Denied</h1><p>You do not have permission to view the ledger.</p>")
		}
		return c.Next()
	}
}

// RequireSystemManage verifies that the logged-in user has permission to manage system roles/users
func RequireSystemManage() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !hasPermission(c, "manage_system") {
			if c.Get("HX-Request") == "true" {
				return renderWarningToast(c, "Access Denied: You do not have permission to manage the system.")
			}
			return c.Status(fiber.StatusForbidden).SendString("<h1>Access Denied</h1><p>You do not have permission to manage the system.</p>")
		}
		return c.Next()
	}
}
