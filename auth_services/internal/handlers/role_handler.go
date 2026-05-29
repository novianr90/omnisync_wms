package handlers

import (
	"auth_services/internal/database"
	"auth_services/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

func GetRoles(c *fiber.Ctx) error {
	var roles []models.Role
	if err := database.DB.Preload("Permissions").Find(&roles).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch roles",
		})
	}

	return c.JSON(fiber.Map{
		"roles": roles,
	})
}

func CreateRole(c *fiber.Ctx) error {
	var req RoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Role name is required",
		})
	}

	var existing models.Role
	if err := database.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Role name already exists",
		})
	}

	newRole := models.Role{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newRole).Error; err != nil {
			return err
		}
		for _, perm := range req.Permissions {
			rolePerm := models.RolePermission{
				RoleID:     newRole.ID,
				Permission: perm,
			}
			if err := tx.Create(&rolePerm).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create role",
		})
	}

	// Fetch with preloaded permissions
	if err := database.DB.Preload("Permissions").Where("id = ?", newRole.ID).First(&newRole).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch created role",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(newRole)
}

func UpdateRole(c *fiber.Ctx) error {
	id := c.Params("id")
	var req RoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse request body",
		})
	}

	var role models.Role
	if err := database.DB.Where("id = ?", id).First(&role).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Role not found",
		})
	}

	role.Name = req.Name
	role.Description = req.Description

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&role).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}
		for _, perm := range req.Permissions {
			rolePerm := models.RolePermission{
				RoleID:     id,
				Permission: perm,
			}
			if err := tx.Create(&rolePerm).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update role",
		})
	}

	// Preload to return updated role
	if err := database.DB.Preload("Permissions").Where("id = ?", id).First(&role).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch updated role",
		})
	}

	return c.JSON(role)
}

func DeleteRole(c *fiber.Ctx) error {
	id := c.Params("id")
	
	// Check if role is used by any user
	var count int64
	database.DB.Model(&models.User{}).Where("role_id = ?", id).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Cannot delete role as it is assigned to users",
		})
	}

	if err := database.DB.Where("id = ?", id).Delete(&models.Role{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete role",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Role deleted successfully",
	})
}
