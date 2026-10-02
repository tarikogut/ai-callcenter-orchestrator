package api

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
	"gorm.io/gorm"
)

// SetupApp configures the Fiber application, middleware, and routes.
func SetupApp(database *gorm.DB) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "AI Call Center Orchestrator v1.0",
	})

	// Global Middleware
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Tenant-ID",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "15:04:05",
	}))

	// Health Check
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "ai-callcenter-orchestrator",
			"version": "1.0.0",
		})
	})

	handler := NewAPIHandler(database)
	api := app.Group("/api/v1")

	// ==========================================
	// ADMIN ROUTES
	// ==========================================
	admin := api.Group("/admin")
	tenants := admin.Group("/tenants")
	tenants.Get("/", handler.ListTenants)
	tenants.Post("/", handler.CreateTenant)
	tenants.Get("/:id", handler.GetTenant)
	tenants.Put("/:id", handler.UpdateTenant)
	tenants.Delete("/:id", handler.DeleteTenant)

	// ==========================================
	// CUSTOMER ROUTES
	// ==========================================
	customer := api.Group("/customer")

	// Extensions
	extensions := customer.Group("/extensions")
	extensions.Get("/", handler.ListExtensions)
	extensions.Post("/", handler.CreateExtension)
	extensions.Get("/:id", handler.GetExtension)
	extensions.Put("/:id", handler.UpdateExtension)
	extensions.Delete("/:id", handler.DeleteExtension)

	// DIDs
	dids := customer.Group("/dids")
	dids.Get("/", handler.ListDIDs)
	dids.Post("/", handler.CreateDID)
	dids.Get("/:id", handler.GetDID)
	dids.Put("/:id", handler.UpdateDID)
	dids.Delete("/:id", handler.DeleteDID)

	// Workflows
	workflows := customer.Group("/workflows")
	workflows.Get("/", handler.ListWorkflows)
	workflows.Post("/", handler.CreateWorkflow)
	workflows.Get("/:id", handler.GetWorkflow)
	workflows.Put("/:id", handler.UpdateWorkflow)
	workflows.Delete("/:id", handler.DeleteWorkflow)

	// Tenant Settings / Persona / FAQ / BYOK
	settings := customer.Group("/settings")
	settings.Get("/", handler.GetSettings)
	settings.Get("/:tenant_id", handler.GetSettings)
	settings.Post("/", handler.UpsertSettings)
	settings.Put("/", handler.UpsertSettings)
	settings.Put("/:tenant_id", handler.UpsertSettings)

	// CDRs & Call History
	cdr := customer.Group("/cdr")
	cdr.Get("/", handler.ListCDRs)
	cdr.Post("/", handler.CreateCDR)
	cdr.Get("/:id", handler.GetCDR)

	// ==========================================
	// WEBHOOK ROUTES (Kamailio SMS-IWF & Telephony)
	// ==========================================
	webhook := api.Group("/webhook")
	webhook.Post("/sms", handler.HandleSmsWebhook)

	// ==========================================
	// WEBSOCKET ROUTES (Live Event & Call Stream)
	// ==========================================
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws/events", websocket.New(func(c *websocket.Conn) {
		tenantID := c.Query("tenant_id")
		log.Printf("[WebSocket] Client connected: tenant=%s, remote=%s", tenantID, c.RemoteAddr())
		defer func() {
			log.Printf("[WebSocket] Client disconnected: tenant=%s", tenantID)
			c.Close()
		}()

		// Welcome message
		if err := c.WriteJSON(fiber.Map{
			"event":     "connected",
			"tenant_id": tenantID,
			"message":   "Connected to AI Call Center Orchestrator live stream",
		}); err != nil {
			return
		}

		for {
			mt, msg, err := c.ReadMessage()
			if err != nil {
				break
			}
			// Echo or handle client ping/commands
			if err := c.WriteMessage(mt, msg); err != nil {
				break
			}
		}
	}))

	return app
}
