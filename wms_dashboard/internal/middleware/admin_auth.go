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

// RequireAdmin verifies that the logged-in user is an administrator
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals("user_role")
		if role != "admin" {
			// For HTMX AJAX requests expecting HTML swaps, return a beautiful notification toast
			if isHtmlRequest(c) {
				return renderWarningToast(c, "Access Denied: Admin role required to modify master data.")
			}
			
			// Standard API fallback
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: Admin role required",
			})
		}
		return c.Next()
	}
}
