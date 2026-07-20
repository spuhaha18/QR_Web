# syntax=docker/dockerfile:1
# 3-stage build: SPA -> static Go binary -> scratch runtime.
# Runtime needs no distro: all assets (SPA, fonts) are go:embed'ed and
# CGO_ENABLED=0 links fully static.

# ---- frontend: Vite/Svelte SPA -> web/dist ----
FROM node:24-alpine AS frontend
WORKDIR /src/web/frontend
COPY web/frontend/package.json web/frontend/package-lock.json ./
RUN npm ci
COPY web/frontend/ ./
RUN npm run build

# ---- backend: static Go binary with embedded SPA ----
FROM golang:1.26-alpine AS build
RUN apk add --no-cache tzdata
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X qrweb/internal/config.defaultVersion=$(cat VERSION)" \
    -o /qrweb ./cmd/qrweb

# ---- runtime ----
FROM scratch
# Log timestamps are rendered in local time; ship KST zoneinfo (scratch has none).
COPY --from=build /usr/share/zoneinfo/Asia/Seoul /usr/share/zoneinfo/Asia/Seoul
COPY --from=build /qrweb /qrweb
ENV TZ=Asia/Seoul LOG_FILE=/data/app.log
VOLUME /data
EXPOSE 5000
ENTRYPOINT ["/qrweb"]
