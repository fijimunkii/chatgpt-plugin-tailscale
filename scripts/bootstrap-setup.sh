policy_walkthrough() {
  $SKIP_POLICY && return 0
  section "Tailscale policy"
  info "Merge, do not replace, the entries in: docs/tailscale-policy.hujson"
  sed -n '1,80p' "$ROOT_DIR/docs/tailscale-policy.hujson"
  open_url "https://login.tailscale.com/admin/acls"
  if $NON_INTERACTIVE; then
    [[ "${BOOTSTRAP_ACCEPT_POLICY:-}" == "1" ]] || die "set BOOTSTRAP_ACCEPT_POLICY=1 or pass --skip-policy"
  else
    prompt_yes_no "Have you merged and saved these policy entries?" y || die "apply the policy before continuing"
  fi
  ok "policy step acknowledged"
}

run_target_bootstrap() {
  $SKIP_TARGET && return 0
  section "Linux target"

  if [[ -z "$TARGET_ADMIN" ]] && is_interactive; then
    TARGET_ADMIN="$(prompt_value "Existing admin SSH target (for example ubuntu@server; blank to do later)" "")"
  fi

  if [[ -n "$TARGET_ADMIN" ]]; then
    require_command ssh
    require_command scp
    local remote_path target_args remote_cmd
    remote_path="/tmp/chatgpt-tailscale-bootstrap.$$"
    target_args="--user chatgpt"
    $WITH_JOURNAL && target_args="$target_args --with-journal"
    if $NON_INTERACTIVE || prompt_yes_no "Configure $TARGET_ADMIN now?" y; then
      scp -q "$ROOT_DIR/scripts/bootstrap-target.sh" "$TARGET_ADMIN:$remote_path"
      printf '\n'
      remote_cmd="sudo bash '$remote_path' $target_args; rc=\$?; rm -f '$remote_path'; exit \$rc"
      ssh -t "$TARGET_ADMIN" "$remote_cmd"
      ok "target helper completed"
    fi
  else
    info "To configure a target later:"
    info "  scp scripts/bootstrap-target.sh ADMIN@HOST:/tmp/bootstrap-target.sh"
    info "  ssh -t ADMIN@HOST 'sudo bash /tmp/bootstrap-target.sh --with-journal'"
  fi

  info "In Tailscale Machines, apply tag:chatgpt-managed to the target."
  open_url "https://login.tailscale.com/admin/machines"
  if ! $NON_INTERACTIVE; then
    prompt_yes_no "Has at least one prepared target been tagged tag:chatgpt-managed?" y || warn "the gateway will not list a target until it is tagged"
  fi
  info "Manual test: ssh chatgpt+password@TARGET_TAILSCALE_IP, then enter any non-empty password."
}

configure_gateway_env() {
  section "Gateway configuration"
  if [[ ! -f "$ENV_FILE" ]]; then
    cp "$ROOT_DIR/config/example.env" "$ENV_FILE"
    ok "created $ENV_FILE from config/example.env"
  else
    ok "preserving existing $ENV_FILE"
  fi
  chmod 600 "$ENV_FILE"

  env_default "$ENV_FILE" TS_HOSTNAME chatgpt-ssh-gateway
  env_default "$ENV_FILE" TS_STATE_DIR ./data/tsnet
  env_default "$ENV_FILE" TS_ADVERTISE_TAGS tag:chatgpt-ssh
  env_default "$ENV_FILE" TS_ALLOWED_TAGS tag:chatgpt-managed
  env_default "$ENV_FILE" TS_ALLOW_SHARED_NODES false
  env_default "$ENV_FILE" SSH_USER chatgpt
  env_default "$ENV_FILE" SSH_ALLOWED_SERVICES tailscaled.service
  env_default "$ENV_FILE" MCP_LISTEN_ADDR 127.0.0.1:8080

  local state_dir auth_key client_secret supplied_auth supplied_client choice
  state_dir="$(resolve_from_root "$(env_get "$ENV_FILE" TS_STATE_DIR)")"
  auth_key="$(env_get "$ENV_FILE" TS_AUTHKEY)"
  client_secret="$(env_get "$ENV_FILE" TS_CLIENT_SECRET)"
  supplied_auth="${TS_AUTHKEY:-}"
  supplied_client="${TS_CLIENT_SECRET:-}"

  if [[ -n "$supplied_auth" ]]; then
    auth_key="$supplied_auth"
    env_set "$ENV_FILE" TS_AUTHKEY "$auth_key"
    env_set "$ENV_FILE" TS_CLIENT_SECRET ""
  elif [[ -n "$supplied_client" ]]; then
    client_secret="$supplied_client"
    env_set "$ENV_FILE" TS_CLIENT_SECRET "$client_secret"
    env_set "$ENV_FILE" TS_AUTHKEY ""
  fi

  if [[ -n "$auth_key" && -n "$client_secret" ]]; then
    die "$ENV_FILE contains both TS_AUTHKEY and TS_CLIENT_SECRET; keep exactly one"
  fi

  if dir_has_state "$state_dir"; then
    ok "existing tsnet identity found at $state_dir; enrollment is reusable"
    if [[ -n "$auth_key" ]]; then
      warn "TS_AUTHKEY remains in $ENV_FILE; after confirming restarts work, you may clear this consumed one-off key"
    fi
  elif [[ -n "$auth_key" || -n "$client_secret" ]]; then
    ok "Tailscale enrollment credential is configured"
  else
    if $NON_INTERACTIVE; then
      die "first run requires TS_AUTHKEY or TS_CLIENT_SECRET"
    fi
    info "Generate a one-off, non-ephemeral auth key carrying tag:chatgpt-ssh."
    open_url "https://login.tailscale.com/admin/settings/keys"
    choice="$(prompt_value "Enrollment method: auth-key or oauth-client" "auth-key")"
    case "$choice" in
      auth-key|auth|key)
        auth_key="$(prompt_secret "Tailscale auth key (tskey-auth-...)")"
        [[ "$auth_key" == tskey-auth-* ]] || die "expected a tskey-auth-... value"
        env_set "$ENV_FILE" TS_AUTHKEY "$auth_key"
        env_set "$ENV_FILE" TS_CLIENT_SECRET ""
        ;;
      oauth-client|oauth|client)
        client_secret="$(prompt_secret "Tailscale OAuth client secret (tskey-client-...)")"
        [[ "$client_secret" == tskey-client-* ]] || die "expected a tskey-client-... value"
        env_set "$ENV_FILE" TS_CLIENT_SECRET "$client_secret"
        env_set "$ENV_FILE" TS_AUTHKEY ""
        ;;
      *)
        die "choose auth-key or oauth-client"
        ;;
    esac
    ok "stored the Tailscale credential in gitignored, mode-600 $ENV_FILE"
  fi

  ok "gateway identity: $(env_get "$ENV_FILE" TS_HOSTNAME) / $(env_get "$ENV_FILE" TS_ADVERTISE_TAGS)"
  ok "target allowlist: $(env_get "$ENV_FILE" TS_ALLOWED_TAGS)"
  ok "remote user: $(env_get "$ENV_FILE" SSH_USER)"
}

configure_tunnel() {
  section "OpenAI Secure MCP Tunnel"
  if ! command -v tunnel-client >/dev/null 2>&1; then
    die "install tunnel-client, put it on PATH, and rerun ./bootstrap; completed policy, target, and gateway settings are preserved"
  fi

  [[ -f "$TUNNEL_ENV_FILE" ]] || : > "$TUNNEL_ENV_FILE"
  chmod 600 "$TUNNEL_ENV_FILE"

  local tunnel_id runtime_key file_tunnel_id file_runtime_key p backup
  file_tunnel_id="$(env_get "$TUNNEL_ENV_FILE" CONTROL_PLANE_TUNNEL_ID)"
  file_runtime_key="$(env_get "$TUNNEL_ENV_FILE" CONTROL_PLANE_API_KEY)"
  tunnel_id="${CONTROL_PLANE_TUNNEL_ID:-$file_tunnel_id}"
  runtime_key="${CONTROL_PLANE_API_KEY:-$file_runtime_key}"

  if [[ -z "$tunnel_id" ]]; then
    $NON_INTERACTIVE && die "CONTROL_PLANE_TUNNEL_ID is required"
    open_url "https://platform.openai.com/settings/organization/tunnels"
    tunnel_id="$(prompt_value "OpenAI tunnel ID" "")"
  fi
  [[ "$tunnel_id" =~ ^tunnel_[0-9a-f]{32}$ ]] || die "invalid tunnel ID; expected tunnel_ followed by 32 lowercase hex characters"

  if [[ -z "$runtime_key" ]]; then
    $NON_INTERACTIVE && die "CONTROL_PLANE_API_KEY is required"
    open_url "https://platform.openai.com/settings/organization/api-keys"
    runtime_key="$(prompt_secret "OpenAI runtime API key with Tunnels Read + Use")"
  fi
  [[ "$runtime_key" == sk-* ]] || die "expected an OpenAI key beginning with sk-"

  env_set "$TUNNEL_ENV_FILE" CONTROL_PLANE_TUNNEL_ID "$tunnel_id"
  env_set "$TUNNEL_ENV_FILE" CONTROL_PLANE_API_KEY "$runtime_key"
  env_set "$TUNNEL_ENV_FILE" TUNNEL_CLIENT_PROFILE "$PROFILE"
  env_set "$TUNNEL_ENV_FILE" TUNNEL_CLIENT_HEALTH_ADDR "$TUNNEL_HEALTH_ADDR"
  env_set "$TUNNEL_ENV_FILE" MCP_SERVER_URL "$MCP_URL"
  ok "stored tunnel settings in gitignored, mode-600 $TUNNEL_ENV_FILE"

  p="$(profile_path)"
  if profile_matches "$p" "$tunnel_id"; then
    ok "matching tunnel-client profile already exists: $p"
    return 0
  fi

  if [[ -f "$p" ]]; then
    warn "profile exists but does not match the selected tunnel, MCP URL, or health listener: $p"
    if $RECONFIGURE_TUNNEL || { ! $NON_INTERACTIVE && prompt_yes_no "Back up and recreate this profile?" n; }; then
      backup="$p.backup.$(date -u +%Y%m%dT%H%M%SZ)"
      cp "$p" "$backup"
      rm -f "$p"
      ok "backed up existing profile to $backup"
    else
      warn "leaving the existing profile unchanged; rerun with --reconfigure-tunnel"
      return 0
    fi
  fi

  CONTROL_PLANE_API_KEY="$runtime_key" tunnel-client init \
    --sample sample_mcp_remote_no_auth \
    --profile "$PROFILE" \
    --tunnel-id "$tunnel_id" \
    --mcp-server-url "$MCP_URL" \
    --health-listen-addr "$TUNNEL_HEALTH_ADDR"
  ok "created tunnel-client profile $PROFILE with health listener $TUNNEL_HEALTH_ADDR"
}
