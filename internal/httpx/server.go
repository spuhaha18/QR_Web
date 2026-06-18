// Package httpx holds the Fiber HTTP layer, porting the routes/handlers of
// app.py. The temp-file lifecycle (uploads/, /download, file_lifecycle) is
// removed: .xlsx is generated in memory and streamed directly. The performance
// and system endpoints are dropped; the log viewer endpoints are kept.
package httpx

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"qrweb/internal/config"
	"qrweb/internal/excel"
	"qrweb/internal/logging"
)

// Server bundles the Fiber app with its dependencies.
type Server struct {
	app *fiber.App
	cfg *config.Config
	log *logging.Logger
	gen *excel.Generator
}

// New builds the Fiber app, registers middleware and all routes, and returns
// the Server. API routes are registered before any static serving so they are
// never shadowed by the SPA fallback (added in Phase 6).
func New(cfg *config.Config, log *logging.Logger) *Server {
	app := fiber.New(fiber.Config{
		BodyLimit:             int(cfg.MaxContentLength), // MAX_CONTENT_LENGTH (16MB)
		DisableStartupMessage: true,
		// recover middleware below converts panics into the Korean 500 JSON;
		// keep Fiber's default error handler for non-panic *fiber.Error.
	})

	s := &Server{
		app: app,
		cfg: cfg,
		log: log,
		gen: excel.NewGenerator(),
	}

	// recover: panic -> {"error":"서버 오류가 발생했습니다."} 500 (handle_errors parity).
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	// request logging via our file+stdout logger.
	app.Use(s.requestLogger())

	s.registerRoutes()
	return s
}

// App exposes the underlying Fiber app (used by tests via app.Test).
func (s *Server) App() *fiber.App { return s.app }

// Listen starts the HTTP server on cfg.Addr().
func (s *Server) Listen() error {
	return s.app.Listen(s.cfg.Addr())
}

// registerRoutes wires every endpoint. API/explicit routes come first; the SPA
// placeholder ("/") is last.
func (s *Server) registerRoutes() {
	// Label creation.
	s.app.Post("/create_label", s.handleCreateLabelPaste)    // paste mode, multipart -> .xlsx
	s.app.Post("/api/create_label", s.handleCreateLabelAuto) // auto mode, JSON -> .xlsx (base64)

	// QR image.
	s.app.Get("/api/qr_image/:text", s.handleQRImage)
	s.app.Post("/api/qr_image_base64", s.handleQRImageBase64)

	// Health.
	s.app.Get("/api/health", s.handleHealth)

	// Log viewer.
	s.app.Get("/api/logs", s.handleGetLogs)
	s.app.Post("/api/logs/clear", s.handleClearLogs)
	s.app.Get("/api/logs/download", s.handleDownloadLogs)

	// SPA placeholder (replaced by embedded Vite build in Phase 6).
	s.app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNoContent).SendString("")
	})
	// Log viewer page placeholder (SPA route in Phase 6).
	s.app.Get("/logs", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNoContent).SendString("")
	})
}

// requestLogger logs each request after it completes (method, path, status).
func (s *Server) requestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		s.log.Info("%s %s -> %d from %s", c.Method(), c.Path(), c.Response().StatusCode(), c.IP())
		return err
	}
}

// errJSON writes {"error": msg} with the given status, matching Flask's
// jsonify({'error': ...}), status.
func errJSON(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(fiber.Map{"error": msg})
}
