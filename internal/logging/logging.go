// Package logging provides the process-wide structured logger: slog JSON lines
// to a size-rotated file (lumberjack) plus stdout. The log-viewer endpoints
// (GET /api/logs etc.) parse the same JSON lines back; the Svelte viewer
// renders known msg keys in Korean, so msg values must stay stable English
// event keys ("request", "label generated", ...) with all variable data in
// slog fields.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger is a slog.Logger plus ownership of the rotated log file.
type Logger struct {
	*slog.Logger
	lj *lumberjack.Logger // nil for stdout-only loggers (tests, Default before Init)
}

var std *Logger

// New returns a JSON logger writing to w, filtered by min level. Used by tests
// and as the pre-Init fallback.
func New(w io.Writer, min slog.Level) *Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: min})
	return &Logger{Logger: slog.New(h)}
}

// Init creates (and on subsequent calls replaces) the process-wide logger,
// writing JSON lines to logFile (rotated at maxSizeMB, keeping maxBackups
// rotated files) and to stdout.
func Init(logFile, level string, maxSizeMB, maxBackups int) (*Logger, error) {
	if logFile == "" {
		logFile = "logs/app.log"
	}
	lj := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxSizeMB,
		MaxBackups: maxBackups,
	}
	// lumberjack defers opening the file to the first write, and a slog
	// handler swallows write errors — so a bad log path (e.g. a
	// mis-mounted Docker volume) would otherwise boot the server with
	// silently-dead file logging. Probe writability now and fail fast.
	if _, err := lj.Write(nil); err != nil {
		return nil, err
	}
	h := slog.NewJSONHandler(io.MultiWriter(lj, os.Stdout), &slog.HandlerOptions{
		Level: parseLevel(level),
	})
	std = &Logger{Logger: slog.New(h), lj: lj}
	return std, nil
}

// Default returns the process-wide logger, or a stdout-only INFO logger if
// Init was never called (e.g. in tests).
func Default() *Logger {
	if std == nil {
		std = New(os.Stdout, slog.LevelInfo)
	}
	return std
}

func parseLevel(s string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Clear empties the log: rotates the current file via lumberjack (keeping its
// internal size accounting consistent — never os.Truncate under lumberjack)
// and deletes every rotated backup. No-op for stdout-only loggers.
func (l *Logger) Clear() error {
	if l == nil || l.lj == nil {
		return nil
	}
	if err := l.lj.Rotate(); err != nil {
		return err
	}
	// lumberjack backups live next to the file as <name>-<timestamp><ext>.
	dir := filepath.Dir(l.lj.Filename)
	base := filepath.Base(l.lj.Filename)
	ext := filepath.Ext(base)
	prefix := strings.TrimSuffix(base, ext) + "-"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if n := e.Name(); strings.HasPrefix(n, prefix) && strings.HasSuffix(n, ext) {
			_ = os.Remove(filepath.Join(dir, n))
		}
	}
	return nil
}

// Close closes the underlying rotated file.
func (l *Logger) Close() error {
	if l == nil || l.lj == nil {
		return nil
	}
	return l.lj.Close()
}
