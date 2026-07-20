package httpx

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"qrweb/internal/config"
	"qrweb/internal/logging"
)

type logsResponse struct {
	Success bool `json:"success"`
	Logs    []struct {
		Time   string         `json:"time"`
		Level  string         `json:"level"`
		Msg    string         `json:"msg"`
		Legacy bool           `json:"legacy"`
		Fields map[string]any `json:"fields"`
	} `json:"logs"`
	TotalLines int `json:"total_lines"`
}

// newLogsServer builds a Server whose cfg.LogFile points at a temp file
// pre-populated with the given lines.
func newLogsServer(t *testing.T, lines ...string) *Server {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app.log")
	var buf bytes.Buffer
	for _, l := range lines {
		buf.WriteString(l + "\n")
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Load()
	cfg.LogFile = path
	return New(cfg, logging.New(io.Discard, slog.LevelInfo))
}

func getLogs(t *testing.T, s *Server, query string) logsResponse {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/logs"+query, nil)
	resp, err := s.App().Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out logsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestGetLogsParsesJSONLines(t *testing.T) {
	s := newLogsServer(t,
		`{"time":"2026-07-20T14:00:00+09:00","level":"INFO","msg":"label generated","file":"a.pdf","request_id":"abc123"}`,
		`{"time":"2026-07-20T14:00:01+09:00","level":"WARN","msg":"request","status":404}`,
	)
	out := getLogs(t, s, "")
	if !out.Success || len(out.Logs) != 2 {
		t.Fatalf("unexpected response: %+v", out)
	}
	e := out.Logs[0]
	if e.Msg != "label generated" || e.Level != "INFO" || e.Fields["file"] != "a.pdf" {
		t.Errorf("first entry wrong: %+v", e)
	}
	if _, leaked := e.Fields["msg"]; leaked {
		t.Error("msg must not appear in fields")
	}
}

func TestGetLogsLegacyLineFallback(t *testing.T) {
	s := newLogsServer(t, `2026-06-01 12:00:00,000 INFO app MainThread : old line`)
	out := getLogs(t, s, "")
	if len(out.Logs) != 1 {
		t.Fatalf("want 1 entry, got %+v", out)
	}
	e := out.Logs[0]
	if !e.Legacy || e.Level != "INFO" || e.Msg != "2026-06-01 12:00:00,000 INFO app MainThread : old line" {
		t.Errorf("legacy entry wrong: %+v", e)
	}
}

func TestGetLogsLevelFilterAcceptsBothSpellings(t *testing.T) {
	s := newLogsServer(t,
		`{"time":"t","level":"INFO","msg":"a"}`,
		`{"time":"t","level":"WARN","msg":"b"}`,
	)
	for _, q := range []string{"?level=WARN", "?level=WARNING", "?level=warning"} {
		out := getLogs(t, s, q)
		if len(out.Logs) != 1 || out.Logs[0].Msg != "b" {
			t.Errorf("%s: got %+v", q, out.Logs)
		}
	}
}

func TestGetLogsRequestIDFilter(t *testing.T) {
	s := newLogsServer(t,
		`{"time":"t","level":"INFO","msg":"request","request_id":"aaa"}`,
		`{"time":"t","level":"INFO","msg":"request","request_id":"bbb"}`,
	)
	out := getLogs(t, s, "?request_id=bbb")
	if len(out.Logs) != 1 || out.Logs[0].Fields["request_id"] != "bbb" {
		t.Errorf("got %+v", out.Logs)
	}
}

func TestGetLogsNegativeLinesDefaults(t *testing.T) {
	s := newLogsServer(t, `{"time":"t","level":"INFO","msg":"a"}`)
	out := getLogs(t, s, "?lines=-1")
	if !out.Success || len(out.Logs) != 1 {
		t.Errorf("got %+v", out)
	}
}

func TestClearLogsUsesRotateAndReportsNoBackupField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	l, err := logging.Init(path, "INFO", 10, 5)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	l.Info("before clear")

	cfg := config.Load()
	cfg.LogFile = path
	s := New(cfg, l)

	req := httptest.NewRequest("POST", "/api/logs/clear", nil)
	resp, err := s.App().Test(req)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["success"] != true {
		t.Fatalf("clear failed: %v", body)
	}
	if _, exists := body["backup_file"]; exists {
		t.Error("backup_file must be gone from the response")
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "app.log" {
			t.Errorf("backup file survived: %s", e.Name())
		}
	}
	// The clear itself is logged afterwards, so the file has exactly the
	// "logs cleared" record.
	data, _ := os.ReadFile(path)
	if !bytes.Contains(data, []byte(`"logs cleared"`)) {
		t.Errorf("expected only the logs-cleared record, got %q", data)
	}
	if bytes.Contains(data, []byte("before clear")) {
		t.Errorf("old content survived clear: %q", data)
	}
}
