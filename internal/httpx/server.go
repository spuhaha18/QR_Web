// Package httpx holds the Fiber HTTP layer, porting the routes/handlers of
// app.py. The temp-file lifecycle (uploads/, /download, file_lifecycle) is
// removed: the label PDF is generated in memory and streamed directly. The performance
// and system endpoints are dropped; the log viewer endpoints are kept.
package httpx

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"qrweb/internal/config"
	"qrweb/internal/label"
	"qrweb/internal/logging"
	"qrweb/internal/pdf"
	"qrweb/internal/qr"
	"qrweb/web"
)

// Server bundles the Fiber app with its dependencies.
type Server struct {
	app *fiber.App
	cfg *config.Config
	log *logging.Logger
	gen *pdf.Generator
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
		gen: pdf.NewGenerator(),
	}

	// recover: panic -> {"error":"서버 오류가 발생했습니다."} 500 (handle_errors parity).
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	// Per-request ID: reuses an incoming X-Request-ID, else generates a UUID;
	// echoed on the response header and attached to every log record.
	app.Use(requestid.New())
	// Access log for business routes only — the log viewer's own traffic and
	// static assets would drown the signal.
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

// registerRoutes wires every endpoint. API/explicit routes are registered
// FIRST so they are never shadowed by the SPA static middleware / fallback;
// the embedded SPA is mounted last as a catch-all at "/".
func (s *Server) registerRoutes() {
	// Label creation.
	s.app.Post("/create_label", s.handleCreateLabelPaste)    // paste mode, multipart -> .pdf
	s.app.Post("/api/create_label", s.handleCreateLabelAuto) // auto mode, JSON -> .pdf (base64)

	// QR image.
	s.app.Get("/api/qr_image/*", s.handleQRImage)
	s.app.Post("/api/qr_image_base64", s.handleQRImageBase64)

	// Health.
	s.app.Get("/api/health", s.handleHealth)

	// Log viewer.
	s.app.Get("/api/logs", s.handleGetLogs)
	s.app.Post("/api/logs/clear", s.handleClearLogs)
	s.app.Get("/api/logs/download", s.handleDownloadLogs)

	// Embedded SPA (Vite/Svelte build). Mounted at "/" AFTER every API route so
	// /create_label and /api/* take precedence. NotFoundFile serves index.html
	// for any unmatched path (client-side routing fallback, e.g. /logs).
	s.app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(web.DistFS()),
		Index:        "index.html",
		NotFoundFile: "index.html",
	}))
}

// requestLogger writes one "request" record per business request. Viewer
// endpoints (/api/logs*) and static/SPA paths are excluded so reading logs
// does not generate logs.
func (s *Server) requestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		path := c.Path()
		if !accessLogged(path) {
			return err
		}
		s.log.Info("request",
			"method", c.Method(),
			"path", path,
			"status", c.Response().StatusCode(),
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", c.IP(),
			"request_id", requestID(c),
		)
		return err
	}
}

// accessLogged reports whether path belongs to the access log: API routes
// minus the log viewer, plus the non-/api label route.
func accessLogged(path string) bool {
	if path == "/create_label" {
		return true
	}
	return strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/api/logs")
}

// requestID returns the Fiber requestid middleware's ID for this request.
func requestID(c *fiber.Ctx) string {
	if v, ok := c.Locals("requestid").(string); ok {
		return v
	}
	return ""
}

// errJSON writes {"error": msg} with the given status, matching Flask's
// jsonify({'error': ...}), status.
func errJSON(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(fiber.Map{"error": msg})
}

// fail is the single domain-error → HTTP mapping. User-facing validation errors
// (label.ErrValidation, qr.ErrInvalidText) become 400 with the error's exact
// Korean message; everything else is an internal 500 with the generic message.
// Handlers return fail(c, err) instead of repeating the errors.Is/errJSON
// branch at every call site. New error classes are registered here, once.
func fail(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, label.ErrValidation), errors.Is(err, qr.ErrInvalidText):
		return errJSON(c, fiber.StatusBadRequest, label.ValidationMessage(err))
	default:
		return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
	}
}
