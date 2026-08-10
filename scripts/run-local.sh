#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="${BOOTSTRAP_ENV_FILE:-$ROOT_DIR/.env}"
TUNNEL_ENV_FILE="${BOOTSTRAP_TUNNEL_ENV_FILE:-$ROOT_DIR/.env.tunnel}"
LOG_DIR="${BOOTSTRAP_LOG_DIR:-$ROOT_DIR/data/logs}"
GATEWAY_PID=""
TUNNEL_PID=""

info() { printf '  • %s\n' "$*"; }
ok() { printf '  ✓ %s\n' "$*"; }
warn() { printf '  ! %s\n' "$*" >&2; }
die() { printf '  ✗ %s\n' "$*" >&2; exit 1; }

cleanup() {
  local pid
  for pid in "$TUNNEL_PID" "$GATEWAY_PID"; do
    if [[ -n "$pid" ]] && kill -0 "$pid" >/dev/null 2>&1; then
      kill "$pid" >/dev/null 2>&1 || true
    fi
  done
  for pid in "$TUNNEL_PID" "$GATEWAY_PID"; do
    if [[ -n "$pid" ]]; then
      wait "$pid" >/dev/null 2>&1 || true
    fi
  done
}
trap cleanup EXIT INT TERM

[[ -f "$ENV_FILE" ]] || die "$ENV_FILE is missing; run ./bootstrap"
[[ -f "$TUNNEL_ENV_FILE" ]] || die "$TUNNEL_ENV_FILE is missing; run ./bootstrap"
command -v curl >/dev/null 2>&1 || die "curl is required"
command -v tunnel-client >/dev/null 2>&1 || die "tunnel-client is required"

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
# shellcheck disable=SC1090
. "$TUNNEL_ENV_FILE"
set +a

: "${CONTROL_PLANE_API_KEY:?CONTROL_PLANE_API_KEY is missing from $TUNNEL_ENV_FILE}"
PROFILE="${TUNNEL_CLIENT_PROFILE:-tailscale-ssh}"
MCP_SERVER_URL="${MCP_SERVER_URL:-http://127.0.0.1:8080/mcp}"
TUNNEL_HEALTH_ADDR="${TUNNEL_CLIENT_HEALTH_ADDR:-127.0.0.1:8081}"
GATEWAY_BASE="${MCP_SERVER_URL%/mcp}"
TUNNEL_BASE="http://$TUNNEL_HEALTH_ADDR"

operator_requested=false
case "${OPERATOR_MODE:-false}" in
  1|t|T|true|TRUE|True) operator_requested=true ;;
esac

mkdir -p "$LOG_DIR"
chmod 700 "$ROOT_DIR/data" "$LOG_DIR" 2>/dev/null || true

wait_for_url() {
  local url="$1" seconds="$2" label="$3" i
  i=0
  while ((i < seconds)); do
    if curl -fsS --max-time 2 "$url" >/dev/null 2>&1; then
      ok "$label is ready"
      return 0
    fi
    sleep 1
    i=$((i + 1))
  done
  return 1
}

prepare_gateway_binary() {
  if command -v go >/dev/null 2>&1; then
    command -v make >/dev/null 2>&1 || die "make is required to build the gateway from source"
    info "building gateway from source"
    make -C "$ROOT_DIR" build >/dev/null
    return 0
  fi
  if [[ -x "$ROOT_DIR/bin/gateway" ]]; then
    info "using prebuilt gateway binary: $ROOT_DIR/bin/gateway"
    return 0
  fi
  die "Go is not installed and no executable prebuilt bin/gateway is present"
}

printf 'Starting TetherBound Community / ChatGPT Tailscale MCP gateway\n'

if curl -fsS --max-time 2 "$GATEWAY_BASE/healthz" >/dev/null 2>&1; then
  if $operator_requested; then
    die "a gateway is already running at $GATEWAY_BASE; stop it before starting a fresh Operator-mode lease"
  fi
  info "gateway is already running at $GATEWAY_BASE"
  wait_for_url "$GATEWAY_BASE/readyz" 15 "gateway" || die "existing gateway is healthy but not ready"
else
  prepare_gateway_binary
  if $operator_requested; then
    info "starting gateway in Operator mode; log: $LOG_DIR/gateway.log"
  else
    info "starting gateway; log: $LOG_DIR/gateway.log"
  fi
  (
    cd "$ROOT_DIR"
    exec ./bin/gateway
  ) >"$LOG_DIR/gateway.log" 2>&1 &
  GATEWAY_PID=$!
  if ! wait_for_url "$GATEWAY_BASE/readyz" 100 "gateway"; then
    tail -n 50 "$LOG_DIR/gateway.log" >&2 || true
    die "gateway failed to become ready"
  fi
fi

if curl -fsS --max-time 2 "$TUNNEL_BASE/healthz" >/dev/null 2>&1; then
  info "tunnel-client is already running at $TUNNEL_BASE"
  wait_for_url "$TUNNEL_BASE/readyz" 20 "tunnel-client" || die "existing tunnel-client is healthy but not ready"
else
  info "starting tunnel-client profile $PROFILE; log: $LOG_DIR/tunnel-client.log"
  tunnel-client run --profile "$PROFILE" >"$LOG_DIR/tunnel-client.log" 2>&1 &
  TUNNEL_PID=$!
  if ! wait_for_url "$TUNNEL_BASE/readyz" 60 "tunnel-client"; then
    tail -n 80 "$LOG_DIR/tunnel-client.log" >&2 || true
    die "tunnel-client failed to become ready"
  fi
fi

printf '\nReady\n'
ok "MCP gateway: $MCP_SERVER_URL"
ok "Tunnel status: $TUNNEL_BASE/ui"
if $operator_requested; then
  ok "Operator mode: active for the locally configured lease; arbitrary shell runs as ${SSH_USER:-chatgpt} without TetherBound-added elevation"
fi
info "In ChatGPT, open your custom MCP app and rescan/refresh its tools if needed."
info "Press Ctrl-C to stop only the processes started by this command."

if [[ -z "$GATEWAY_PID" && -z "$TUNNEL_PID" ]]; then
  info "both processes were already running; leaving them untouched"
  trap - EXIT INT TERM
  exit 0
fi

while :; do
  if [[ -n "$GATEWAY_PID" ]] && ! kill -0 "$GATEWAY_PID" >/dev/null 2>&1; then
    wait "$GATEWAY_PID" || true
    tail -n 50 "$LOG_DIR/gateway.log" >&2 || true
    die "gateway exited"
  fi
  if [[ -n "$TUNNEL_PID" ]] && ! kill -0 "$TUNNEL_PID" >/dev/null 2>&1; then
    wait "$TUNNEL_PID" || true
    tail -n 80 "$LOG_DIR/tunnel-client.log" >&2 || true
    die "tunnel-client failed"
  fi
  sleep 1
done
