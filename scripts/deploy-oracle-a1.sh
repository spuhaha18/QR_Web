#!/usr/bin/env bash
#
# QR_Web 배포/업그레이드 스크립트 — Oracle Cloud A1 (arm64) / Ubuntu 24.04
#
# 전제: 도메인(label.inno-n.duckdns.org)·Nginx Proxy Manager·iptables는 이미
#       구성/동작 중 (v2.1.1 운영 중). 이 스크립트는 Go 바이너리(v3.0.0)를
#       설치하고 systemd로 상주시키는 것까지만 담당한다.
#       NPM/방화벽/DuckDNS 는 건드리지 않는다.
#
# 사용:
#   sudo PORT=<NPM이 forward하는 포트> ./deploy-oracle-a1.sh
#   sudo VERSION=v3.0.0 INSTALL_DIR=/opt/qrweb PORT=8080 ./deploy-oracle-a1.sh
#
set -euo pipefail

# ── 설정 (환경변수로 override) ──────────────────────────────────
VERSION="${VERSION:-v3.0.0}"
INSTALL_DIR="${INSTALL_DIR:-/opt/qrweb}"   # 디렉터리는 직접 조정
PORT="${PORT:-8080}"                       # NPM Proxy Host가 forward하는 포트와 일치시킬 것
SVC_USER="${SVC_USER:-qrweb}"
LOG_LEVEL="${LOG_LEVEL:-INFO}"
ASSET="qrweb-linux-arm64"
REPO="spuhaha18/QR_Web"

URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"
BIN="${INSTALL_DIR}/qrweb"
UNIT="/etc/systemd/system/qrweb.service"

log()  { printf '\033[1;32m[+]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[!]\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31m[x]\033[0m %s\n' "$*" >&2; exit 1; }

# ── 사전 점검 ────────────────────────────────────────────────────
[ "$(id -u)" -eq 0 ] || die "root로 실행하세요 (sudo)."
ARCH="$(uname -m)"
[ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ] || die "arm64(A1) 전용. 현재: ${ARCH}."
log "아키텍처: ${ARCH} (arm64 OK)"

# ── 1. 사용자/디렉터리 ──────────────────────────────────────────
if ! id -u "$SVC_USER" >/dev/null 2>&1; then
  log "시스템 계정 생성: ${SVC_USER}"
  useradd --system --no-create-home --shell /usr/sbin/nologin "$SVC_USER"
fi
install -d -o "$SVC_USER" -g "$SVC_USER" "$INSTALL_DIR" "${INSTALL_DIR}/logs"

# ── 2. 바이너리 다운로드/배치 (멱등, 롤백 백업) ──────────────────
log "다운로드: ${URL}"
TMP="$(mktemp)"
curl -fL --retry 3 -o "$TMP" "$URL" || die "다운로드 실패 — VERSION(${VERSION})/네트워크 확인."
file "$TMP" | grep -q "ARM aarch64" || die "arm64 ELF 아님: $(file -b "$TMP")"

if systemctl is-active --quiet qrweb 2>/dev/null; then
  log "기존 qrweb 서비스 중지(교체)"; systemctl stop qrweb
fi
[ -f "$BIN" ] && cp -a "$BIN" "${BIN}.bak" && log "롤백 백업: ${BIN}.bak"
install -o "$SVC_USER" -g "$SVC_USER" -m 0755 "$TMP" "$BIN"; rm -f "$TMP"
log "바이너리 설치: ${BIN}"

# ── 3. 포트 점유 가드 (구 v2.1.1 충돌 방지) ─────────────────────
# qrweb 정지 상태에서 PORT가 여전히 LISTEN이면 다른 프로세스(예: v2.1.1)가 점유 중.
if ss -ltnH "sport = :${PORT}" 2>/dev/null | grep -q .; then
  HOLDER="$(ss -ltnpH "sport = :${PORT}" 2>/dev/null | sed -E 's/.*users:\(\("([^"]+)".*/\1/' | head -1)"
  die "포트 ${PORT} 를 다른 프로세스(${HOLDER:-unknown})가 점유 중. \
구 v2.1.1을 먼저 중지하거나, 새 PORT로 띄운 뒤 NPM Forward Port를 그 포트로 바꾸세요."
fi

# ── 4. systemd 유닛 ─────────────────────────────────────────────
log "systemd 유닛: ${UNIT}"
cat > "$UNIT" <<EOF
[Unit]
Description=QR_Web label generator
After=network.target

[Service]
User=${SVC_USER}
Group=${SVC_USER}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${BIN}
Environment=HOST=0.0.0.0
Environment=PORT=${PORT}
Environment=LOG_LEVEL=${LOG_LEVEL}
Environment=LOG_FILE=${INSTALL_DIR}/logs/app.log
Restart=always
RestartSec=3
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now qrweb

# ── 5. 헬스 체크 ────────────────────────────────────────────────
log "기동 확인..."
for i in $(seq 1 10); do
  if OUT="$(curl -fsS "http://127.0.0.1:${PORT}/api/health" 2>/dev/null)"; then
    log "정상: ${OUT}"; break
  fi
  [ "$i" -eq 10 ] && { systemctl --no-pager status qrweb | tail -20; die "헬스체크 실패 — 위 status 확인."; }
  sleep 1
done

cat <<EOF

────────────────────────────────────────────────────────────
 v3.0.0 기동 완료 — http://127.0.0.1:${PORT} (호스트 로컬)

 ▼ NPM 확인 (이미 운영 중이므로 보통 무변경)
   - 기존 Proxy Host의 Forward Port 가 ${PORT} 인지 확인.
     다르면(구 v2.1.1과 포트가 바뀐 경우) NPM에서 Forward Port만 ${PORT} 로 수정.
   - 확인: curl -I https://label.inno-n.duckdns.org/api/health  → version 3.0.0

 관리: systemctl status qrweb · journalctl -u qrweb -f
 롤백(이전 바이너리로): systemctl stop qrweb && mv ${BIN}.bak ${BIN} && systemctl start qrweb
        (구 v2.1.1로 되돌리려면 v2.1.1 서비스를 다시 기동)
────────────────────────────────────────────────────────────
EOF
