# 로그 기능 개선 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 커스텀 텍스트 로거를 slog JSON lines + lumberjack 로테이션으로 교체하고, request ID 추적과 한글 렌더 로그 뷰어를 얹는다.

**Architecture:** `internal/logging`을 stdlib `log/slog`(JSON 핸들러) + `lumberjack.v2`(크기 로테이션) 래퍼로 재작성. Fiber `requestid` 미들웨어로 요청 추적, 액세스 로그는 `/api/*`(로그 뷰어 제외)+`/create_label`만. 뷰어 API는 JSON lines를 파싱해 객체 배열로 반환, Svelte 뷰어가 프론트 카탈로그(`logMessages.ts`)로 한글 렌더.

**Tech Stack:** Go 1.26, `log/slog`, `gopkg.in/natefinch/lumberjack.v2`, Fiber v2 (`middleware/requestid`), Svelte 4 + TypeScript, vitest.

**Spec:** `docs/superpowers/specs/2026-07-20-logging-improvement-design.md`

## Global Constraints

- 로그 파일 포맷: JSON lines (한 줄 = 한 JSON 객체). 레벨 표기는 slog 기본 `DEBUG`/`INFO`/`WARN`/`ERROR` — `ReplaceAttr` 커스텀 금지.
- `msg`는 안정된 영어 이벤트 키: `"server started"`, `"request"`, `"label generated"`, `"logs cleared"`. 가변 데이터는 전부 slog 필드.
- env: 기존 `LOG_LEVEL`, `LOG_FILE` 유지. 신규 `LOG_MAX_SIZE_MB`(기본 10), `LOG_MAX_BACKUPS`(기본 5).
- API 레벨 필터는 `WARN`/`WARNING` 양쪽 수용. `search` 쿼리 파라미터는 제거(검색은 클라이언트).
- JSON 파싱 실패 줄은 버리지 않고 `{level:"INFO", msg:"<원문>", legacy:true}`로 반환.
- 뷰어/다운로드는 현재 파일만. clear는 `lumberjack.Rotate()` 후 백업 전부 삭제(`os.Truncate` 금지).
- 프론트 미등록 이벤트 키는 원문 fallback. 한글 번역은 프론트 전용(`logMessages.ts`), 백엔드에 한글 로그 금지.
- Go 테스트: `~/.local/go/bin/go test ./...` (Go는 `~/.local/go`, sudo 없음. `make build`로 전체 빌드).
- 커밋 메시지 끝에 `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

---

### Task 1: `internal/logging` slog+lumberjack 재작성

**Files:**
- Modify: `internal/logging/logging.go` (전면 재작성)
- Modify: `internal/logging/logging_test.go` (전면 재작성)
- Modify: `internal/config/config.go` (필드 2개 추가)
- Modify: `go.mod` (lumberjack 추가)

**Interfaces:**
- Consumes: 없음 (최하층)
- Produces:
  - `logging.New(w io.Writer, min slog.Level) *Logger` — 테스트/stdout 전용 로거
  - `logging.Init(logFile, level string, maxSizeMB, maxBackups int) (*Logger, error)` — 파일+stdout 프로세스 로거
  - `logging.Default() *Logger` — Init 안 됐으면 stdout INFO 로거
  - `type Logger struct { *slog.Logger; ... }` — slog 메서드 임베드: `l.Info(msg, k, v, ...)`
  - `(*Logger).Clear() error` — Rotate 후 백업 삭제
  - `(*Logger).Close() error`
  - `config.Config`에 `LogMaxSizeMB int`, `LogMaxBackups int` 필드
- 삭제: `Level` 타입, `LevelOf`, printf 스타일 메서드 전부

- [ ] **Step 1: lumberjack 의존성 추가**

```bash
cd /home/spuhaha18/Project/QR_Web && ~/.local/go/bin/go get gopkg.in/natefinch/lumberjack.v2
```

- [ ] **Step 2: 실패 테스트 작성** — `internal/logging/logging_test.go` 전체 교체:

```go
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
```

- [ ] **Step 3: 실패 확인**

Run: `~/.local/go/bin/go test ./internal/logging/ -v`
Expected: COMPILE FAIL (`New`, `parseLevel`, `Clear` 미정의)

- [ ] **Step 4: `internal/logging/logging.go` 전체 교체**

```go
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
```

- [ ] **Step 5: config 필드 추가** — `internal/config/config.go`의 `Config` 구조체 `LogFile` 필드 아래에 추가:

```go
	LogMaxSizeMB  int // LOG_MAX_SIZE_MB (10) — lumberjack MaxSize
	LogMaxBackups int // LOG_MAX_BACKUPS (5) — lumberjack MaxBackups
```

`Load()`의 `LogFile:` 줄 아래에 추가:

```go
		LogMaxSizeMB:  getEnvInt("LOG_MAX_SIZE_MB", 10),
		LogMaxBackups: getEnvInt("LOG_MAX_BACKUPS", 5),
```

- [ ] **Step 6: 테스트 통과 확인**

Run: `~/.local/go/bin/go test ./internal/logging/ ./internal/config/ -v`
Expected: PASS 전부. (이 시점에 `internal/httpx`와 `cmd/qrweb`는 컴파일 깨짐 — Task 2·3에서 복구, 전체 `./...` 실행은 아직 금지)

- [ ] **Step 7: Commit**

```bash
git add internal/logging/ internal/config/config.go go.mod go.sum
git commit -m "feat(logging): slog JSON lines + lumberjack rotation

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: httpx 요청 추적 + 액세스 로그 + 호출부 전환

**Files:**
- Modify: `internal/httpx/server.go` (requestid 미들웨어, 액세스 로거 교체, 헬퍼 추가)
- Modify: `internal/httpx/label_handler.go:83,125` (slog 필드 스타일 전환)
- Modify: `cmd/qrweb/main.go` (Init 시그니처, 시작 로그)
- Test: `internal/httpx/access_log_test.go` (신규)

**Interfaces:**
- Consumes: Task 1의 `logging.New`, `logging.Init(logFile, level, maxSizeMB, maxBackups)`, `Logger.Info(msg, k, v...)`
- Produces:
  - `requestID(c *fiber.Ctx) string` — httpx 패키지 내부 헬퍼, Fiber requestid Locals에서 추출
  - 액세스 로그 이벤트: `msg="request"` + `method`,`path`,`status`,`duration_ms`,`ip`,`request_id`
  - 비즈니스 이벤트: `msg="label generated"` + `mode`("paste"|"auto"),`file`,`ip`,`request_id`
  - 응답 헤더 `X-Request-ID` (전 요청)

- [ ] **Step 1: 실패 테스트 작성** — `internal/httpx/access_log_test.go` 신규:

```go
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
```

- [ ] **Step 2: 실패 확인**

Run: `~/.local/go/bin/go test ./internal/httpx/ -run TestAccessLog -v`
Expected: COMPILE FAIL 또는 FAIL (구 printf 로거와 비호환, X-Request-ID 없음)

- [ ] **Step 3: `server.go` 수정**

import에 `"strings"`, `"time"`, `"github.com/gofiber/fiber/v2/middleware/requestid"` 추가. `New()`의 미들웨어 등록부를:

```go
	// recover: panic -> {"error":"서버 오류가 발생했습니다."} 500 (handle_errors parity).
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	// Per-request ID: reuses an incoming X-Request-ID, else generates a UUID;
	// echoed on the response header and attached to every log record.
	app.Use(requestid.New())
	// Access log for business routes only — the log viewer's own traffic and
	// static assets would drown the signal.
	app.Use(s.requestLogger())
```

`requestLogger()`를 교체하고 헬퍼 2개 추가:

```go
// requestLogger writes one "request" record per business request. Viewer
// endpoints (/api/logs*) and static/SPA paths are excluded so reading logs
// does not generate logs.
func (s *Server) requestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		path := c.Path()
		if !accessLogged(path) {
			return err
		}
		s.log.Info("request",
			"method", c.Method(),
			"path", path,
			"status", c.Response().StatusCode(),
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", c.IP(),
			"request_id", requestID(c),
		)
		return err
	}
}

// accessLogged reports whether path belongs to the access log: API routes
// minus the log viewer, plus the non-/api label route.
func accessLogged(path string) bool {
	if path == "/create_label" {
		return true
	}
	return strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/api/logs")
}

// requestID returns the Fiber requestid middleware's ID for this request.
func requestID(c *fiber.Ctx) string {
	if v, ok := c.Locals("requestid").(string); ok {
		return v
	}
	return ""
}
```

- [ ] **Step 4: `label_handler.go` 호출부 전환**

83행:

```go
	s.log.Info("label generated", "mode", "paste", "file", filename, "ip", c.IP(), "request_id", requestID(c))
```

125행:

```go
	s.log.Info("label generated", "mode", "auto", "file", filename, "ip", c.IP(), "request_id", requestID(c))
```

- [ ] **Step 5: `cmd/qrweb/main.go` 전환**

```go
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
```

- [ ] **Step 6: 통과 확인** (logs_handler.go는 Task 3에서 고치므로 아직 컴파일 깨짐 — 임시로 확인 범위 제한)

`internal/httpx/logs_handler.go`가 구 로거 API(`s.log.Warn(format, ...)`)를 쓰므로 이 시점엔 컴파일 에러. Step 4와 같은 요령으로 **임시 최소 수정**: `logs_handler.go`의 `s.log.Info("Log file backed up to: %s", backupPath)` → `s.log.Info("log backup", "path", backupPath)`, `s.log.Warn("Failed to backup log file: %v", err)` 2곳 → `s.log.Warn("log backup failed", "err", err.Error())`, `s.log.Info("Log file cleared by user request")` → `s.log.Info("logs cleared")`. (Task 3에서 이 핸들러 전체가 재작성되며 백업 로직 자체가 사라짐.)

Run: `~/.local/go/bin/go test ./... 2>&1 | tail -20` (프로젝트 루트, `~/.local/go/bin/go` 사용)
Expected: 전부 PASS (기존 handler_test 포함)

- [ ] **Step 7: Commit**

```bash
git add internal/httpx/ cmd/qrweb/main.go
git commit -m "feat(httpx): request IDs + structured access log for business routes

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: 뷰어 API 재작성 (`logs_handler.go`)

**Files:**
- Modify: `internal/httpx/logs_handler.go` (전면 재작성)
- Test: `internal/httpx/logs_handler_test.go` (신규)
- Modify: `internal/httpx/handler_test.go:174` 근처의 기존 `/api/logs?lines=-1` 테스트가 새 응답 shape을 기대하도록 확인/조정

**Interfaces:**
- Consumes: Task 1의 `Logger.Clear()`; Task 2의 `requestID(c)`
- Produces: `GET /api/logs` 응답:

```json
{
  "success": true,
  "logs": [
    {"time": "2026-07-20T14:03:22.481+09:00", "level": "INFO", "msg": "label generated",
     "fields": {"mode": "paste", "file": "a.pdf", "ip": "10.0.0.5", "request_id": "..."}},
    {"time": "", "level": "INFO", "msg": "2026-06-01 old text line", "legacy": true, "fields": {}}
  ],
  "total_lines": 2, "requested_lines": 100, "level_filter": "ALL"
}
```

  쿼리: `lines`(기본 100, 최대 1000, 1 미만 → 100), `level`(`all|DEBUG|INFO|WARN|WARNING|ERROR`, WARNING→WARN 정규화), `request_id`(정확 매치). `search` 없음.
- `POST /api/logs/clear` → `Logger.Clear()`, 응답 `{"success":true,"message":"로그 파일이 초기화되었습니다."}` (`backup_file` 필드 제거)
- `GET /api/logs/download` 변경 없음

- [ ] **Step 1: 실패 테스트 작성** — `internal/httpx/logs_handler_test.go` 신규:

```go
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
```

- [ ] **Step 2: 실패 확인**

Run: `~/.local/go/bin/go test ./internal/httpx/ -run 'TestGetLogs|TestClearLogs' -v`
Expected: FAIL (구 구현은 문자열 배열 반환, backup_file 존재)

- [ ] **Step 3: `logs_handler.go` 전면 재작성**

```go
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
```

주의: `newLogsServer` 테스트는 stdout-only 로거(`lj == nil`)로 clear를 안 부르고, clear 테스트는 `logging.Init` 로거 사용 — `Clear()`의 no-op 분기와 실동작 분기 둘 다 커버됨.

- [ ] **Step 4: 통과 확인 + 기존 테스트 회귀**

Run: `~/.local/go/bin/go test ./... 2>&1 | tail -20`
Expected: 전부 PASS. `handler_test.go:174`의 `lines=-1` 테스트가 응답 shape 변경으로 깨지면 새 shape(`logs` 배열이 객체)에 맞게 어서션 수정.

- [ ] **Step 5: Commit**

```bash
git add internal/httpx/
git commit -m "feat(logs-api): serve parsed JSON log entries; clear via rotate

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 4: 프론트 한글 카탈로그 `logMessages.ts`

**Files:**
- Create: `web/frontend/src/lib/logMessages.ts`
- Test: `web/frontend/src/lib/logMessages.test.ts`

**Interfaces:**
- Consumes: Task 3의 응답 entry shape `{time, level, msg, legacy?, fields}`
- Produces:

```ts
export interface LogEntry {
  time: string;
  level: string;
  msg: string;
  legacy?: boolean;
  fields: Record<string, unknown>;
}
export function renderMessage(e: LogEntry): string; // 한글 렌더, 미등록 키는 원문
export function levelLabel(level: string): string;  // "WARN" -> "경고" 등
```

- [ ] **Step 1: 실패 테스트 작성** — `web/frontend/src/lib/logMessages.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { levelLabel, renderMessage, type LogEntry } from './logMessages';

function entry(msg: string, fields: Record<string, unknown> = {}, legacy = false): LogEntry {
  return { time: '2026-07-20T14:00:00+09:00', level: 'INFO', msg, fields, legacy };
}

describe('renderMessage', () => {
  it('renders label generated in Korean', () => {
    const s = renderMessage(entry('label generated', { mode: 'paste', file: 'a.pdf', ip: '10.0.0.5' }));
    expect(s).toBe('라벨 생성: a.pdf (붙여넣기, 10.0.0.5)');
  });

  it('renders auto mode label', () => {
    const s = renderMessage(entry('label generated', { mode: 'auto', file: 'b.pdf', ip: '10.0.0.6' }));
    expect(s).toBe('라벨 생성: b.pdf (자동, 10.0.0.6)');
  });

  it('renders request with duration', () => {
    const s = renderMessage(entry('request', { method: 'POST', path: '/api/create_label', status: 200, duration_ms: 142 }));
    expect(s).toBe('POST /api/create_label → 200 (142ms)');
  });

  it('renders server started and logs cleared', () => {
    expect(renderMessage(entry('server started', { addr: '0.0.0.0:5000' }))).toBe('서버 시작 (0.0.0.0:5000)');
    expect(renderMessage(entry('logs cleared'))).toBe('로그 초기화됨');
  });

  it('falls back to raw msg for unknown keys and legacy lines', () => {
    expect(renderMessage(entry('some new event', { a: 1 }))).toBe('some new event');
    expect(renderMessage(entry('2026-06-01 old text line', {}, true))).toBe('2026-06-01 old text line');
  });
});

describe('levelLabel', () => {
  it('maps slog levels to Korean', () => {
    expect(levelLabel('DEBUG')).toBe('디버그');
    expect(levelLabel('INFO')).toBe('정보');
    expect(levelLabel('WARN')).toBe('경고');
    expect(levelLabel('ERROR')).toBe('오류');
    expect(levelLabel('WEIRD')).toBe('WEIRD');
  });
});
```

- [ ] **Step 2: 실패 확인**

Run: `cd web/frontend && npm test -- logMessages`
Expected: FAIL (모듈 없음)

- [ ] **Step 3: `web/frontend/src/lib/logMessages.ts` 작성**

```ts
// Korean rendering catalog for the log viewer. The backend logs stable English
// event keys (msg) with structured fields; this module is the only place that
// translates them. Unknown keys and legacy (pre-JSON) lines fall back to the
// raw msg, so catalog drift degrades to English instead of breaking.

export interface LogEntry {
  time: string;
  level: string;
  msg: string;
  legacy?: boolean;
  fields: Record<string, unknown>;
}

type Renderer = (f: Record<string, unknown>) => string;

const modeLabels: Record<string, string> = {
  paste: '붙여넣기',
  auto: '자동',
};

const catalog: Record<string, Renderer> = {
  'server started': (f) => `서버 시작 (${f.addr})`,
  'request': (f) => `${f.method} ${f.path} → ${f.status} (${f.duration_ms}ms)`,
  'label generated': (f) => `라벨 생성: ${f.file} (${modeLabels[String(f.mode)] ?? f.mode}, ${f.ip})`,
  'logs cleared': () => '로그 초기화됨',
  'log clear failed': (f) => `로그 초기화 실패: ${f.err}`,
  'server exited': (f) => `서버 종료: ${f.err}`,
};

export function renderMessage(e: LogEntry): string {
  if (e.legacy) return e.msg;
  const render = catalog[e.msg];
  return render ? render(e.fields) : e.msg;
}

const levelLabels: Record<string, string> = {
  DEBUG: '디버그',
  INFO: '정보',
  WARN: '경고',
  ERROR: '오류',
};

export function levelLabel(level: string): string {
  return levelLabels[level] ?? level;
}
```

- [ ] **Step 4: 통과 확인**

Run: `cd web/frontend && npm test -- logMessages`
Expected: PASS 전부

- [ ] **Step 5: Commit**

```bash
git add web/frontend/src/lib/logMessages.ts web/frontend/src/lib/logMessages.test.ts
git commit -m "feat(viewer): Korean log message catalog

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 5: `LogsModal.svelte` 개편

**Files:**
- Modify: `web/frontend/src/components/LogsModal.svelte`

**Interfaces:**
- Consumes: Task 3의 `GET /api/logs` 응답 shape, Task 4의 `renderMessage`/`levelLabel`/`LogEntry`
- Produces: 사용자 UI만 (다른 코드가 의존하지 않음)

- [ ] **Step 1: script 부 개편** — 아래 요소를 반영해 `<script>` 재작성:

```ts
  import { onMount, onDestroy } from 'svelte';
  import { ClipboardList, X, RefreshCw, Download, Trash2 } from 'lucide-svelte';
  import { levelLabel, renderMessage, type LogEntry } from '../lib/logMessages';

  export let open = false;
  export let onClose: () => void;

  interface LogsResponse {
    success: boolean;
    logs: LogEntry[];
    total_lines: number;
    message?: string;
  }

  let entries: LogEntry[] = [];
  let totalLines = 0;
  let levelFilter = 'all';
  let searchQuery = '';
  let requestIdFilter = '';
  let showRaw = false;
  let loading = false;
  let errorMsg = '';

  const levelOptions = [
    { value: 'all', label: '전체' },
    { value: 'DEBUG', label: '디버그' },
    { value: 'INFO', label: '정보' },
    { value: 'WARN', label: '경고' },
    { value: 'ERROR', label: '오류' },
  ];

  async function fetchLogs() {
    loading = true;
    errorMsg = '';
    try {
      const params = new URLSearchParams({ lines: '200' });
      if (levelFilter !== 'all') params.set('level', levelFilter);
      if (requestIdFilter) params.set('request_id', requestIdFilter);
      const res = await fetch(`/api/logs?${params.toString()}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: LogsResponse = await res.json();
      entries = data.logs ?? [];
      totalLines = data.total_lines ?? 0;
      if (entries.length === 0 && data.message) errorMsg = data.message;
    } catch (e) {
      errorMsg = e instanceof Error ? e.message : '로그를 불러올 수 없습니다.';
      entries = [];
    } finally {
      loading = false;
    }
  }

  // Client-side search: matches the rendered Korean text AND raw msg/fields,
  // so both "라벨" and "label" (or a filename) find the same line.
  function matches(e: LogEntry, q: string): boolean {
    const hay = (renderMessage(e) + ' ' + e.msg + ' ' + JSON.stringify(e.fields)).toLowerCase();
    return hay.includes(q.toLowerCase());
  }
  $: visible = searchQuery.trim()
    ? entries.filter((e) => matches(e, searchQuery.trim()))
    : entries;

  function filterByRequestId(id: string) {
    requestIdFilter = id;
    fetchLogs();
  }
  function clearRequestIdFilter() {
    requestIdFilter = '';
    fetchLogs();
  }

  function shortId(e: LogEntry): string {
    const id = e.fields['request_id'];
    return typeof id === 'string' ? id.slice(0, 8) : '';
  }
  function fullId(e: LogEntry): string {
    const id = e.fields['request_id'];
    return typeof id === 'string' ? id : '';
  }

  function fmtTime(iso: string): string {
    if (!iso) return '';
    const d = new Date(iso);
    if (isNaN(d.getTime())) return iso;
    const p = (n: number) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
  }
```

`downloadLogs`/`clearLogs`/`onKey`/모달 open-close reactive 블록/`onMount`/`onDestroy`는 기존 유지하되, open-close 리셋에 `requestIdFilter = ''; showRaw = false;` 추가, `logs = []` → `entries = []`. 검색 디바운스 타이머 로직은 삭제(클라 필터링이라 reactive `visible`로 충분).

- [ ] **Step 2: 마크업 개편** — 로그 줄 렌더부(`logs-lines` 내부)를 entry 기반으로 교체:

```svelte
        <div class="logs-lines" role="log" aria-live="polite">
          {#each visible as e, i}
            <div class="logs-line" class:logs-line-error={e.level === 'ERROR'} class:logs-line-warn={e.level === 'WARN'}>
              <span class="logs-line-num">{i + 1}</span>
              <span class="logs-time">{fmtTime(e.time)}</span>
              <span class="logs-badge logs-badge-{e.level.toLowerCase()}">{levelLabel(e.level)}</span>
              {#if shortId(e)}
                <button type="button" class="logs-reqid" title={fullId(e)} on:click={() => filterByRequestId(fullId(e))}>
                  {shortId(e)}
                </button>
              {/if}
              <span class="logs-line-text">
                {#if showRaw}{JSON.stringify({ time: e.time, level: e.level, msg: e.msg, ...e.fields })}{:else}{renderMessage(e)}{/if}
              </span>
            </div>
          {/each}
        </div>
```

툴바에 추가: 검색 input은 유지(단, `on:input` 디바운스 제거 — bind만으로 reactive 필터), raw 토글 버튼과 request ID 필터 해제 칩:

```svelte
      <button type="button" class="logs-action-btn" class:logs-raw-on={showRaw} on:click={() => (showRaw = !showRaw)}>
        원문
      </button>
      {#if requestIdFilter}
        <button type="button" class="logs-reqid-chip" on:click={clearRequestIdFilter} title="요청 필터 해제">
          요청 {requestIdFilter.slice(0, 8)} ✕
        </button>
      {/if}
```

카운트 표시는 `{visible.length} / {totalLines}줄`.

- [ ] **Step 3: 스타일 추가** — `<style>`에 추가(기존 substring 기반 `logs-line-info` 규칙과 사용 안 하는 클래스 제거):

```css
  .logs-time {
    flex-shrink: 0;
    color: var(--text-muted);
    padding-right: 10px;
    font-size: 0.75rem;
  }

  .logs-badge {
    flex-shrink: 0;
    padding: 1px 7px;
    margin-right: 8px;
    border-radius: 999px;
    font-size: 0.7rem;
    font-weight: 700;
  }
  .logs-badge-debug { background: var(--secondary-bg); color: var(--text-muted); }
  .logs-badge-info  { background: rgba(59, 130, 246, 0.12); color: #2563eb; }
  .logs-badge-warn  { background: rgba(217, 119, 6, 0.12);  color: #d97706; }
  .logs-badge-error { background: var(--error-bg);           color: var(--error-text); }

  .logs-reqid {
    flex-shrink: 0;
    margin-right: 8px;
    padding: 0 4px;
    border: none;
    background: none;
    color: var(--primary-color);
    font-family: inherit;
    font-size: 0.72rem;
    cursor: pointer;
    text-decoration: underline dotted;
  }

  .logs-reqid-chip {
    padding: 4px 10px;
    border-radius: 999px;
    border: 1px solid var(--primary-color);
    background: var(--surface-color);
    color: var(--primary-color);
    font-size: 0.78rem;
    cursor: pointer;
  }

  .logs-raw-on {
    color: var(--primary-color);
    border-color: var(--primary-color);
  }
```

- [ ] **Step 4: 타입/빌드 검증**

Run: `cd web/frontend && npm run check && npm run build`
Expected: svelte-check 0 errors, build 성공

- [ ] **Step 5: Commit**

```bash
git add web/frontend/src/components/LogsModal.svelte
git commit -m "feat(viewer): Korean log rendering, level badges, request-id filter

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 6: 통합 검증 + 문서

**Files:**
- Modify: `CHANGELOG.md` (변경 요약 1entry)
- 검증만: 전체 빌드 + 수동 스모크

**Interfaces:**
- Consumes: 전 Task 결과
- Produces: 릴리스 가능한 상태

- [ ] **Step 1: 전체 테스트 + 빌드**

Run: `~/.local/go/bin/go test ./... && make build`
Expected: 전 테스트 PASS, `bin/qrweb` 생성 (make가 frontend build → embed → go build 체인 수행)

- [ ] **Step 2: 스모크 테스트**

```bash
LOG_FILE=/tmp/claude-1001/-home-spuhaha18-Project-QR-Web/b814bf64-f870-4cd6-8cad-4a92bb4a327a/scratchpad/smoke.log PORT=15000 ./bin/qrweb &
sleep 1
curl -s localhost:15000/api/health >/dev/null
curl -s "localhost:15000/api/logs" | head -c 400; echo
curl -s -X POST localhost:15000/api/logs/clear
kill %1
```

확인 사항: ① `smoke.log`가 JSON lines ② `/api/logs` 응답의 `logs`가 객체 배열이고 `request` entry에 `request_id`/`duration_ms` 존재 ③ `/api/logs` 조회 자체는 로그에 없음 ④ clear 후 파일에 `logs cleared`만 남음.

- [ ] **Step 3: CHANGELOG 갱신**

`CHANGELOG.md` 최상단에 항목 추가 (기존 포맷 따름):

```markdown
- 로그 개선: slog JSON lines + lumberjack 로테이션(10MB×5), request ID 추적, 액세스 로그(비즈니스 라우트만), 뷰어 한글 렌더/레벨 뱃지/요청 필터. `/api/logs`의 `search` 파라미터 제거(클라이언트 검색), clear의 `backup_file` 응답 제거.
```

- [ ] **Step 4: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: changelog for logging improvement

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```
