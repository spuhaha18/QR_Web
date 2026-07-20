package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// decodeLines parses each non-empty line of buf as a JSON object.
func decodeLines(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("line not JSON: %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

func TestNewWritesJSONWithFields(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, slog.LevelInfo)
	l.Info("label generated", "file", "a.pdf", "ip", "10.0.0.5")

	recs := decodeLines(t, buf.Bytes())
	if len(recs) != 1 {
		t.Fatalf("want 1 record, got %d", len(recs))
	}
	r := recs[0]
	if r["level"] != "INFO" || r["msg"] != "label generated" || r["file"] != "a.pdf" || r["ip"] != "10.0.0.5" {
		t.Errorf("unexpected record: %v", r)
	}
	if _, ok := r["time"].(string); !ok {
		t.Errorf("missing time field: %v", r)
	}
}

func TestNewFiltersBelowMinLevel(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, slog.LevelInfo)
	l.Debug("hidden")
	l.Warn("shown")

	recs := decodeLines(t, buf.Bytes())
	if len(recs) != 1 || recs[0]["level"] != "WARN" {
		t.Fatalf("want only WARN record, got %v", recs)
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"DEBUG": slog.LevelDebug, "debug": slog.LevelDebug,
		"INFO": slog.LevelInfo, "": slog.LevelInfo, "junk": slog.LevelInfo,
		"WARN": slog.LevelWarn, "WARNING": slog.LevelWarn,
		"ERROR": slog.LevelError,
	}
	for in, want := range cases {
		if got := parseLevel(in); got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestInitWritesFileAndClearRemovesEverything(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	l, err := Init(path, "INFO", 10, 5)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	l.Info("server started", "addr", "0.0.0.0:5000")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not written: %v", err)
	}
	recs := decodeLines(t, data)
	if len(recs) != 1 || recs[0]["msg"] != "server started" {
		t.Fatalf("unexpected file content: %v", recs)
	}

	if err := l.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	// Current file must be empty (or freshly recreated), no backups left.
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		t.Errorf("current file not empty after Clear: %q", data)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "app.log" {
			t.Errorf("backup survived Clear: %s", e.Name())
		}
	}
}

func TestClearOnStdoutOnlyLoggerIsNoop(t *testing.T) {
	l := New(&bytes.Buffer{}, slog.LevelInfo)
	if err := l.Clear(); err != nil {
		t.Fatalf("Clear on stdout-only logger: %v", err)
	}
}
