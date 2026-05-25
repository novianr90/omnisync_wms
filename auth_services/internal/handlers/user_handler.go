package handlers

import (
	"auth_services/internal/database"
	"auth_services/internal/models"
	"github.com/gofiber/fiber/v2"
)

func GetUsers(c *fiber.Ctx) error {
	var users []models.User
	if err := database.DB.Preload("Role").Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch users",
		})
	}

	// We shouldn't return PasswordHash
	// The json:"-" tag in the model will hide it, but just to be sure we can map it if we want.
	return c.JSON(fiber.Map{
		"users": users,
	})
}

func UpdateUserStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	
	type StatusRequest struct {
		IsActive bool `json:"is_active"`
	}

	var req StatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse request body",
		})
	}

	var user models.User
	if err := database.DB.Where("id = ?", id).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	user.IsActive = req.IsActive
	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user status",
		})
	}

	return c.JSON(fiber.Map{
		"message": "User status updated",
		"is_active": user.IsActive,
	})
}
