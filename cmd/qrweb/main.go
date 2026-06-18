// Command qrweb is the QR Web Label Generator server (Go/Fiber port of app.py).
// It loads config from the environment, initializes the file+stdout logger, and
// serves the HTTP API. The static SPA is embedded in Phase 6; until then "/"
// returns 204.
package main

import (
	"log"

	"qrweb/internal/config"
	"qrweb/internal/httpx"
	"qrweb/internal/logging"
)

func main() {
	cfg := config.Load()

	logger, err := logging.Init(cfg.LogFile, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	logger.Info("Starting QR Web application on %s (level=%s)", cfg.Addr(), cfg.LogLevel)

	srv := httpx.New(cfg, logger)
	if err := srv.Listen(); err != nil {
		logger.Error("server exited: %v", err)
		log.Fatalf("server exited: %v", err)
	}
}
