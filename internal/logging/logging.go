// Package logging provides a file+stdout logger ported from the logging.basicConfig
// setup in app.py. Log lines are written to a single file (config.LogFile) that
// the log-viewer endpoints (GET /api/logs etc.) read back. The line format
// approximates the Python format string
//
//	%(asctime)s %(levelname)s %(name)s %(threadName)s : %(message)s
//
// so that the log viewer's level/search filters behave the same.
package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Level is a syslog-style severity. Ordering matches Python's logging levels.
type Level int

const (
	DEBUG Level = iota
	INFO
	WARNING
	ERROR
)

func parseLevel(s string) Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return DEBUG
	case "WARNING", "WARN":
		return WARNING
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// LevelOf extracts the severity from a formatted log line, reporting false if
// the line does not match the write() format
// "DATE TIME LEVEL logger thread : message" (LEVEL is the 3rd space-field).
// The log viewer uses this instead of a substring match so that a message
// merely containing "INFO" is not mistaken for an INFO-level line, and so the
// coupling to the line format lives here, with the format's owner.
func LevelOf(line string) (Level, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return 0, false
	}
	switch fields[2] {
	case "DEBUG":
		return DEBUG, true
	case "INFO":
		return INFO, true
	case "WARNING":
		return WARNING, true
	case "ERROR":
		return ERROR, true
	default:
		return 0, false
	}
}

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	default:
		return "INFO"
	}
}

// Logger writes formatted records to a file and to stdout, filtered by minimum
// level.
type Logger struct {
	mu       sync.Mutex
	min      Level
	file     *os.File
	stdout   io.Writer
	loggerNm string
}

var std *Logger

// Init creates (and on subsequent calls replaces) the process-wide logger,
// writing to logFile with the given minimum level. The parent directory is
// created if missing. Returns the logger and the resolved log file path.
func Init(logFile, level string) (*Logger, error) {
	if logFile == "" {
		logFile = "logs/app.log"
	}
	if dir := filepath.Dir(logFile); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	std = &Logger{
		min:      parseLevel(level),
		file:     f,
		stdout:   os.Stdout,
		loggerNm: "app",
	}
	return std, nil
}

// Default returns the process-wide logger, or a no-op-ish stdout logger if Init
// was never called (e.g. in tests).
func Default() *Logger {
	if std == nil {
		std = &Logger{min: INFO, stdout: os.Stdout, loggerNm: "app"}
	}
	return std
}

func (l *Logger) write(lvl Level, msg string) {
	if l == nil || lvl < l.min {
		return
	}
	// %(asctime)s %(levelname)s %(name)s %(threadName)s : %(message)s
	line := fmt.Sprintf("%s %s %s %s : %s\n",
		time.Now().Format("2006-01-02 15:04:05,000"),
		lvl.String(),
		l.loggerNm,
		"MainThread",
		msg,
	)
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_, _ = l.file.WriteString(line)
	}
	if l.stdout != nil {
		_, _ = io.WriteString(l.stdout, line)
	}
}

// Debug logs at DEBUG level.
func (l *Logger) Debug(format string, a ...any) { l.write(DEBUG, fmt.Sprintf(format, a...)) }

// Info logs at INFO level.
func (l *Logger) Info(format string, a ...any) { l.write(INFO, fmt.Sprintf(format, a...)) }

// Warn logs at WARNING level.
func (l *Logger) Warn(format string, a ...any) { l.write(WARNING, fmt.Sprintf(format, a...)) }

// Error logs at ERROR level.
func (l *Logger) Error(format string, a ...any) { l.write(ERROR, fmt.Sprintf(format, a...)) }

// Close flushes and closes the underlying file.
func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}
