package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/handlers"
	"wms_dashboard/internal/middleware"
)



func main() {
	// 0. Load .env file (ignored if not present — e.g. in production with real env vars)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	// 1. Initialize Database
	database.InitDB()

	// 3. Initialize Go Fiber Application
	app := fiber.New(fiber.Config{
		AppName: "Omnisync WMS - Dashboard Service",
	})

	// 3.1 Health Check (before any middleware — used by CI/Playwright readiness probe)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
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

	// Movements (Guarded by manage_movements permission)
	movementGroup := app.Group("/wms/movements", middleware.RequireManageMovements())
	movementGroup.Get("/", handlers.ServeMovementsPage)
	movementGroup.Get("/new", handlers.ServeNewMovementPage)
	movementGroup.Post("/", handlers.CreateMovement)
	movementGroup.Get("/:id", handlers.ServeMovementDetailPage)
	movementGroup.Post("/:id/claim", handlers.ClaimMovement)
	movementGroup.Post("/:id/journal", handlers.JournalMovement)
	movementGroup.Post("/:id/complete", handlers.CompleteMovement)
	movementGroup.Post("/:id/reject", handlers.RejectMovement)
	movementGroup.Post("/:id/crossdock/inbound", handlers.ConfirmCrossDockInbound)
	movementGroup.Post("/:id/crossdock/shipping", handlers.ConfirmCrossDockShipping)
	movementGroup.Post("/:id/crossdock/outbound", handlers.ConfirmCrossDockOutbound)


	// Adjustments
	app.Get("/wms/adjustments", handlers.ServeAdjustments)
	app.Get("/wms/adjustments/locators", handlers.ServeAdjustmentLocatorsByProduct)
	app.Post("/wms/adjustments/new", handlers.CreateAdjustment)
	app.Post("/wms/adjustments/:id/journal", handlers.JournalAdjustment)
	app.Post("/wms/adjustments/:id/reject", handlers.RejectAdjustment)

	// Kitting
	app.Get("/wms/kitting", handlers.ServeKittingOrders)
	app.Get("/wms/kitting/locators", handlers.ServeKittingLocatorsByProduct)
	app.Post("/wms/kitting/new", handlers.CreateKittingOrder)
	app.Post("/wms/kitting/:id/journal", handlers.JournalKittingOrder)
	app.Post("/wms/kitting/:id/reject", handlers.RejectKittingOrder)

	// QC Holds
	app.Get("/wms/qc-holds", handlers.ServeQCHolds)
	app.Post("/wms/qc-holds", handlers.CreateQCHold)
	app.Post("/wms/qc-holds/:id/release", handlers.ReleaseQCHold)

	// Return to Vendor (RTV)
	app.Get("/wms/rtv", handlers.ServeRTV)
	app.Get("/wms/rtv/storages", handlers.ServeRTVStoragesByProduct)
	app.Post("/wms/rtv/new", handlers.CreateRTV)

	// Inventory Transfers
	app.Get("/wms/transfers", handlers.ServeTransfers)
	app.Get("/wms/transfers/source-storages", handlers.ServeTransferSourceStorages)
	app.Get("/wms/transfers/destinations", handlers.ServeTransferDestinations)
	app.Post("/wms/transfers/new", handlers.CreateTransfer)

	app.Get("/wms/ledger", middleware.RequireSystemAdmin(), handlers.ServeLedger)
	app.Get("/wms/ledger/export/pdf", middleware.RequireSystemAdmin(), handlers.ExportLedgerPDF)
	app.Get("/wms/ledger/export/excel", middleware.RequireSystemAdmin(), handlers.ExportLedgerExcel)
	app.Get("/wms/crossdock", handlers.ServeCrossDock)
	app.Get("/wms/in-progress", handlers.ServeInProgressDocs)

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

	// --- SYSTEM ADMINISTRATION ENDPOINTS (SYSTEM MANAGE ONLY) ---
	systemManageOnly := middleware.RequireSystemManage()
	app.Get("/wms/system/users", systemManageOnly, handlers.ServeUsersMaster)
	app.Get("/wms/system/users/rows", systemManageOnly, handlers.GetUsersRows)
	app.Post("/wms/system/users", systemManageOnly, handlers.CreateUser)
	app.Put("/wms/system/users/:id/status", systemManageOnly, handlers.UpdateUserStatus)

	app.Get("/wms/system/roles", systemManageOnly, handlers.ServeRolesMaster)
	app.Get("/wms/system/roles/rows", systemManageOnly, handlers.GetRolesRows)
	app.Post("/wms/system/roles", systemManageOnly, handlers.CreateRole)
	app.Put("/wms/system/roles/:id", systemManageOnly, handlers.UpdateRole)
	app.Delete("/wms/system/roles/:id", systemManageOnly, handlers.DeleteRole)

	// --- MOBILE APP REST API ENDPOINTS ---
	api := app.Group("/api/v1")
	api.Post("/auth/login", handlers.APILogin) // Public Login

	// Protected API Routes
	protectedApi := api.Group("/", middleware.JWTAuth())
	protectedApi.Get("/products/scan/:sku", handlers.APIGetProductBySKU)
	protectedApi.Get("/locators/scan/:code", handlers.APIGetLocatorByCode)
	protectedApi.Post("/movements", handlers.APICreateMovement)

	// Issue 52 Mobile APIs
	protectedApi.Get("/movements", handlers.APIListMovements)
	protectedApi.Get("/movements/:id", handlers.APIGetMovementByID)
	protectedApi.Post("/movements/:id/claim", handlers.APIClaimMovement)
	protectedApi.Post("/movements/:id/scan-verify", handlers.APIScanVerifyMovementLine)
	protectedApi.Post("/movements/:id/submit", handlers.APISubmitMovement)

	// 7. Start the Dashboard Service
	port := os.Getenv("PORT")
	if port == "" {
		port = "9901"
	}
	log.Printf("WMS Dashboard Service starting on port %s...", port)
	log.Fatal(app.Listen(":" + port))
}
