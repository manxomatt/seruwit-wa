package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func APIKeyMiddleware(expectedKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Bypass auth for health check
		if c.Path() == "/health" {
			return c.Next()
		}

		key := c.Get("X-API-Key")
		if key == "" {
			authHeader := c.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				key = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if key == "" || key != expectedKey {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Unauthorized: Invalid or missing API key",
			})
		}

		return c.Next()
	}
}
