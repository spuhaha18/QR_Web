#!/usr/bin/env bash
#
# QR_Web 배포/업그레이드 스크립트 — Oracle Cloud A1 (arm64) / Ubuntu 24.04
#
# 운영 환경 사실(2026-06 배포로 확인):
#   - 앱은 호스트 systemd 서비스(0.0.0.0:PORT)로 상주.
#   - Nginx Proxy Manager 는 Docker 컨테이너(jc21/nginx-proxy-manager), compose
#     네트워크 `qr_web_default`(gateway 172.20.0.1)에 붙어 있음. 도메인·SSL·외부
#     방화벽(Oracle Security List 443)은 이미 구성됨.
#   - NPM 컨테이너 → 호스트 바이너리(:PORT) 도달은 호스트 iptables 가 기본
#     REJECT(icmp-host-prohibited) 하므로 "NPM 네트워크 서브넷 → :PORT ACCEPT"
#     규칙 1줄이 반드시 필요. (구 v2.1.1은 같은 네트워크의 컨테이너였어서 불필요했음.)
#   - PORT(5000)는 외부로 안 나가므로 Oracle Security List 추가는 불필요.
#
# 이 스크립트가 하는 일: arm64 바이너리 설치 · systemd 등록 · (도커 NPM 감지 시)
#   NPM 서브넷→PORT iptables 규칙 자동 추가 · 헬스체크 · NPM forward 대상 안내.
#   NPM Proxy Host 의 Forward Hostname/Port 설정만 웹 UI에서 수동(스크립트 영역 밖).
#
# 사용:
#   sudo ./deploy-oracle-a1.sh
#   sudo VERSION=v3.0.0 PORT=5000 INSTALL_DIR=/opt/qrweb ./deploy-oracle-a1.sh
#
set -euo pipefail

# ── 설정 (환경변수로 override) ──────────────────────────────────
VERSION="${VERSION:-v3.0.0}"
INSTALL_DIR="${INSTALL_DIR:-/opt/qrweb}"   # 디렉터리는 직접 조정
PORT="${PORT:-5000}"                       # NPM Forward Port 와 일치
SVC_USER="${SVC_USER:-qrweb}"
LOG_LEVEL="${LOG_LEVEL:-INFO}"
NPM_CONTAINER="${NPM_CONTAINER:-nginx-proxy-manager}"  # 도커 NPM 컨테이너명
NPM_GATEWAY="${NPM_GATEWAY:-}"             # 비우면 자동 탐지
NPM_SUBNET="${NPM_SUBNET:-}"               # 비우면 자동 탐지
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

# ── 3. 포트 점유 가드 (구버전/타서비스 충돌 방지) ───────────────
if ss -ltnH "sport = :${PORT}" 2>/dev/null | grep -q .; then
  HOLDER="$(ss -ltnpH "sport = :${PORT}" 2>/dev/null | sed -E 's/.*users:\(\("([^"]+)".*/\1/' | head -1)"
  die "포트 ${PORT} 를 다른 프로세스(${HOLDER:-unknown})가 점유 중. 구버전 중지 또는 새 PORT 사용."
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

# ── 5. NPM(도커) → 호스트:PORT iptables 규칙 (감지 시 자동) ──────
# NPM 컨테이너가 있으면 그 도커 네트워크의 gateway/subnet 을 알아내
# "subnet → :PORT ACCEPT" 를 REJECT 위(최상단)에 멱등 삽입한다.
add_rule() {  # 없을 때만 INPUT 최상단에 삽입
  if ! iptables -C INPUT "$@" 2>/dev/null; then
    iptables -I INPUT "$@"; log "iptables 추가: $*"
  else
    log "iptables 이미 존재: $*"
  fi
}

if command -v docker >/dev/null 2>&1 && docker inspect "$NPM_CONTAINER" >/dev/null 2>&1; then
  NET="$(docker inspect -f '{{range $k,$_ := .NetworkSettings.Networks}}{{$k}} {{end}}' "$NPM_CONTAINER" | awk '{print $1}')"
  [ -z "$NPM_GATEWAY" ] && NPM_GATEWAY="$(docker inspect -f "{{(index .NetworkSettings.Networks \"$NET\").Gateway}}" "$NPM_CONTAINER" 2>/dev/null || true)"
  [ -z "$NPM_SUBNET" ]  && NPM_SUBNET="$(docker network inspect -f '{{range .IPAM.Config}}{{.Subnet}} {{end}}' "$NET" 2>/dev/null | awk '{print $1}' || true)"
  log "NPM 네트워크: ${NET}  gateway=${NPM_GATEWAY:-?}  subnet=${NPM_SUBNET:-?}"

  if [ -n "$NPM_SUBNET" ]; then
    add_rule -s "$NPM_SUBNET" -p tcp --dport "$PORT" -j ACCEPT
    if command -v netfilter-persistent >/dev/null 2>&1; then
      netfilter-persistent save && log "iptables 영속화"
    else
      warn "netfilter-persistent 없음 → 'apt install iptables-persistent' 권장(규칙 영속화 필요)."
    fi
  else
    warn "NPM 서브넷 자동탐지 실패 → NPM_SUBNET=... 로 직접 지정 후 재실행."
  fi
else
  warn "도커 NPM(${NPM_CONTAINER}) 미감지 → iptables 규칙 건너뜀(필요시 수동)."
fi

# ── 6. 헬스 체크 (호스트 로컬) ──────────────────────────────────
log "기동 확인..."
for i in $(seq 1 10); do
  if OUT="$(curl -fsS "http://127.0.0.1:${PORT}/api/health" 2>/dev/null)"; then
    log "정상: ${OUT}"; break
  fi
  [ "$i" -eq 10 ] && { systemctl --no-pager status qrweb | tail -20; die "헬스체크 실패 — 위 status 확인."; }
  sleep 1
done

# ── 7. NPM 도달 검증 + 안내 ─────────────────────────────────────
FWD="${NPM_GATEWAY:-<NPM 게이트웨이 IP>}"
if [ -n "${NPM_GATEWAY:-}" ] && docker inspect "$NPM_CONTAINER" >/dev/null 2>&1; then
  if docker exec "$NPM_CONTAINER" python3 -c "import socket;s=socket.socket();s.settimeout(2);s.connect(('${NPM_GATEWAY}',${PORT}))" >/dev/null 2>&1; then
    log "NPM→호스트 도달 OK: ${NPM_GATEWAY}:${PORT}"
  else
    warn "NPM→호스트(${NPM_GATEWAY}:${PORT}) 도달 실패 — iptables 서브넷/영속화 확인."
  fi
fi

cat <<EOF

────────────────────────────────────────────────────────────
 v${VERSION#v} 기동 완료 — 호스트 http://127.0.0.1:${PORT}

 ▼ NPM 웹 UI (Proxy Host) — 보통 한 번만 설정
   - Forward Hostname/IP : ${FWD}
   - Forward Port        : ${PORT}
   - Scheme              : http
   확인: curl -I https://label.inno-n.duckdns.org/api/health  → version ${VERSION#v}

 ※ Oracle Security List 에 ${PORT} 추가 불필요(내부 트래픽).
 관리: systemctl status qrweb · journalctl -u qrweb -f
 롤백: NPM Forward Port 되돌리기 / systemctl stop qrweb && mv ${BIN}.bak ${BIN} && systemctl start qrweb
────────────────────────────────────────────────────────────
EOF
