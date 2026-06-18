# QR Web — Go/Fiber backend + embedded Vite/Svelte SPA.
# `make build` produces a single static binary (bin/qrweb) with the SPA embedded.

BIN        := bin/qrweb
PKG        := ./cmd/qrweb
FRONTEND   := web/frontend
DIST       := web/dist

# Inject the VERSION file (if present) into the /api/health version string.
VERSION    := $(shell cat VERSION 2>/dev/null)
ifeq ($(strip $(VERSION)),)
LDFLAGS    := -s -w
else
LDFLAGS    := -s -w -X qrweb/internal/config.defaultVersion=$(VERSION)
endif

GO         := go
export CGO_ENABLED := 0

.PHONY: all build frontend run dev test clean

all: build

## frontend: install deps and build the SPA into web/dist (Go embed target).
frontend:
	cd $(FRONTEND) && npm ci && npm run build

## build: build the frontend then compile the static binary with embedded SPA.
build: frontend
	$(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(BIN) $(PKG)

## run: build (incl. frontend) then run the binary.
run: build
	./$(BIN)

## dev: run the Go backend (:5000) and the Vite dev server (proxy) together.
## Ctrl-C stops both.
dev:
	@echo "Starting Go backend (:5000) and Vite dev server (:5173)..."
	@trap 'kill 0' INT TERM EXIT; \
	$(GO) run $(PKG) & \
	(cd $(FRONTEND) && npm run dev) & \
	wait

## test: run all Go tests.
test:
	$(GO) test ./...

## clean: remove build artifacts (binary + dist).
clean:
	rm -rf $(BIN) $(DIST)
