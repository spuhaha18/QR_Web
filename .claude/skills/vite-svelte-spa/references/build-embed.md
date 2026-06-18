# 빌드 + Go embed 연동

## vite.config.ts
```ts
import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte()],
  base: './',                       // 상대 경로 — embed 루트 서빙
  build: {
    outDir: '../dist',              // → web/dist (Go embed 대상)
    emptyOutDir: true,
  },
  server: {
    proxy: {                        // dev: Go 백엔드로 프록시
      '/create_label': 'http://localhost:5000',
      '/api': 'http://localhost:5000',
    },
  },
});
```
디렉토리: `web/frontend/`(소스) → 빌드 → `web/dist/`(출력).

## Go embed (web/embed.go)
```go
package web

import (
    "embed"
    "io/fs"
)

//go:embed all:dist
var distFS embed.FS

func DistFS() fs.FS {
    sub, _ := fs.Sub(distFS, "dist")
    return sub
}
```

## Fiber 정적 서빙 (cmd/qrweb 또는 httpx)
```go
import "github.com/gofiber/fiber/v2/middleware/filesystem"

app.Use("/", filesystem.New(filesystem.Config{
    Root:         http.FS(web.DistFS()),
    Index:        "index.html",
    NotFoundFile: "index.html",  // SPA 라우팅 폴백
}))
```
> API 라우트를 정적 미들웨어 **앞에** 등록해 `/api/*`, `/create_label`이 가로채지지 않게 한다.

## 빌드 체인
```
cd web/frontend && npm ci && npm run build   # → web/dist
go build -trimpath -ldflags="-s -w" -o bin/qrweb ./cmd/qrweb
```
`CGO_ENABLED=0` — go-qrcode/excelize 순수 Go → 완전 정적 바이너리. 크로스컴파일 `GOOS/GOARCH`.

## Makefile
```
make frontend   # npm ci + build → web/dist
make build      # frontend + go build
make dev        # go run + vite dev (프록시) 동시
make test       # go test ./...
make clean
```
