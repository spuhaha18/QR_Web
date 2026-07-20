package httpx

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// logEntry is one parsed log line as served to the viewer. Known top-level
// slog keys (time/level/msg) are lifted out; every other key lands in Fields.
// Lines that fail JSON parsing (pre-migration text format, corrupt tails) are
// preserved verbatim as legacy INFO entries rather than dropped.
type logEntry struct {
	Time   string         `json:"time"`
	Level  string         `json:"level"`
	Msg    string         `json:"msg"`
	Legacy bool           `json:"legacy,omitempty"`
	Fields map[string]any `json:"fields"`
}

func parseLogLine(line string) logEntry {
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		return logEntry{Level: "INFO", Msg: line, Legacy: true, Fields: map[string]any{}}
	}
	e := logEntry{Fields: make(map[string]any, len(m))}
	for k, v := range m {
		switch k {
		case "time":
			e.Time, _ = v.(string)
		case "level":
			e.Level, _ = v.(string)
		case "msg":
			e.Msg, _ = v.(string)
		default:
			e.Fields[k] = v
		}
	}
	return e
}

// normalizeLevel maps the legacy "WARNING" spelling (and lowercase input) onto
// slog's "WARN" so both filter values keep working.
func normalizeLevel(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "WARNING" {
		return "WARN"
	}
	return s
}

// handleGetLogs returns the most recent N parsed log entries, filtered by
// level and request_id. Text search moved client-side with the Korean
// rendering; there is deliberately no search param anymore.
func (s *Server) handleGetLogs(c *fiber.Ctx) error {
	lines := 100
	if v := c.Query("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			lines = n
		}
	}
	if lines < 1 {
		lines = 100
	}
	if lines > 1000 {
		lines = 1000
	}
	level := normalizeLevel(c.Query("level", "all"))
	reqID := c.Query("request_id", "")

	path := s.cfg.LogFile
	if _, err := os.Stat(path); err != nil {
		return c.JSON(fiber.Map{
			"success": true,
			"logs":    []logEntry{},
			"message": "로그 파일이 아직 생성되지 않았습니다.",
		})
	}

	all, err := readAllLines(path)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
	}
	if len(all) > lines {
		all = all[len(all)-lines:]
	}

	logs := []logEntry{}
	for _, line := range all {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		e := parseLogLine(line)
		if level != "ALL" && e.Level != level {
			continue
		}
		if reqID != "" {
			if id, _ := e.Fields["request_id"].(string); id != reqID {
				continue
			}
		}
		logs = append(logs, e)
	}

	return c.JSON(fiber.Map{
		"success":         true,
		"logs":            logs,
		"total_lines":     len(logs),
		"requested_lines": lines,
		"level_filter":    level,
	})
}

// handleClearLogs empties the log via the logger's rotate-then-delete-backups
// path (owning lumberjack's size accounting), then logs the action.
func (s *Server) handleClearLogs(c *fiber.Ctx) error {
	if err := s.log.Clear(); err != nil {
		s.log.Error("log clear failed", "err", err.Error(), "request_id", requestID(c))
		return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
	}
	s.log.Info("logs cleared", "request_id", requestID(c))
	return c.JSON(fiber.Map{
		"success": true,
		"message": "로그 파일이 초기화되었습니다.",
	})
}

// handleDownloadLogs streams the current log file as a timestamped attachment.
// Rotated backups are reachable on the volume, not through the API.
func (s *Server) handleDownloadLogs(c *fiber.Ctx) error {
	path := s.cfg.LogFile
	data, err := os.ReadFile(path)
	if err != nil {
		return errJSON(c, fiber.StatusNotFound, "다운로드할 로그 파일이 없습니다.")
	}
	name := fmt.Sprintf("app_logs_%s.log", time.Now().Format("20060102_150405"))
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	c.Set(fiber.HeaderContentType, "text/plain; charset=utf-8")
	return c.Send(data)
}

// readAllLines reads all lines from path (without trailing newlines).
func readAllLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}
