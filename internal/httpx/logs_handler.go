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

// handleGetLogs ports GET /api/logs — returns the most recent N log lines,
// filtered by level and search. Query params: lines (default 100, max 1000),
// level (default "all"), search (default "").
func (s *Server) handleGetLogs(c *fiber.Ctx) error {
	lines := 100
	if v := c.Query("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			lines = n
		}
	}
	if lines < 1 {
		// Guard against negative/zero `lines`: all[len(all)-lines:] would index
		// out of range (panic). Fall back to the default tail size.
		lines = 100
	}
	if lines > 1000 {
		lines = 1000
	}
	level := strings.ToUpper(c.Query("level", "all"))
	search := c.Query("search", "")

	path := s.cfg.LogFile
	if _, err := os.Stat(path); err != nil {
		return c.JSON(fiber.Map{
			"success": true,
			"logs":    []string{},
			"message": "로그 파일이 아직 생성되지 않았습니다.",
		})
	}

	all, err := readAllLines(path)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
	}

	// Take the most recent `lines` lines.
	if len(all) > lines {
		all = all[len(all)-lines:]
	}

	logs := []string{}
	for _, line := range all {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if level != "ALL" {
			// ponytail: Task 3 rewrites this handler entirely — a minimal inline
			// JSON "level" field read is enough to keep the package compiling.
			lvl, ok := jsonLevel(line)
			if !ok || lvl != level {
				continue
			}
		}
		if search != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(search)) {
			continue
		}
		logs = append(logs, line)
	}

	return c.JSON(fiber.Map{
		"success":         true,
		"logs":            logs,
		"total_lines":     len(logs),
		"requested_lines": lines,
		"level_filter":    level,
		"search_filter":   search,
	})
}

// handleClearLogs ports POST /api/logs/clear — backs up then truncates the log
// file.
func (s *Server) handleClearLogs(c *fiber.Ctx) error {
	path := s.cfg.LogFile
	if _, err := os.Stat(path); err != nil {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "초기화할 로그 파일이 없습니다.",
		})
	}

	var backupFile any
	backupPath := fmt.Sprintf("app_backup_%s.log", time.Now().Format("20060102_150405"))
	if src, err := os.ReadFile(path); err == nil {
		if err := os.WriteFile(backupPath, src, 0o644); err == nil {
			backupFile = backupPath
			s.log.Info("log backup", "path", backupPath)
		} else {
			s.log.Warn("log backup failed", "err", err.Error())
		}
	} else {
		s.log.Warn("log backup failed", "err", err.Error())
	}

	if err := os.Truncate(path, 0); err != nil {
		return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
	}
	s.log.Info("logs cleared")

	return c.JSON(fiber.Map{
		"success":     true,
		"message":     "로그 파일이 초기화되었습니다.",
		"backup_file": backupFile,
	})
}

// handleDownloadLogs ports GET /api/logs/download — streams the log file as a
// timestamped attachment.
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

// jsonLevel extracts the "level" field from a JSON log line.
func jsonLevel(line string) (string, bool) {
	var m struct {
		Level string `json:"level"`
	}
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		return "", false
	}
	return m.Level, m.Level != ""
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
