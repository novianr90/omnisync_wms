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

		admin := models.User{
			ID:           uuid.New().String(),
			Email:        adminEmail,
			PasswordHash: hashPassword(adminPwd),
			FirstName:    "Omni",
			LastName:     "Admin",
			Role:         "admin",
			IsActive:     true,
		}

		operator := models.User{
			ID:           uuid.New().String(),
			Email:        operatorEmail,
			PasswordHash: hashPassword(operatorPwd),
			FirstName:    "Alex",
			LastName:     "Mercer",
			Role:         "operator",
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

	// 2. Seed Default Users
	seedUsers()

	// 3. Initialize Go Fiber Application
	app := fiber.New(fiber.Config{
		AppName: "Omnisync WMS - Auth Service",
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
	app.Post("/auth/register", handlers.Register)
	app.Post("/auth/login", handlers.Login)
	app.Post("/auth/logout", handlers.Logout)
	app.Get("/auth/verify", handlers.VerifyToken)

	// 6. Start the Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	log.Printf("Authentication Service is starting on port %s...", port)
	log.Fatal(app.Listen(":" + port))
}
