package api

import (
	"context"
	"time"

	"seruwit-wa/internal/whatsapp"

	"github.com/gofiber/fiber/v2"
)

type API struct {
	manager *whatsapp.Manager
}

func NewAPI(manager *whatsapp.Manager) *API {
	return &API{manager: manager}
}

type StartSessionRequest struct {
	SessionID string `json:"session_id"`
}

type PairCodeRequest struct {
	SessionID string `json:"session_id"`
	Phone     string `json:"phone"`
}

type SendMessageRequest struct {
	SessionID string `json:"session_id"`
	To        string `json:"to"`
	Message   string `json:"message"`
}

func (a *API) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}

func (a *API) StartSession(c *fiber.Ctx) error {
	var req StartSessionRequest
	if err := c.BodyParser(&req); err != nil || req.SessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request: session_id is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sess, err := a.manager.StartSession(ctx, req.SessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"session_id": sess.SessionID,
		"status":     sess.GetStatus(),
		"qr_base64":  sess.GetQR(),
	})
}

func (a *API) PairPhone(c *fiber.Ctx) error {
	var req PairCodeRequest
	if err := c.BodyParser(&req); err != nil || req.SessionID == "" || req.Phone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request: session_id and phone are required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	code, err := a.manager.PairWithCode(ctx, req.SessionID, req.Phone)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":     true,
		"session_id":  req.SessionID,
		"pairing_code": code,
	})
}

func (a *API) GetStatus(c *fiber.Ctx) error {
	sessionID := c.Params("id")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Session ID parameter is required",
		})
	}

	sess, err := a.manager.GetSession(sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"session_id": sess.SessionID,
		"status":     sess.GetStatus(),
		"phone":      sess.Phone,
		"push_name":  sess.PushName,
		"qr_base64":  sess.GetQR(),
	})
}

func (a *API) Logout(c *fiber.Ctx) error {
	sessionID := c.Params("id")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Session ID parameter is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.manager.Logout(ctx, sessionID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Session logged out and removed successfully",
	})
}

func (a *API) SendTextMessage(c *fiber.Ctx) error {
	var req SendMessageRequest
	if err := c.BodyParser(&req); err != nil || req.SessionID == "" || req.To == "" || req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request: session_id, to, and message are required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	msgID, err := a.manager.SendTextMessage(ctx, req.SessionID, req.To, req.Message)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"message_id": msgID,
		"to":         req.To,
	})
}
