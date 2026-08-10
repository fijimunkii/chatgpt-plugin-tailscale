is_interactive() {
  ! $NON_INTERACTIVE && [[ -t 0 ]]
}

prompt_yes_no() {
  local question="$1" default_answer="${2:-y}" answer
  if ! is_interactive; then
    [[ "$default_answer" == "y" ]]
    return
  fi
  if [[ "$default_answer" == "y" ]]; then
    read -r -p "$question [Y/n] " answer
    answer="${answer:-y}"
  else
    read -r -p "$question [y/N] " answer
    answer="${answer:-n}"
  fi
  [[ "$answer" == "y" || "$answer" == "Y" || "$answer" == "yes" || "$answer" == "YES" ]]
}

prompt_value() {
  local label="$1" current="${2:-}" answer
  if ! is_interactive; then
    printf '%s' "$current"
    return
  fi
  if [[ -n "$current" ]]; then
    read -r -p "$label [$current]: " answer
    printf '%s' "${answer:-$current}"
  else
    read -r -p "$label: " answer
    printf '%s' "$answer"
  fi
}

prompt_secret() {
  local label="$1" answer
  if ! is_interactive; then
    printf ''
    return
  fi
  read -r -s -p "$label: " answer
  printf '\n' >&2
  printf '%s' "$answer"
}

open_url() {
  local url="$1"
  $NO_OPEN && return 0
  if command -v open >/dev/null 2>&1; then
    open "$url" >/dev/null 2>&1 || true
  elif command -v xdg-open >/dev/null 2>&1; then
    xdg-open "$url" >/dev/null 2>&1 || true
  fi
}

env_get() {
  local file="$1" key="$2"
  [[ -f "$file" ]] || return 0
  awk -v key="$key" '
    index($0, key "=") == 1 {
      sub("^[^=]*=", "")
      print
      exit
    }
  ' "$file"
}

env_set() {
  local file="$1" key="$2" value="$3" tmp
  mkdir -p "$(dirname "$file")"
  [[ -f "$file" ]] || : > "$file"
  tmp="$(mktemp "${TMPDIR:-/tmp}/chatgpt-ts-env.XXXXXX")"
  awk -v key="$key" -v value="$value" '
    BEGIN { found = 0 }
    index($0, key "=") == 1 {
      if (!found) print key "=" value
      found = 1
      next
    }
    { print }
    END { if (!found) print key "=" value }
  ' "$file" > "$tmp"
  cat "$tmp" > "$file"
  rm -f "$tmp"
  chmod 600 "$file"
}

env_has_key() {
  local file="$1" key="$2"
  [[ -f "$file" ]] && grep -Eq "^${key}=" "$file"
}

env_default() {
  local file="$1" key="$2" value="$3"
  if ! env_has_key "$file" "$key"; then
    env_set "$file" "$key" "$value"
  fi
}

resolve_from_root() {
  local path="$1"
  case "$path" in
    /*) printf '%s' "$path" ;;
    *) printf '%s/%s' "$ROOT_DIR" "$path" ;;
  esac
}

dir_has_state() {
  local dir="$1"
  [[ -d "$dir" ]] && find "$dir" -mindepth 1 -print -quit 2>/dev/null | grep -q .
}

profile_path() {
  printf '%s/tunnel-client/%s.yaml' "${XDG_CONFIG_HOME:-$HOME/.config}" "$PROFILE"
}

profile_matches() {
  local file="$1" tunnel_id="$2" health_addr="${3:-$TUNNEL_HEALTH_ADDR}" mcp_url="${4:-$MCP_URL}"
  [[ -f "$file" ]] || return 1
  grep -Fq "tunnel_id: \"$tunnel_id\"" "$file" &&
    grep -Fq "listen_addr: \"$health_addr\"" "$file" &&
    grep -Fq "url: \"$mcp_url\"" "$file"
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "$1 is required"
}

preflight() {
  section "Local prerequisites"
  require_command bash
  require_command curl
  ok "bash: $(bash --version | head -n 1)"

  if command -v go >/dev/null 2>&1; then
    ok "Go: $(go version)"
  elif [[ -x "$ROOT_DIR/bin/gateway" ]]; then
    ok "gateway: using prebuilt release binary at $ROOT_DIR/bin/gateway (Go toolchain not required)"
  else
    die "Go is required when no prebuilt bin/gateway is present"
  fi

  if command -v tunnel-client >/dev/null 2>&1; then
    ok "tunnel-client: $(tunnel-client --version 2>/dev/null || printf installed)"
  else
    warn "tunnel-client is not installed or not on PATH"
    open_url "https://platform.openai.com/settings/organization/tunnels"
    info "download the supported build from OpenAI Platform → Organization → Tunnels"
    info "after approving the binary in macOS Privacy & Security, rerun ./bootstrap"
  fi
}
