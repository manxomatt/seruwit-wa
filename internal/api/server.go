package api

import (
	"seruwit-wa/internal/config"
	"seruwit-wa/internal/whatsapp"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func NewServer(cfg *config.Config, manager *whatsapp.Manager) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Seruwit WA Gateway (whatsmeow)",
		ServerHeader: "Seruwit-WA",
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Apply API Key security
	app.Use(APIKeyMiddleware(cfg.APIKey))

	handler := NewAPI(manager)

	// Health check
	app.Get("/health", handler.HealthCheck)

	// Sessions API
	api := app.Group("/api")
	sessions := api.Group("/sessions")
	sessions.Post("/start", handler.StartSession)
	sessions.Post("/pair-code", handler.PairPhone)
	sessions.Get("/:id/status", handler.GetStatus)
	sessions.Post("/:id/logout", handler.Logout)

	// Messages API
	messages := api.Group("/messages")
	messages.Post("/send-text", handler.SendTextMessage)

	return app
}
