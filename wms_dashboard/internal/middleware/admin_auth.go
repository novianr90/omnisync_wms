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

// Helper to render styled access denied page in middleware
func renderAccessDeniedPage(c *fiber.Ctx, msg string) error {
	lp := filepath.Join("web", "templates", "layouts", "base.html")
	fp := filepath.Join("web", "templates", "pages", "forbidden.html")

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

	tmpl, err := template.New("base").Funcs(funcMap).ParseFiles(lp, fp)
	if err != nil {
		log.Printf("Forbidden page template parsing error: %v", err)
		return c.Status(fiber.StatusForbidden).SendString("Access Denied: " + msg)
	}

	var buf bytes.Buffer
	data := fiber.Map{
		"Username":        c.Locals("user_name"),
		"UserRole":         c.Locals("user_role"),
		"UserEmail":        c.Locals("user_email"),
		"UserPermissions": c.Locals("user_permissions"),
		"ErrorMessage":    msg,
	}

	if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
		log.Printf("Forbidden page template execution error: %v", err)
		return c.Status(fiber.StatusForbidden).SendString("Access Denied: " + msg)
	}

	c.Set("Content-Type", "text/html")
	return c.Status(fiber.StatusForbidden).Send(buf.Bytes())
}

func hasPermission(c *fiber.Ctx, required string) bool {
	permsVal := c.Locals("user_permissions")
	
	isEmpty := true
	if permsVal != nil {
		switch slice := permsVal.(type) {
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
			return renderAccessDeniedPage(c, "You do not have permission to modify master data.")
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
			return renderAccessDeniedPage(c, "You do not have permission to view the ledger.")
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
			return renderAccessDeniedPage(c, "You do not have permission to manage the system.")
		}
		return c.Next()
	}
}

// RequireManageMovements verifies that the logged-in user has permission to manage movements
func RequireManageMovements() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !hasPermission(c, "manage_movements") {
			if c.Get("HX-Request") == "true" {
				return renderWarningToast(c, "Access Denied: You do not have permission to manage movements.")
			}
			return renderAccessDeniedPage(c, "You do not have permission to manage movements.")
		}
		return c.Next()
	}
}
