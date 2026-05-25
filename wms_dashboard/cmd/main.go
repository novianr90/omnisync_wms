package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/handlers"
	"wms_dashboard/internal/middleware"
	"wms_dashboard/internal/models"
)

func seedWMSData() {
	var productCount int64
	database.DB.Model(&models.Product{}).Count(&productCount)
	if productCount == 0 {
		log.Println("Seeding physical WMS Master Data...")

		// 0. Seed UoMs
		uomKg := models.UoM{ID: uuid.New().String(), Code: "kg", Name: "Kilogram", Description: "Standard metric unit for weight"}
		uomPack := models.UoM{ID: uuid.New().String(), Code: "pack", Name: "Pack", Description: "Pack unit of items"}
		uomBox := models.UoM{ID: uuid.New().String(), Code: "box", Name: "Box", Description: "Box containing multiple individual items"}
		uomPcs := models.UoM{ID: uuid.New().String(), Code: "pcs", Name: "Pieces", Description: "Individual singular units"}
		_ = database.DB.Create(&uomKg).Error
		_ = database.DB.Create(&uomPack).Error
		_ = database.DB.Create(&uomBox).Error
		_ = database.DB.Create(&uomPcs).Error

		// 1. Seed Warehouses
		wh := models.Warehouse{
			ID:       uuid.New().String(),
			Code:     "WH-MAIN",
			Name:     "Central Logistics Hub",
			Address:  "Golden Gate Sector 4, Silicon Valley",
			IsActive: true,
		}
		_ = database.DB.Create(&wh).Error

		// 2. Seed Locators (Shelves)
		var locs []models.Locator
		zones := []string{"Zone-A", "Zone-B"}
		aisles := []string{"Aisle-1", "Aisle-2"}
		shelves := []string{"Shelf-1", "Shelf-2"}
		levels := []string{"Level-1", "Level-2"}

		for _, z := range zones {
			for _, a := range aisles {
				for _, s := range shelves {
					for _, l := range levels {
						code := fmt.Sprintf("%s-%s-%s-%s-%s", wh.Code, z, a, s, l)
						loc := models.Locator{
							ID:          uuid.New().String(),
							WarehouseID: wh.ID,
							Zone:        z,
							Aisle:       a,
							Shelf:       s,
							Level:       l,
							Code:        code,
							IsActive:    true,
						}
						_ = database.DB.Create(&loc).Error
						locs = append(locs, loc)
					}
				}
			}
		}

		// 3. Seed Products
		prod1 := models.Product{
			ID:          uuid.New().String(),
			SKU:         "PROD-KYBD-01",
			Name:        "Mechanical Keychron K2 Keyboard",
			Description: "Wireless 84-Key mechanical keyboard with Gateron switches",
			Category:    "Electronics",
			Price:       89.99,
			UoMID:       uomPcs.ID,
		}
		prod2 := models.Product{
			ID:          uuid.New().String(),
			SKU:         "PROD-MOUS-02",
			Name:        "Logitech MX Master 3S Mouse",
			Description: "Ergonomic wireless office mouse with silent clicks",
			Category:    "Electronics",
			Price:       99.99,
			UoMID:       uomPcs.ID,
		}
		prod3 := models.Product{
			ID:          uuid.New().String(),
			SKU:         "PROD-MON-03",
			Name:        "Dell UltraSharp 27\" 4K Monitor",
			Description: "U2723QE USB-C Hub monitor with IPS Black technology",
			Category:    "Electronics",
			Price:       499.00,
			UoMID:       uomPcs.ID,
		}
		prod4 := models.Product{
			ID:          uuid.New().String(),
			SKU:         "PROD-SUGR-04",
			Name:        "Refined White Sugar",
			Description: "Fine granular white table sugar",
			Category:    "Consumables",
			Price:       1.99,
			UoMID:       uomKg.ID,
		}

		_ = database.DB.Create(&prod1).Error
		_ = database.DB.Create(&prod2).Error
		_ = database.DB.Create(&prod3).Error
		_ = database.DB.Create(&prod4).Error

		// 3.5 Seed Sugar Conversion Rule (1 pack of sugar = 1.0 kg of sugar)
		sugarConv := models.UoMConversion{
			ID:             uuid.New().String(),
			ProductID:      prod4.ID,
			FromUoMID:      uomPack.ID,
			ToUoMID:        uomKg.ID,
			MultiplyFactor: 1.0,
		}
		_ = database.DB.Create(&sugarConv).Error

		// 4. Seed Storage Lots (with FIFO demo items!)
		// Batch 1 (Oldest - Received 5 days ago)
		storage1 := models.Storage{
			ID:          uuid.New().String(),
			ProductID:   prod1.ID,
			LocatorID:   locs[0].ID, // WH-MAIN-Zone-A-Aisle-1-Shelf-1-Level-1
			BatchNumber: "BAT-INB-20260520-1",
			ReceivedAt:  time.Now().Add(-5 * 24 * time.Hour), // 5 days ago
			QtyOnHand:   40,
			QtyReserved: 0,
			UpdatedAt:   time.Now(),
		}

		// Batch 2 (Newer - Received today)
		storage2 := models.Storage{
			ID:          uuid.New().String(),
			ProductID:   prod1.ID,
			LocatorID:   locs[0].ID,
			BatchNumber: "BAT-INB-20260525-2",
			ReceivedAt:  time.Now(),
			QtyOnHand:   60,
			QtyReserved: 0,
			UpdatedAt:   time.Now(),
		}

		// Mouse Storage (Received 2 days ago)
		storage3 := models.Storage{
			ID:          uuid.New().String(),
			ProductID:   prod2.ID,
			LocatorID:   locs[1].ID, // WH-MAIN-Zone-A-Aisle-1-Shelf-1-Level-2
			BatchNumber: "BAT-INB-20260523-1",
			ReceivedAt:  time.Now().Add(-2 * 24 * time.Hour),
			QtyOnHand:   80,
			QtyReserved: 0,
			UpdatedAt:   time.Now(),
		}

		// Sugar Storage (Received 1 day ago)
		storage4 := models.Storage{
			ID:          uuid.New().String(),
			ProductID:   prod4.ID,
			LocatorID:   locs[0].ID,
			BatchNumber: "BAT-SUGR-20260524-1",
			ReceivedAt:  time.Now().Add(-24 * time.Hour),
			QtyOnHand:   150,
			QtyReserved: 0,
			UpdatedAt:   time.Now(),
		}

		_ = database.DB.Create(&storage1).Error
		_ = database.DB.Create(&storage2).Error
		_ = database.DB.Create(&storage3).Error
		_ = database.DB.Create(&storage4).Error

		log.Println("WMS master data and mock storage lots seeded successfully.")
	}
}

func main() {
	// 0. Load .env file (ignored if not present — e.g. in production with real env vars)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	// 1. Initialize Database
	database.InitDB()

	// 2. Seed WMS Master Data
	seedWMSData()

	// 3. Initialize Go Fiber Application
	app := fiber.New(fiber.Config{
		AppName: "Omnisync WMS - Dashboard Service",
	})

	// 4. Mount Static Assets
	app.Static("/static", "./web/static")

	// 5. Public Views
	app.Get("/login", handlers.ServeLogin)
	app.Post("/auth/login-submit", handlers.HandleLogin)
	app.Get("/logout", handlers.HandleLogout)

	// 6. Secure Operational Routes with JWT Middleware
	app.Use(middleware.JWTAuth())

	// Main Views
	app.Get("/", handlers.ServeDashboard)
	app.Get("/dashboard", handlers.ServeDashboard)

	// WMS Core AJAX Fragments (HTMX)
	app.Get("/wms/inventory", handlers.GetInventoryList)
	app.Post("/wms/movements/new", handlers.CreateMovement)
	app.Post("/wms/movements/:id/claim", handlers.ClaimMovement)
	app.Post("/wms/movements/:id/journal", handlers.JournalMovement)
	app.Post("/wms/movements/:id/complete", handlers.CompleteMovement)
	app.Post("/wms/movements/:id/reject", handlers.RejectMovement)

	// --- MASTER MAINTENANCE CRUD ENDPOINTS ---
	// View lists (Operator & Admin)
	app.Get("/wms/masters/products", handlers.ServeProductsMaster)
	app.Get("/wms/masters/warehouses", handlers.ServeWarehousesMaster)
	app.Get("/wms/masters/locators", handlers.ServeLocatorsMaster)
	app.Get("/wms/masters/uoms", handlers.ServeUoMsMaster)

	// Form Loaders (Operator & Admin)
	app.Get("/wms/masters/products/new", handlers.ServeNewProductForm)
	app.Get("/wms/masters/products/:id/edit", handlers.ServeEditProductForm)
	app.Get("/wms/masters/warehouses/new", handlers.ServeNewWarehouseForm)
	app.Get("/wms/masters/warehouses/:id/edit", handlers.ServeEditWarehouseForm)
	app.Get("/wms/masters/locators/new", handlers.ServeNewLocatorForm)
	app.Get("/wms/masters/locators/:id/edit", handlers.ServeEditLocatorForm)
	app.Get("/wms/masters/uoms/new", handlers.ServeNewUoMForm)
	app.Get("/wms/masters/uoms/:id/edit", handlers.ServeEditUoMForm)
	app.Get("/wms/masters/conversions/new", handlers.ServeNewConversionForm)

	// Mutate actions (ADMIN ONLY)
	adminOnly := middleware.RequireAdmin()
	app.Post("/wms/masters/products", adminOnly, handlers.CreateProduct)
	app.Put("/wms/masters/products/:id", adminOnly, handlers.UpdateProduct)
	app.Delete("/wms/masters/products/:id", adminOnly, handlers.DeleteProduct)

	app.Post("/wms/masters/warehouses", adminOnly, handlers.CreateWarehouse)
	app.Put("/wms/masters/warehouses/:id", adminOnly, handlers.UpdateWarehouse)
	app.Delete("/wms/masters/warehouses/:id", adminOnly, handlers.DeleteWarehouse)

	app.Post("/wms/masters/locators", adminOnly, handlers.CreateLocator)
	app.Put("/wms/masters/locators/:id", adminOnly, handlers.UpdateLocator)
	app.Delete("/wms/masters/locators/:id", adminOnly, handlers.DeleteLocator)

	app.Post("/wms/masters/uoms", adminOnly, handlers.CreateUoM)
	app.Put("/wms/masters/uoms/:id", adminOnly, handlers.UpdateUoM)
	app.Delete("/wms/masters/uoms/:id", adminOnly, handlers.DeleteUoM)

	app.Post("/wms/masters/conversions", adminOnly, handlers.CreateConversion)
	app.Delete("/wms/masters/conversions/:id", adminOnly, handlers.DeleteConversion)

	// 7. Start the Dashboard Service
	port := os.Getenv("PORT")
	if port == "" {
		port = "9901"
	}
	log.Printf("WMS Dashboard Service starting on port %s...", port)
	log.Fatal(app.Listen(":" + port))
}
