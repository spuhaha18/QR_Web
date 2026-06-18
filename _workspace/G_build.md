# G — Build & Embed (Phase 6 일부)

go-backend-engineer 산출물. Vite/Svelte SPA를 Go 바이너리에 임베드 → 단일 정적 실행파일.

## 산출물
- `web/embed.go` (`package web`) — `//go:embed all:dist` + `DistFS() fs.FS`.
- `internal/httpx/server.go` 수정 — Fiber `filesystem` 미들웨어로 SPA 서빙, `/` `/logs` 플레이스홀더 제거.
- `cmd/qrweb/main.go` — 헤더 주석만 갱신(임베드 반영).
- `internal/config/config.go` — `defaultVersion` 패키지 var 추가(ldflags 주입 대상).
- `Makefile` — frontend/build/run/dev/test/clean.

## embed 연동 방식
- `web/embed.go`: `//go:embed all:dist` → `var distFS embed.FS`. `DistFS()`는 `fs.Sub(distFS,"dist")`로 dist 루트 반환(경로가 `index.html`, `assets/...`).
  - `all:` 접두로 `.`/`_` 시작 파일도 포함(미래 안전). dist 존재 상태에서만 컴파일 성공.
- `internal/httpx/server.go` `registerRoutes()`:
  - **API/명시 라우트를 전부 먼저 등록**: `POST /create_label`, `POST /api/create_label`, `GET /api/qr_image/:text`, `POST /api/qr_image_base64`, `GET /api/health`, `GET /api/logs`, `POST /api/logs/clear`, `GET /api/logs/download`.
  - **마지막에** `app.Use("/", filesystem.New(...))`:
    - `Root: http.FS(web.DistFS())`, `Index:"index.html"`, `NotFoundFile:"index.html"`(SPA 폴백).
  - Fiber는 등록 순서대로 매칭 → API 라우트가 정적 catch-all보다 우선. `/api/*`·`/create_label`이 SPA로 새지 않음(스모크로 검증).

## 버전 주입
- `VERSION` 파일(`2.1.1.0`) → Makefile이 `-ldflags "-X qrweb/internal/config.defaultVersion=$(VERSION)"`로 주입.
- `config.Load()`의 `Version` 기본값이 `defaultVersion`. `APP_VERSION` env는 여전히 런타임 오버라이드.
- VERSION 없으면 ldflags에서 `-X` 생략 → `1.0.0`(config.py 패리티) 유지.
- `go build ./...`(Makefile 미경유) 시엔 주입 없이 `1.0.0`.

## Makefile 타겟
- `make frontend` — `cd web/frontend && npm ci && npm run build` → `web/dist`.
- `make build` — frontend + `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w [-X ...version]" -o bin/qrweb ./cmd/qrweb`.
- `make run` — build 후 `./bin/qrweb`.
- `make dev` — `go run ./cmd/qrweb`(:5000) + `npm run dev`(:5173 vite, 프록시) 동시, Ctrl-C로 둘 다 종료(`trap 'kill 0'`).
- `make test` — `go test ./...`.
- `make clean` — `bin/qrweb` + `web/dist` 제거.
- `export CGO_ENABLED := 0`(Makefile 전역) → 순수 Go(go-qrcode/excelize) 완전 정적.

## 바이너리
- `bin/qrweb`: ELF 64-bit, **statically linked, stripped**. 크기 **14M**.
- `file bin/qrweb` → `statically linked ... stripped` 확인.

## 검증 결과
- `go build ./...` (dist 존재) → 통과(embed 성공).
- `make build` → 성공, `bin/qrweb` 생성. svelte-check 0/0, vite build OK.
- **단일 바이너리 스모크** (`PORT=5098 ./bin/qrweb`):
  | 요청 | 결과 |
  |---|---|
  | `GET /` | 200, `text/html` (SPA index.html, `<!doctype html>`) |
  | `GET /api/health` | 200, `{"status":"healthy","timestamp":"...","version":"2.1.1.0"}` |
  | `HEAD /assets/index-BGrzfmWG.js` | 200, `text/javascript`, Content-Length 120880 |
  | `GET /somespapath` | 200, `text/html` (index.html 폴백, `<div id="app">` 포함) |
  | `GET /logs` | 200, `text/html` (SPA 폴백) |
  | `POST /api/qr_image_base64` `{"text":""}` | **400** `{"error":"QR 코드 텍스트가 제공되지 않았습니다."}` (API 우선, SPA로 안 샘) |
- **깨끗한 빌드 재현성**: `rm -rf web/dist && make frontend && go build ./...` → 통과(Makefile이 dist 재생성).
- `go test ./...` → 전부 green (excel/httpx/imaging/label/qr).

## 막힌 점
- 없음.

## 비고
- `web/dist`는 gitignore 대상(`.gitignore`에 `web/dist/`). `//go:embed`는 빌드 시점에 dist가 있어야 하므로 `make build`가 frontend를 선행하도록 구성. CI/클린 체크아웃에서도 `make build` 한 번이면 충족.
- API 계약(E)/프론트(F) 변경 없음. 라우트 등록 순서만 추가.
