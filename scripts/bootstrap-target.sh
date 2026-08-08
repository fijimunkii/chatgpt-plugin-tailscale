#!/usr/bin/env bash
set -euo pipefail

ORIGINAL_ARGS=("$@")
SSH_USER_NAME="chatgpt"
WITH_JOURNAL=false
CHECK_ONLY=false

usage() {
  cat <<'USAGE'
Usage: bootstrap-target.sh [options]

Prepare a Linux machine for the read-only Tailscale SSH gateway. The operation
is idempotent: an existing user is preserved, existing group membership is
preserved, and enabling Tailscale SSH again is safe.

Options:
  --user NAME       Dedicated local SSH account (default: chatgpt)
  --with-journal    Add the account to systemd-journal or adm when available
  --check           Read-only verification; do not change the target
  -h, --help        Show this help
USAGE
}

info() { printf '  • %s\n' "$*"; }
ok() { printf '  ✓ %s\n' "$*"; }
warn() { printf '  ! %s\n' "$*" >&2; }
die() { printf '  ✗ %s\n' "$*" >&2; exit 1; }

while (($#)); do
  case "$1" in
    --user)
      [[ $# -ge 2 ]] || die "--user requires a value"
      SSH_USER_NAME="$2"
      shift 2
      ;;
    --with-journal)
      WITH_JOURNAL=true
      shift
      ;;
    --check)
      CHECK_ONLY=true
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

[[ "$SSH_USER_NAME" =~ ^[a-z_][a-z0-9_-]{0,31}$ ]] || die "unsafe user name: $SSH_USER_NAME"
[[ "$(uname -s)" == "Linux" ]] || die "this helper supports Linux targets only"

if [[ "${BOOTSTRAP_TARGET_ASSUME_ROOT:-0}" != "1" && ${EUID:-$(id -u)} -ne 0 ]]; then
  command -v sudo >/dev/null 2>&1 || die "run as root or install sudo"
  exec sudo -- "$0" "${ORIGINAL_ARGS[@]}"
fi

command -v tailscale >/dev/null 2>&1 || die "tailscale is not installed"
command -v getent >/dev/null 2>&1 || die "getent is required"

tailscale status >/dev/null 2>&1 || die "this machine is not connected to a tailnet"

printf '\nTarget account\n'
if id "$SSH_USER_NAME" >/dev/null 2>&1; then
  ok "local account $SSH_USER_NAME already exists"
else
  if $CHECK_ONLY; then
    warn "local account $SSH_USER_NAME is missing"
  else
    command -v useradd >/dev/null 2>&1 || die "useradd is required to create $SSH_USER_NAME"
    [[ -x /bin/bash ]] || die "/bin/bash is required for the dedicated account"
    useradd --create-home --shell /bin/bash "$SSH_USER_NAME"
    ok "created local account $SSH_USER_NAME with a locked password"
  fi
fi

if id "$SSH_USER_NAME" >/dev/null 2>&1; then
  passwd_entry="$(getent passwd "$SSH_USER_NAME")"
  home_dir="$(printf '%s' "$passwd_entry" | awk -F: '{print $6}')"
  login_shell="$(printf '%s' "$passwd_entry" | awk -F: '{print $7}')"
  info "home: $home_dir"
  info "shell: $login_shell"
fi

if $WITH_JOURNAL; then
  printf '\nJournal access\n'
  journal_group=""
  if getent group systemd-journal >/dev/null 2>&1; then
    journal_group="systemd-journal"
  elif getent group adm >/dev/null 2>&1; then
    journal_group="adm"
  fi

  if [[ -z "$journal_group" ]]; then
    warn "no systemd-journal or adm group exists; service_logs may be unavailable"
  elif id "$SSH_USER_NAME" >/dev/null 2>&1 && id -nG "$SSH_USER_NAME" | tr ' ' '\n' | grep -Fxq "$journal_group"; then
    ok "$SSH_USER_NAME is already a member of $journal_group"
  elif $CHECK_ONLY; then
    warn "$SSH_USER_NAME is not a member of $journal_group"
  else
    command -v usermod >/dev/null 2>&1 || die "usermod is required to grant journal access"
    usermod -aG "$journal_group" "$SSH_USER_NAME"
    ok "added $SSH_USER_NAME to $journal_group"
  fi
fi

printf '\nTailscale SSH\n'
if $CHECK_ONLY; then
  if prefs="$(tailscale debug prefs 2>/dev/null)"; then
    if printf '%s\n' "$prefs" | grep -Eq '"RunSSH"[[:space:]]*:[[:space:]]*true|RunSSH:[[:space:]]*true'; then
      ok "Tailscale SSH is enabled"
    else
      warn "Tailscale SSH does not appear to be enabled; run: sudo tailscale set --ssh"
    fi
  else
    warn "could not inspect Tailscale SSH preference with this client version"
  fi
else
  tailscale set --ssh
  ok "Tailscale SSH is enabled"
fi

ipv4="$(tailscale ip -4 2>/dev/null | head -n 1 || true)"

printf '\nTarget summary\n'
ok "target bootstrap completed"
info "dedicated account: $SSH_USER_NAME"
[[ -n "$ipv4" ]] && info "Tailscale IPv4: $ipv4"
info "no Linux password was set for $SSH_USER_NAME"
info "next: apply tag:chatgpt-managed in the Tailscale Machines page"
if [[ -n "$ipv4" ]]; then
  info "manual test: ssh ${SSH_USER_NAME}+password@${ipv4} (enter any non-empty password)"
fi
