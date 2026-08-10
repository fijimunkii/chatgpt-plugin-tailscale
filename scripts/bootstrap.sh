#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="${BOOTSTRAP_ROOT:-$(cd "$SCRIPT_DIR/.." && pwd)}"
ENV_FILE="${BOOTSTRAP_ENV_FILE:-$ROOT_DIR/.env}"
TUNNEL_ENV_FILE="${BOOTSTRAP_TUNNEL_ENV_FILE:-$ROOT_DIR/.env.tunnel}"
PROFILE="${BOOTSTRAP_PROFILE:-tailscale-ssh}"
MCP_URL="${BOOTSTRAP_MCP_URL:-http://127.0.0.1:8080/mcp}"
TUNNEL_HEALTH_ADDR="${BOOTSTRAP_TUNNEL_HEALTH_ADDR:-127.0.0.1:8081}"
TARGET_ADMIN="${BOOTSTRAP_TARGET:-}"
WITH_JOURNAL=false
CHECK_ONLY=false
NON_INTERACTIVE=false
RECONFIGURE_TUNNEL=false
NO_OPEN=false
SKIP_POLICY=false
SKIP_TARGET=false

usage() {
  cat <<'USAGE'
Usage: ./bootstrap [options]

Interactive, rerunnable setup for the TetherBound Community Tailscale SSH MCP
gateway. Existing configuration, target users, Tailscale SSH settings, and
matching tunnel-client profiles are preserved.

Options:
  --check                 Read-only diagnostics
  --non-interactive       Never prompt; read values from environment/files
  --target USER@HOST      Configure a Linux target over existing admin SSH
  --with-journal          Grant the target account journal-read membership
  --profile NAME          tunnel-client profile (default: tailscale-ssh)
  --reconfigure-tunnel    Back up and recreate a mismatched tunnel profile
  --skip-policy           Skip the policy walkthrough
  --skip-target           Skip target preparation and tagging walkthrough
  --no-open               Do not open admin-console pages in a browser
  -h, --help              Show this help

Non-interactive inputs:
  TS_AUTHKEY or TS_CLIENT_SECRET
  CONTROL_PLANE_TUNNEL_ID
  CONTROL_PLANE_API_KEY
  BOOTSTRAP_ACCEPT_POLICY=1 (unless --skip-policy)
USAGE
}

section() { printf '\n== %s ==\n' "$*"; }
info() { printf '  • %s\n' "$*"; }
ok() { printf '  ✓ %s\n' "$*"; }
warn() { printf '  ! %s\n' "$*" >&2; }
fail() { printf '  ✗ %s\n' "$*" >&2; }
die() { fail "$*"; exit 1; }

while (($#)); do
  case "$1" in
    --check)
      CHECK_ONLY=true
      shift
      ;;
    --non-interactive)
      NON_INTERACTIVE=true
      shift
      ;;
    --target)
      [[ $# -ge 2 ]] || die "--target requires USER@HOST"
      TARGET_ADMIN="$2"
      shift 2
      ;;
    --with-journal)
      WITH_JOURNAL=true
      shift
      ;;
    --profile)
      [[ $# -ge 2 ]] || die "--profile requires a value"
      PROFILE="$2"
      shift 2
      ;;
    --reconfigure-tunnel)
      RECONFIGURE_TUNNEL=true
      shift
      ;;
    --skip-policy)
      SKIP_POLICY=true
      shift
      ;;
    --skip-target)
      SKIP_TARGET=true
      shift
      ;;
    --no-open)
      NO_OPEN=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
done

[[ "$PROFILE" =~ ^[A-Za-z0-9._-]+$ ]] || die "unsafe tunnel profile name: $PROFILE"

# Keep the operator-facing entry point small; implementation is grouped by concern.
# shellcheck source=scripts/bootstrap-common.sh
. "$SCRIPT_DIR/bootstrap-common.sh"
# shellcheck source=scripts/bootstrap-setup.sh
. "$SCRIPT_DIR/bootstrap-setup.sh"
# shellcheck source=scripts/bootstrap-check.sh
. "$SCRIPT_DIR/bootstrap-check.sh"
printf 'TetherBound Community / ChatGPT Tailscale MCP bootstrap\n'
printf 'Safe to rerun: existing matching resources are preserved.\n'

if $CHECK_ONLY; then
  check_setup
  exit $?
fi

preflight
policy_walkthrough
run_target_bootstrap
configure_gateway_env
configure_tunnel

section "Next step"
info "Run both local processes with: ./scripts/run-local.sh"
info "Then create/open the ChatGPT custom MCP app for this tunnel and scan its tools."
info "Rerun ./bootstrap at any time; use ./bootstrap --check for diagnostics."

if is_interactive && command -v tunnel-client >/dev/null 2>&1; then
  if prompt_yes_no "Start the gateway and tunnel now?" y; then
    exec "$ROOT_DIR/scripts/run-local.sh"
  fi
fi
