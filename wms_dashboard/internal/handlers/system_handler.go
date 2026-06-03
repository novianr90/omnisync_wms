package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
)

type SystemUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	IsActive  bool   `json:"is_active"`
	Role      struct {
		Name string `json:"name"`
	} `json:"role"`
}

type SystemRolePermission struct {
	RoleID     string `json:"role_id"`
	Permission string `json:"permission"`
}

type SystemRole struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Permissions []SystemRolePermission `json:"permissions"`
}

func getAuthAPIUrl() string {
	url := os.Getenv("AUTH_API_URL")
	if url == "" {
		return "http://localhost:8000"
	}
	return url
}

func doAuthRequest(c *fiber.Ctx, method, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, getAuthAPIUrl()+path, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := c.Cookies("jwt_token"); token != "" {
		req.AddCookie(&http.Cookie{Name: "jwt_token", Value: token})
	}
	return http.DefaultClient.Do(req)
}

func ServeUsersMaster(c *fiber.Ctx) error {
	resp, err := doAuthRequest(c, "GET", "/users", nil)
	if err != nil || resp.StatusCode != 200 {
		return renderPage(c, "users_master.html", fiber.Map{"Users": []SystemUser{}, "Roles": []SystemRole{}})
	}
	defer resp.Body.Close()

	var result struct {
		Users []SystemUser `json:"users"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	// Fetch roles for the dropdown
	roleResp, _ := doAuthRequest(c, "GET", "/roles", nil)
	var roleResult struct {
		Roles []SystemRole `json:"roles"`
	}
	if roleResp != nil && roleResp.StatusCode == 200 {
		json.NewDecoder(roleResp.Body).Decode(&roleResult)
		roleResp.Body.Close()
	}

	return renderPage(c, "users_master.html", fiber.Map{
		"Users": result.Users,
		"Roles": roleResult.Roles,
	})
}

var AllPermissions = []string{"view_ledger", "modify_masters", "manage_system", "manage_movements"}

func ServeRolesMaster(c *fiber.Ctx) error {
	resp, err := doAuthRequest(c, "GET", "/roles", nil)
	if err != nil || resp.StatusCode != 200 {
		return renderPage(c, "roles_master.html", fiber.Map{"Roles": []SystemRole{}, "AllPermissions": AllPermissions})
	}
	defer resp.Body.Close()

	var result struct {
		Roles []SystemRole `json:"roles"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return renderPage(c, "roles_master.html", fiber.Map{
		"Roles":          result.Roles,
		"AllPermissions": AllPermissions,
	})
}

func GetUsersRows(c *fiber.Ctx) error {
	resp, err := doAuthRequest(c, "GET", "/users", nil)
	if err != nil || resp.StatusCode != 200 {
		return c.SendString("<tr><td colspan='5'>Failed to fetch users</td></tr>")
	}
	defer resp.Body.Close()

	var result struct {
		Users []SystemUser `json:"users"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return renderPartial(c, "partials/user_rows.html", "user_rows", fiber.Map{
		"Users": result.Users,
	})
}

func GetRolesRows(c *fiber.Ctx) error {
	resp, err := doAuthRequest(c, "GET", "/roles", nil)
	if err != nil || resp.StatusCode != 200 {
		return c.SendString("<tr><td colspan='3'>Failed to fetch roles</td></tr>")
	}
	defer resp.Body.Close()

	var result struct {
		Roles []SystemRole `json:"roles"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return renderPartial(c, "partials/role_rows.html", "role_rows", fiber.Map{
		"Roles": result.Roles,
	})
}

func CreateUser(c *fiber.Ctx) error {
	reqBody, _ := json.Marshal(map[string]string{
		"email":      c.FormValue("email"),
		"password":   c.FormValue("password"),
		"first_name": c.FormValue("first_name"),
		"last_name":  c.FormValue("last_name"),
		"role":       c.FormValue("role"), // ID or Name based on how auth service expects it (we changed it to expect Name earlier, so we should send the Name, wait, we send RoleName)
	})

	resp, err := doAuthRequest(c, "POST", "/auth/register", reqBody)
	if err != nil || resp.StatusCode != 201 {
		msg := "Failed to create user"
		if resp != nil {
			var errResp map[string]string
			json.NewDecoder(resp.Body).Decode(&errResp)
			if m, ok := errResp["error"]; ok {
				msg = m
			}
		}
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{"Success": false, "Message": msg})
	}

	c.Set("HX-Trigger", "refreshUsersList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{"Success": true, "Message": "User created successfully"})
}

func UpdateUserStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	isActive := c.FormValue("is_active") == "true"
	
	reqBody, _ := json.Marshal(map[string]bool{
		"is_active": isActive,
	})

	resp, err := doAuthRequest(c, "PUT", fmt.Sprintf("/users/%s/status", id), reqBody)
	if err != nil || resp.StatusCode != 200 {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{"Success": false, "Message": "Failed to update status"})
	}

	c.Set("HX-Trigger", "refreshUsersList")
	return c.SendStatus(200)
}

func CreateRole(c *fiber.Ctx) error {
	var permissions []string
	c.Context().PostArgs().VisitAll(func(key, val []byte) {
		if string(key) == "permissions" || string(key) == "permissions[]" {
			permissions = append(permissions, string(val))
		}
	})

	reqBody, _ := json.Marshal(map[string]interface{}{
		"name":        c.FormValue("name"),
		"description": c.FormValue("description"),
		"permissions": permissions,
	})

	resp, err := doAuthRequest(c, "POST", "/roles", reqBody)
	if err != nil || resp.StatusCode != 201 {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{"Success": false, "Message": "Failed to create role"})
	}

	c.Set("HX-Trigger", "refreshRolesList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{"Success": true, "Message": "Role created successfully"})
}

func UpdateRole(c *fiber.Ctx) error {
	id := c.Params("id")
	var permissions []string
	c.Context().PostArgs().VisitAll(func(key, val []byte) {
		if string(key) == "permissions" || string(key) == "permissions[]" {
			permissions = append(permissions, string(val))
		}
	})

	reqBody, _ := json.Marshal(map[string]interface{}{
		"name":        c.FormValue("name"),
		"description": c.FormValue("description"),
		"permissions": permissions,
	})

	resp, err := doAuthRequest(c, "PUT", fmt.Sprintf("/roles/%s", id), reqBody)
	if err != nil || resp.StatusCode != 200 {
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{"Success": false, "Message": "Failed to update role"})
	}

	c.Set("HX-Trigger", "refreshRolesList")
	return renderPartial(c, "partials/notification.html", "notification", fiber.Map{"Success": true, "Message": "Role updated successfully"})
}

func DeleteRole(c *fiber.Ctx) error {
	id := c.Params("id")
	resp, err := doAuthRequest(c, "DELETE", fmt.Sprintf("/roles/%s", id), nil)
	if err != nil || resp.StatusCode != 200 {
		msg := "Failed to delete role"
		if resp != nil && resp.StatusCode == 409 {
			msg = "Cannot delete role assigned to users"
		}
		return renderPartial(c, "partials/notification.html", "notification", fiber.Map{"Success": false, "Message": msg})
	}

	c.Set("HX-Trigger", "refreshRolesList")
	return c.SendStatus(200)
}
