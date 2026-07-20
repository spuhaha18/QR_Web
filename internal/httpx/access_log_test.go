package httpx

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"qrweb/internal/config"
	"qrweb/internal/logging"
)

// newCaptureServer builds a Server whose logger writes JSON lines into buf.
func newCaptureServer(t *testing.T) (*Server, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	cfg := config.Load()
	cfg.LogFile = ""
	return New(cfg, logging.New(&buf, slog.LevelInfo)), &buf
}

func logRecords(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("non-JSON log line %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

func TestAccessLogOnAPIRoute(t *testing.T) {
	s, buf := newCaptureServer(t)
	req := httptest.NewRequest("GET", "/api/health", nil)
	resp, err := s.App().Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get("X-Request-ID") == "" {
		t.Error("missing X-Request-ID response header")
	}
	recs := logRecords(t, buf)
	if len(recs) != 1 {
		t.Fatalf("want 1 access log record, got %d: %v", len(recs), recs)
	}
	r := recs[0]
	if r["msg"] != "request" || r["method"] != "GET" || r["path"] != "/api/health" {
		t.Errorf("unexpected record: %v", r)
	}
	if r["status"] != float64(200) {
		t.Errorf("status = %v, want 200", r["status"])
	}
	if _, ok := r["duration_ms"]; !ok {
		t.Error("missing duration_ms")
	}
	if id, _ := r["request_id"].(string); id == "" {
		t.Error("missing request_id")
	}
}

func TestAccessLogSkipsLogsEndpointsAndStatic(t *testing.T) {
	s, buf := newCaptureServer(t)
	for _, path := range []string{"/api/logs?lines=10", "/", "/assets/nope.js"} {
		req := httptest.NewRequest("GET", path, nil)
		if _, err := s.App().Test(req); err != nil {
			t.Fatal(err)
		}
	}
	if recs := logRecords(t, buf); len(recs) != 0 {
		t.Errorf("expected no access log records, got %v", recs)
	}
}

func TestAccessLogIncludesCreateLabelRoute(t *testing.T) {
	s, buf := newCaptureServer(t)
	req := httptest.NewRequest("POST", "/create_label", nil)
	if _, err := s.App().Test(req); err != nil {
		t.Fatal(err)
	}
	recs := logRecords(t, buf)
	if len(recs) < 1 {
		t.Fatal("expected access log record for /create_label")
	}
	if recs[len(recs)-1]["path"] != "/create_label" {
		t.Errorf("unexpected records: %v", recs)
	}
}
