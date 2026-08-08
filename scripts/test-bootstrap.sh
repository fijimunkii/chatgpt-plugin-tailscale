#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/chatgpt-ts-bootstrap-test.XXXXXX")"
trap 'rm -rf "$TMP_ROOT"' EXIT

FIXTURE="$TMP_ROOT/repo"
FAKEBIN="$TMP_ROOT/bin"
HOME_DIR="$TMP_ROOT/home"
XDG_DIR="$TMP_ROOT/xdg"
LOG_FILE="$TMP_ROOT/tunnel-client.log"
mkdir -p "$FIXTURE/config" "$FAKEBIN" "$HOME_DIR" "$XDG_DIR"
cp "$ROOT_DIR/config/example.env" "$FIXTURE/config/example.env"

cat > "$FAKEBIN/go" <<'SH'
#!/usr/bin/env bash
if [[ "${1:-}" == "version" ]]; then
  echo 'go version go1.26.5 test/amd64'
  exit 0
fi
exit 0
SH

cat > "$FAKEBIN/curl" <<'SH'
#!/usr/bin/env bash
exit 7
SH

cat > "$FAKEBIN/tunnel-client" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "--version" ]]; then
  echo 'tunnel-client v0.test'
  exit 0
fi
if [[ "${1:-}" == "init" ]]; then
  profile=''
  tunnel_id=''
  mcp_url=''
  health=''
  shift
  while (($#)); do
    case "$1" in
      --profile) profile="$2"; shift 2 ;;
      --tunnel-id) tunnel_id="$2"; shift 2 ;;
      --mcp-server-url) mcp_url="$2"; shift 2 ;;
      --health-listen-addr) health="$2"; shift 2 ;;
      --sample) shift 2 ;;
      *) shift ;;
    esac
  done
  mkdir -p "${XDG_CONFIG_HOME}/tunnel-client"
  cat > "${XDG_CONFIG_HOME}/tunnel-client/${profile}.yaml" <<YAML
config_version: 1
control_plane:
  tunnel_id: "${tunnel_id}"
health:
  listen_addr: "${health}"
mcp:
  server_urls:
    - channel: main
      url: "${mcp_url}"
YAML
  printf 'init %s\n' "$profile" >> "${FAKE_TUNNEL_LOG}"
  exit 0
fi
exit 0
SH
chmod +x "$FAKEBIN/go" "$FAKEBIN/curl" "$FAKEBIN/tunnel-client"

run_bootstrap() {
  PATH="$FAKEBIN:$PATH" \
  HOME="$HOME_DIR" \
  XDG_CONFIG_HOME="$XDG_DIR" \
  FAKE_TUNNEL_LOG="$LOG_FILE" \
  BOOTSTRAP_ROOT="$FIXTURE" \
  TS_AUTHKEY='tskey-auth-test' \
  CONTROL_PLANE_TUNNEL_ID='tunnel_0123456789abcdef0123456789abcdef' \
  CONTROL_PLANE_API_KEY='sk-test-runtime-key' \
    "$ROOT_DIR/scripts/bootstrap.sh" \
      --non-interactive \
      --skip-policy \
      --skip-target \
      --no-open
}

run_bootstrap
printf 'CUSTOM_KEEP=1\n' >> "$FIXTURE/.env"
awk 'BEGIN { done = 0 } /^SSH_ALLOWED_SERVICES=/ { if (!done) print "SSH_ALLOWED_SERVICES="; done = 1; next } { print }' "$FIXTURE/.env" > "$FIXTURE/.env.next"
mv "$FIXTURE/.env.next" "$FIXTURE/.env"
chmod 600 "$FIXTURE/.env"
run_bootstrap

[[ -f "$FIXTURE/.env" ]]
[[ -f "$FIXTURE/.env.tunnel" ]]
[[ -f "$XDG_DIR/tunnel-client/tailscale-ssh.yaml" ]]
[[ "$(grep -c '^TS_AUTHKEY=' "$FIXTURE/.env")" -eq 1 ]]
[[ "$(grep -c '^CONTROL_PLANE_API_KEY=' "$FIXTURE/.env.tunnel")" -eq 1 ]]
grep -Fxq 'CUSTOM_KEEP=1' "$FIXTURE/.env"
grep -Fxq 'SSH_ALLOWED_SERVICES=' "$FIXTURE/.env"
[[ "$(grep -c '^init tailscale-ssh$' "$LOG_FILE")" -eq 1 ]]

PATH="$FAKEBIN:$PATH" \
HOME="$HOME_DIR" \
XDG_CONFIG_HOME="$XDG_DIR" \
FAKE_TUNNEL_LOG="$LOG_FILE" \
BOOTSTRAP_ROOT="$FIXTURE" \
  "$ROOT_DIR/scripts/bootstrap.sh" --check --no-open

TARGET_FAKEBIN="$TMP_ROOT/target-bin"
mkdir -p "$TARGET_FAKEBIN"
cat > "$TARGET_FAKEBIN/tailscale" <<'SH'
#!/usr/bin/env bash
case "${1:-}" in
  status) exit 0 ;;
  debug) printf '{"RunSSH": true}\n' ;;
  ip) printf '100.64.0.10\n' ;;
  *) exit 0 ;;
esac
SH
cat > "$TARGET_FAKEBIN/id" <<'SH'
#!/usr/bin/env bash
case "${1:-}" in
  -u) printf '0\n' ;;
  -nG) printf 'chatgpt systemd-journal\n' ;;
  chatgpt) printf 'uid=1001(chatgpt) gid=1001(chatgpt) groups=1001(chatgpt)\n' ;;
  *) exit 0 ;;
esac
SH
cat > "$TARGET_FAKEBIN/getent" <<'SH'
#!/usr/bin/env bash
if [[ "$1" == passwd ]]; then
  printf 'chatgpt:x:1001:1001::/home/chatgpt:/bin/bash\n'
else
  printf 'systemd-journal:x:190:chatgpt\n'
fi
SH
chmod +x "$TARGET_FAKEBIN/tailscale" "$TARGET_FAKEBIN/id" "$TARGET_FAKEBIN/getent"
BOOTSTRAP_TARGET_ASSUME_ROOT=1 PATH="$TARGET_FAKEBIN:$PATH" "$ROOT_DIR/scripts/bootstrap-target.sh" --check --with-journal

RUN_FAKEBIN="$TMP_ROOT/run-bin"
RUN_LOG="$TMP_ROOT/run-local.log"
mkdir -p "$RUN_FAKEBIN"
cat > "$RUN_FAKEBIN/curl" <<'SH'
#!/usr/bin/env bash
exit 0
SH
cat > "$RUN_FAKEBIN/make" <<'SH'
#!/usr/bin/env bash
printf 'make\n' >> "${RUN_LOCAL_LOG}"
exit 0
SH
cat > "$RUN_FAKEBIN/tunnel-client" <<'SH'
#!/usr/bin/env bash
printf 'tunnel-client %s\n' "$*" >> "${RUN_LOCAL_LOG}"
exit 0
SH
chmod +x "$RUN_FAKEBIN/curl" "$RUN_FAKEBIN/make" "$RUN_FAKEBIN/tunnel-client"
: > "$RUN_LOG"
RUN_LOCAL_LOG="$RUN_LOG" \
PATH="$RUN_FAKEBIN:$PATH" \
BOOTSTRAP_ENV_FILE="$FIXTURE/.env" \
BOOTSTRAP_TUNNEL_ENV_FILE="$FIXTURE/.env.tunnel" \
BOOTSTRAP_LOG_DIR="$TMP_ROOT/logs" \
  "$ROOT_DIR/scripts/run-local.sh"
[[ ! -s "$RUN_LOG" ]]

printf 'bootstrap tests passed\n'
