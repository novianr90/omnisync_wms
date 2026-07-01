package main

import (
	"log"
	"os"

	"auth_services/internal/database"
	"auth_services/internal/handlers"
	"auth_services/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func createAdmin(email, password string) {
	hashBytes, _ := bcrypt.GenerateFromPassword([]byte(password), 14)

	var adminRole models.Role
	if err := database.DB.Where("name = ?", "System Admin").First(&adminRole).Error; err != nil {
		log.Fatal("Could not find System Admin role. Run migrations first.")
	}

	user := models.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: string(hashBytes),
		FirstName:    "Admin",
		LastName:     "User",
		RoleID:       adminRole.ID,
		IsActive:     true,
	}
	if err := database.DB.Create(&user).Error; err != nil {
		log.Fatalf("Failed to create admin: %v", err)
	}
	log.Printf("Admin created: %s", email)
}

func seedUsers() {
	var count int64
	database.DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		log.Println("Seeding default users...")

		hashPassword := func(pwd string) string {
			bytes, _ := bcrypt.GenerateFromPassword([]byte(pwd), 14)
			return string(bytes)
		}

		adminEmail := os.Getenv("SEED_ADMIN_EMAIL")
		if adminEmail == "" {
			adminEmail = "admin@omnisync.com"
		}
		adminPwd := os.Getenv("SEED_ADMIN_PASSWORD")
		if adminPwd == "" {
			adminPwd = "admin123"
		}
		operatorEmail := os.Getenv("SEED_OPERATOR_EMAIL")
		if operatorEmail == "" {
			operatorEmail = "operator@omnisync.com"
		}
		operatorPwd := os.Getenv("SEED_OPERATOR_PASSWORD")
		if operatorPwd == "" {
			operatorPwd = "operator123"
		}

		var adminRole models.Role
		if err := database.DB.Where("name = ?", "System Admin").First(&adminRole).Error; err != nil {
			log.Println("Could not find System Admin role, skipping user seed")
			return
		}

		var posRole models.Role
		if err := database.DB.Where("name = ?", "POS").First(&posRole).Error; err != nil {
			log.Println("Could not find POS role, skipping user seed")
			return
		}

		admin := models.User{
			ID:           uuid.New().String(),
			Email:        adminEmail,
			PasswordHash: hashPassword(adminPwd),
			FirstName:    "Omni",
			LastName:     "Admin",
			RoleID:       adminRole.ID,
			IsActive:     true,
		}

		operator := models.User{
			ID:           uuid.New().String(),
			Email:        operatorEmail,
			PasswordHash: hashPassword(operatorPwd),
			FirstName:    "Alex",
			LastName:     "Mercer",
			RoleID:       posRole.ID,
			IsActive:     true,
		}

		_ = database.DB.Create(&admin).Error
		_ = database.DB.Create(&operator).Error

		log.Printf("Default users seeded successfully:")
		log.Printf("1. Admin: %s", adminEmail)
		log.Printf("2. Operator: %s", operatorEmail)
	}
}

func main() {
	// 0. Load .env file (ignored if not present — e.g. in production with real env vars)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	// 1. Initialize Database
	database.InitDB()

	// CLI subcommand: create-admin <email> <password>
	if len(os.Args) > 1 && os.Args[1] == "create-admin" {
		if len(os.Args) != 4 {
			log.Fatal("Usage: auth_services create-admin <email> <password>")
		}
		createAdmin(os.Args[2], os.Args[3])
		return
	}

	// 2. Seed Default Users (skipped in production)
	if os.Getenv("APP_ENV") != "production" {
		seedUsers()
	}

	// 3. Initialize Go Fiber Application
	app := fiber.New(fiber.Config{
		AppName: "Omnisync WMS - Auth Service",
	})

	// 3.1 Health Check (before any middleware — used by CI/Playwright readiness probe)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// 4. CORS Middleware
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:9901, http://127.0.0.1:9901"
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigin,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, OPTIONS, PUT, DELETE",
		AllowCredentials: true,
	}))

	// 5. Define Authentication Endpoints
	app.Post("/auth/register", handlers.AdminMiddleware, handlers.Register)
	app.Post("/auth/login", handlers.Login)
	app.Post("/auth/logout", handlers.Logout)
	app.Get("/auth/verify", handlers.VerifyToken)

	// User Management Endpoints
	app.Get("/users", handlers.AdminMiddleware, handlers.GetUsers)
	app.Put("/users/:id/status", handlers.AdminMiddleware, handlers.UpdateUserStatus)

	// Role Management Endpoints
	app.Get("/roles", handlers.AdminMiddleware, handlers.GetRoles)
	app.Post("/roles", handlers.AdminMiddleware, handlers.CreateRole)
	app.Put("/roles/:id", handlers.AdminMiddleware, handlers.UpdateRole)
	app.Delete("/roles/:id", handlers.AdminMiddleware, handlers.DeleteRole)

	// 6. Start the Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	log.Printf("Authentication Service is starting on port %s...", port)
	log.Fatal(app.Listen(":" + port))
}
