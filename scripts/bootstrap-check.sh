check_setup() {
  local failures=0 state_dir p profile_name health_addr
  section "Bootstrap diagnostics"

  for cmd in go curl tunnel-client; do
    if command -v "$cmd" >/dev/null 2>&1; then
      ok "$cmd is installed"
    else
      fail "$cmd is missing"
      failures=$((failures + 1))
    fi
  done

  if [[ -f "$ENV_FILE" ]]; then
    ok "$ENV_FILE exists"
    state_dir="$(resolve_from_root "$(env_get "$ENV_FILE" TS_STATE_DIR)")"
    if dir_has_state "$state_dir"; then
      ok "tsnet identity exists at $state_dir"
    elif [[ -n "$(env_get "$ENV_FILE" TS_AUTHKEY)$(env_get "$ENV_FILE" TS_CLIENT_SECRET)" ]]; then
      ok "Tailscale enrollment credential is present for first start"
    else
      fail "no tsnet state or enrollment credential is configured"
      failures=$((failures + 1))
    fi
  else
    fail "$ENV_FILE is missing"
    failures=$((failures + 1))
  fi

  if [[ -f "$TUNNEL_ENV_FILE" ]]; then
    ok "$TUNNEL_ENV_FILE exists"
    profile_name="$(env_get "$TUNNEL_ENV_FILE" TUNNEL_CLIENT_PROFILE)"
    profile_name="${profile_name:-$PROFILE}"
    health_addr="$(env_get "$TUNNEL_ENV_FILE" TUNNEL_CLIENT_HEALTH_ADDR)"
    health_addr="${health_addr:-$TUNNEL_HEALTH_ADDR}"
    p="${XDG_CONFIG_HOME:-$HOME/.config}/tunnel-client/${profile_name}.yaml"
    if [[ -f "$p" ]]; then
      tunnel_id="$(env_get "$TUNNEL_ENV_FILE" CONTROL_PLANE_TUNNEL_ID)"
      mcp_url="$(env_get "$TUNNEL_ENV_FILE" MCP_SERVER_URL)"
      mcp_url="${mcp_url:-$MCP_URL}"
      if profile_matches "$p" "$tunnel_id" "$health_addr" "$mcp_url"; then
        ok "tunnel-client profile matches local settings: $p"
      else
        fail "tunnel-client profile does not match the tunnel ID, MCP URL, or health listener"
        failures=$((failures + 1))
      fi
    else
      fail "tunnel-client profile is missing: $p"
      failures=$((failures + 1))
    fi
  else
    fail "$TUNNEL_ENV_FILE is missing"
    failures=$((failures + 1))
    health_addr="$TUNNEL_HEALTH_ADDR"
  fi

  if curl -fsS --max-time 2 "http://127.0.0.1:8080/readyz" >/dev/null 2>&1; then
    ok "gateway is ready at http://127.0.0.1:8080"
  else
    warn "gateway is not currently ready at http://127.0.0.1:8080"
  fi
  if curl -fsS --max-time 2 "http://$health_addr/readyz" >/dev/null 2>&1; then
    ok "tunnel-client is ready at http://$health_addr"
  else
    warn "tunnel-client is not currently ready at http://$health_addr"
  fi

  if ((failures)); then
    fail "$failures required bootstrap check(s) failed"
    return 1
  fi
  ok "static bootstrap configuration is complete"
}
