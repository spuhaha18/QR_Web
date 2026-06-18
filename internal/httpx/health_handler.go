package httpx

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// handleHealth ports GET /api/health -> {status, timestamp, version}.
func (s *Server) handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   s.cfg.Version,
	})
}
