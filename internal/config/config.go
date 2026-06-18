// Package config holds the runtime configuration, ported from config.py.
// It uses only the stdlib (os.Getenv + typed helpers); viper is unnecessary.
// Dropped relative to config.py: SECRET_KEY, DEBUG, DELETE_DELAY,
// UPLOAD_FOLDER, QR_CACHE_TTL and the performance-monitoring flags — all tied
// to the temp-file lifecycle / perf subsystems removed in the Go port.
package config

import (
	"os"
	"strconv"
)

// defaultVersion is the fallback /api/health version. The Makefile injects the
// VERSION file content via -ldflags "-X qrweb/internal/config.defaultVersion=...";
// without injection it stays "1.0.0" (config.py parity). APP_VERSION env still
// overrides it at runtime.
var defaultVersion = "1.0.0"

// Config is the resolved application configuration. Field defaults mirror the
// config.py Config class.
type Config struct {
	Host             string // FLASK_HOST -> HOST (0.0.0.0)
	Port             int    // FLASK_PORT -> PORT (5000)
	LogLevel         string // LOG_LEVEL (INFO)
	LogFile          string // LOG_FILE (logs/app.log) — log viewer reads this
	MaxContentLength int64  // MAX_CONTENT_LENGTH (16MB) — Fiber BodyLimit
	MaxQRFiles       int    // MAX_QR_FILES (50)
	MaxQRFileSize    int64  // MAX_QR_FILE_SIZE (2MB)
	QRBoxSize        int    // QR_BOX_SIZE (10)
	QRBorder         int    // QR_BORDER (2)
	Version          string // /api/health version string
}

// Load reads the environment and returns the Config with config.py defaults.
func Load() *Config {
	return &Config{
		Host:             getEnv("HOST", getEnv("FLASK_HOST", "0.0.0.0")),
		Port:             getEnvInt("PORT", getEnvInt("FLASK_PORT", 5000)),
		LogLevel:         getEnv("LOG_LEVEL", "INFO"),
		LogFile:          getEnv("LOG_FILE", "logs/app.log"),
		MaxContentLength: getEnvInt64("MAX_CONTENT_LENGTH", 16*1024*1024),
		MaxQRFiles:       getEnvInt("MAX_QR_FILES", 50),
		MaxQRFileSize:    getEnvInt64("MAX_QR_FILE_SIZE", 2*1024*1024),
		QRBoxSize:        getEnvInt("QR_BOX_SIZE", 10),
		QRBorder:         getEnvInt("QR_BORDER", 2),
		Version:          getEnv("APP_VERSION", defaultVersion),
	}
}

// Addr returns the host:port listen address.
func (c *Config) Addr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvInt64(key string, def int64) int64 {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}
