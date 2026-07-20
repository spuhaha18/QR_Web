// Command qrweb is the QR Web Label Generator server (Go/Fiber port of app.py).
// It loads config from the environment, initializes the file+stdout logger, and
// serves the HTTP API plus the embedded Vite/Svelte SPA (web/dist) as a single
// static binary.
package main

import (
	"log"

	"qrweb/internal/config"
	"qrweb/internal/httpx"
	"qrweb/internal/logging"
)

func main() {
	cfg := config.Load()

	logger, err := logging.Init(cfg.LogFile, cfg.LogLevel, cfg.LogMaxSizeMB, cfg.LogMaxBackups)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	logger.Info("server started", "addr", cfg.Addr(), "level", cfg.LogLevel)

	srv := httpx.New(cfg, logger)
	if err := srv.Listen(); err != nil {
		logger.Error("server exited", "err", err.Error())
		log.Fatalf("server exited: %v", err)
	}
}
