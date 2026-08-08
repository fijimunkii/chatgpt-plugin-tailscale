# ChatGPT Tailscale SSH Inspector

[![CI](https://github.com/fijimunkii/chatgpt-plugin-tailscale/actions/workflows/ci.yml/badge.svg)](https://github.com/fijimunkii/chatgpt-plugin-tailscale/actions/workflows/ci.yml)

A small, security-first MCP server that lets ChatGPT inspect explicitly allowed Linux hosts over [Tailscale SSH](https://tailscale.com/docs/features/tailscale-ssh). This is an independent community project and is not affiliated with or endorsed by Tailscale or OpenAI.

The gateway embeds a Tailscale node with [`tsnet`](https://tailscale.com/docs/reference/tsnet-server-api), discovers peers from its local tailnet status, verifies SSH host keys distributed by Tailscale's control plane, and exposes only fixed read-only operations. It does **not** accept arbitrary commands and does **not** store SSH private keys.

> [!IMPORTANT]
> This is an initial implementation intended for private testing. Remote hostnames, logs, process arguments, and command output are untrusted data and can contain prompt-injection text. Keep the host allowlist narrow and review the threat model before deployment.

## Included tools

| Tool | Behavior |
| --- | --- |
| `list_hosts` | Lists only peers admitted by `TS_ALLOWED_TAGS` or `TS_ALLOWED_HOSTS`. |
| `inspect_host` | Returns tailnet metadata and readiness for one allowed peer. |
| `system_summary` | Fixed Linux read of kernel, uptime, load, memory, swap, and boot ID. |
| `disk_usage` | Fixed `df` inspection with bounded output. |
| `service_status` | Fixed `systemctl show` for an exact service allowlist. |
| `service_logs` | Fixed, line-bounded `journalctl` read for an exact service allowlist. |
| `process_list` | Fixed `ps` read; optional filtering happens locally in the gateway. |

All tools are marked read-only, idempotent, and open-world in MCP metadata: they do not modify targets, but they do read external systems.

## Security model

The gateway applies several independent boundaries:

1. Its own Tailscale identity should be `tag:chatgpt-ssh`.
2. Tailscale policy grants that tag TCP/22 and Tailscale SSH access only to `tag:chatgpt-managed` targets.
3. The gateway independently filters the netmap with `TS_ALLOWED_TAGS` and `TS_ALLOWED_HOSTS`.
4. Shared nodes are excluded unless `TS_ALLOW_SHARED_NODES=true`.
5. Execution requires an online, unexpired Linux peer with a Tailscale IP and control-plane-distributed SSH host keys.
6. The remote login is a dedicated non-root account, `chatgpt` by default.
7. Commands are opaque values constructible only by the fixed command package. No MCP argument or generic string can reach the SSH runner as a command.
8. Timeouts and a combined stdout/stderr byte limit bound each operation.
9. Logs contain operation metadata but never command output or secrets.

Read [`docs/threat-model.md`](docs/threat-model.md) before enabling access to sensitive hosts.

## Prerequisites

- Go 1.26.5 or later.
- A Tailscale tailnet.
- Tailscale SSH enabled on each target: `sudo tailscale set --ssh`.
- A dedicated `chatgpt` OS user on each target.
- A tagged auth key, or a Tailscale OAuth client with `auth_keys` write scope and permission to advertise `tag:chatgpt-ssh`.
- A tailnet policy granting the gateway only the required access.

The target OS user needs enough local permission for the selected reads. `systemctl show`, `ps`, and `df` are usually available without elevation. Reading the system journal commonly requires membership in `systemd-journal` or `adm`; grant that only when service logs are required.

## 1. Add the Tailscale policy

Merge [`docs/tailscale-policy.hujson`](docs/tailscale-policy.hujson) into the existing tailnet policy. Do not replace unrelated grants or SSH rules blindly.

The essential rules are:

```jsonc
{
  "tagOwners": {
    "tag:chatgpt-ssh": ["autogroup:admin"],
    "tag:chatgpt-managed": ["autogroup:admin"]
  },
  "grants": [
    {
      "src": ["tag:chatgpt-ssh"],
      "dst": ["tag:chatgpt-managed"],
      "ip": ["tcp:22"]
    }
  ],
  "ssh": [
    {
      "action": "accept",
      "src": ["tag:chatgpt-ssh"],
      "dst": ["tag:chatgpt-managed"],
      "users": ["chatgpt"]
    }
  ]
}
```

Tagged devices can use Tailscale SSH only to other tagged devices, so targets must actually carry `tag:chatgpt-managed`.

## 2. Configure the gateway

```bash
cp config/example.env .env
chmod 600 .env
```

Edit `.env` and provide either `TS_AUTHKEY` or `TS_CLIENT_SECRET`, but not both. When using `TS_CLIENT_SECRET`, the OAuth client needs `auth_keys` write scope and the advertised gateway tag. The default MCP listener is loopback-only and the default target selector is `tag:chatgpt-managed`.

Service inspection is disabled until `SSH_ALLOWED_SERVICES` is populated with exact systemd unit names, for example:

```dotenv
SSH_ALLOWED_SERVICES=tailscaled.service,nginx.service
```

Load the environment and run:

```bash
set -a
. ./.env
set +a

go run ./cmd/gateway
```

The MCP endpoint is `http://127.0.0.1:8080/mcp`. Health checks are available at `/healthz` and `/readyz`.

## 3. Connect ChatGPT

ChatGPT connects to remote MCP servers. For a gateway on a private network or developer machine, use OpenAI's Secure MCP Tunnel and point it at the loopback MCP endpoint. Then create a custom app in ChatGPT developer mode and scan the endpoint's tools.

The loopback default intentionally permits the local tunnel or an authenticated reverse proxy to provide the outer authentication layer. If the MCP SDK rejects a tunneled request with an invalid Host-header error, set:

```dotenv
MCP_DISABLE_LOCALHOST_PROTECTION=true
```

Use that exception only when the listener remains private behind the trusted tunnel. The gateway also rejects requests carrying a browser `Origin` header.

For a non-loopback listener, configuration fails closed unless either:

- `MCP_BEARER_TOKEN` contains at least 24 characters, or
- `MCP_ALLOW_UNAUTHENTICATED=true` is explicitly set.

The second option is dangerous and should be reserved for a private listener behind an authenticating proxy. The built-in bearer check is useful for compatible MCP clients and edge testing; a public ChatGPT deployment should use the platform-supported authentication mechanism at the tunnel or reverse-proxy layer.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `MCP_LISTEN_ADDR` | `127.0.0.1:8080` | HTTP listener. |
| `MCP_BEARER_TOKEN` | empty | Optional static bearer token; required by default off loopback. |
| `MCP_ALLOW_UNAUTHENTICATED` | `false` | Explicitly allow a non-loopback listener without the built-in token. |
| `MCP_DISABLE_LOCALHOST_PROTECTION` | `false` | Disable MCP SDK loopback Host-header protection for a trusted tunnel only. |
| `MCP_MAX_REQUEST_BODY_BYTES` | `1048576` | Maximum MCP request body. |
| `TS_HOSTNAME` | `chatgpt-ssh-gateway` | Embedded node hostname. |
| `TS_STATE_DIR` | `./data/tsnet` | Persistent `tsnet` identity and state. |
| `TS_AUTHKEY` | empty | Tailscale auth key. |
| `TS_CLIENT_SECRET` | empty | Tailscale OAuth client secret used by `tsnet` to mint an auth key. |
| `TS_ADVERTISE_TAGS` | `tag:chatgpt-ssh` | Tags requested by the gateway node. |
| `TS_ALLOWED_TAGS` | `tag:chatgpt-managed` | Exact target tags admitted by the local allowlist. |
| `TS_ALLOWED_HOSTS` | empty | Optional exact target IDs, names, or Tailscale IPs. |
| `TS_ALLOW_SHARED_NODES` | `false` | Admit shared-to nodes after other allowlist checks. |
| `TS_EPHEMERAL` | `false` | Register the embedded node as ephemeral. |
| `TS_CONTROL_URL` | empty | Optional alternate coordination server. |
| `TS_UP_TIMEOUT` | `90s` | Tailnet startup deadline. |
| `SSH_USER` | `chatgpt` | Remote OS account. |
| `SSH_ALLOWED_SERVICES` | empty | Exact systemd unit allowlist; empty disables service tools. |
| `SSH_DIAL_TIMEOUT` | `10s` | Tailnet TCP dial deadline. |
| `SSH_COMMAND_TIMEOUT` | `15s` | End-to-end command deadline. |
| `SSH_MAX_OUTPUT_BYTES` | `262144` | Combined stdout/stderr capture limit. |
| `SSH_MAX_LOG_LINES` | `200` | Maximum journal lines per call. |
| `SSH_MAX_PROCESSES` | `100` | Maximum returned process rows. |
| `SERVER_SHUTDOWN_TIMEOUT` | `10s` | Graceful HTTP shutdown deadline. |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error`. |

At least one target tag or host selector is always required. The supplied default is intentionally narrow and does not mean that the corresponding Tailscale tag exists automatically. For `TS_ALLOWED_HOSTS`, prefer a stable node ID, full MagicDNS name, or Tailscale IP. A short selector matching multiple peers fails closed instead of admitting all matches.

## Docker

Build:

```bash
docker build -t chatgpt-plugin-tailscale .
```

For Linux and a loopback Secure MCP Tunnel, host networking is simplest:

```bash
docker run --rm \
  --network host \
  --env-file .env \
  -v chatgpt-tailscale-state:/var/lib/chatgpt-plugin-tailscale \
  -e TS_STATE_DIR=/var/lib/chatgpt-plugin-tailscale/tsnet \
  chatgpt-plugin-tailscale
```

When publishing a container port instead, bind it to host loopback and configure a bearer token because the process itself listens on `0.0.0.0` inside the container:

```bash
docker run --rm \
  --env-file .env \
  -e MCP_LISTEN_ADDR=0.0.0.0:8080 \
  -e MCP_BEARER_TOKEN="$(openssl rand -hex 32)" \
  -p 127.0.0.1:8080:8080 \
  -v chatgpt-tailscale-state:/var/lib/chatgpt-plugin-tailscale \
  -e TS_STATE_DIR=/var/lib/chatgpt-plugin-tailscale/tsnet \
  chatgpt-plugin-tailscale
```

## Development

```bash
make check
```

The test suite covers configuration fail-closed behavior, target filtering and ambiguous allowlist rejection, opaque command construction, local process filtering, bearer authentication, strict host-key verification, and combined output limits.

## Current limitations

- Read-only inspection only; no arbitrary command execution.
- Linux targets and Linux inspection commands only.
- Tailscale SSH is required. Traditional SSH keys are not supported.
- Systemd is required for service status and service logs.
- One embedded Tailscale identity and one tailnet per process.
- No built-in OAuth authorization server; use Secure MCP Tunnel or an authenticating reverse proxy.
- The Tailscale Admin API is not needed for v0.1. Inventory comes from the embedded node's netmap/LocalAPI, avoiding broad administrative credentials.

See [`docs/architecture.md`](docs/architecture.md) for component details and the planned path to separately approved write actions.

## License

MIT
