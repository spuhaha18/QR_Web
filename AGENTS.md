# AGENTS.md

## Project

QR_Web is a Go (Fiber) + Svelte SPA application that generates binder-label
PDFs with embedded QR codes, shipped as a single static binary. The Python
Flask original was fully rewritten and removed; see git history if you need
the legacy implementation.

## Layout

- `cmd/qrweb/` — binary entry point.
- `internal/label/` — domain layer: doc types, binder sizes, label schema,
  validation, QR image intake. Korean-facing error strings live here.
- `internal/pdf/` — PDF rendering: geometry (Excel-unit→mm conversion,
  calibrated to the office's measured prints), mixed Korean/Latin text engine
  with auto-shrink-to-fit, label piece renderer, A4 shelf packer, generator.
  Fonts are embedded via `go:embed` (`internal/pdf/fonts/`).
- `internal/qr/` — QR payload building (CP949) and PNG generation.
- `internal/httpx/` — Fiber server and handlers (`/create_label` paste mode,
  `/api/create_label` auto mode, logs, health).
- `web/frontend/` — Vite+Svelte SPA; built output embeds into the binary via
  `web/embed.go`.
- `doc/spec/`, `doc/plan/` — design specs and implementation plans.

## Commands

Go toolchain lives at `~/.local/go/bin/go` (not on PATH by default).

- `make build GO=$HOME/.local/go/bin/go` — frontend build + Go binary to `bin/qrweb`.
- `~/.local/go/bin/go test ./internal/...` — full test suite.
- `PORT=5000 ./bin/qrweb` — run locally.

Deployment: `scripts/deploy-oracle-a1.sh` (Oracle A1 arm64, systemd + Nginx
Proxy Manager; the app itself is not containerized).

## Conventions

- Commit subjects use conventional prefixes (`feat:`, `fix:`, `chore:`, `docs:`).
- Korean user-facing messages are exact contract strings — do not reword.
- Label physical sizes are locked by snapshot tests in
  `internal/pdf/geometry_test.go`; the calibration anchors (47×150mm for the
  7cm label) come from real print measurements — change only with new
  measurements.
- Never truncate label text: the text engine shrinks the font to fit.
