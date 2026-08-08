# ChatGPT Tailscale SSH Inspector

[![CI](https://github.com/fijimunkii/chatgpt-plugin-tailscale/actions/workflows/ci.yml/badge.svg)](https://github.com/fijimunkii/chatgpt-plugin-tailscale/actions/workflows/ci.yml)

A small, security-first MCP server that lets ChatGPT inspect explicitly allowed
Linux hosts over [Tailscale SSH](https://tailscale.com/docs/features/tailscale-ssh).
This is an independent community project and is not affiliated with or endorsed
by Tailscale or OpenAI.

The gateway embeds a Tailscale node with
[`tsnet`](https://tailscale.com/docs/reference/tsnet-server-api), discovers peers
from its local tailnet status, verifies SSH host keys distributed by Tailscale's
control plane, and exposes only fixed read-only operations. It does **not**
accept arbitrary commands and does **not** store SSH private keys.

> [!IMPORTANT]
> This is an initial implementation intended for private testing. Remote
> hostnames, logs, process arguments, and command output are untrusted data and
> can contain prompt-injection text. Keep the host allowlist narrow and review
> the threat model before deployment.

## Quick start

The supported setup path is the interactive, rerunnable bootstrap wizard:

```bash
./bootstrap
```

It walks through all five systems involved in the setup:

1. Tailscale policy and tags.
2. The dedicated `chatgpt` account and Tailscale SSH on a Linux target.
3. The embedded gateway node and its tagged enrollment credential.
4. OpenAI Secure MCP Tunnel, including the runtime key, tunnel ID, profile, and
   the `8080`/`8081` port split.
5. Starting the gateway and tunnel before scanning tools in ChatGPT.

The wizard preserves existing configuration and is safe to run again. Use its
read-only diagnostics at any time:

```bash
./bootstrap --check
```

To configure a target through an existing administrator SSH path:

```bash
./bootstrap --target ubuntu@server.example --with-journal
```

After bootstrap, launch both local processes with one command:

```bash
./scripts/run-local.sh
# or: make up
```

The launcher reuses already healthy processes, waits for both readiness checks,
and stops only processes it started. The local endpoints are:

```text
MCP gateway:       http://127.0.0.1:8080/mcp
Gateway readiness: http://127.0.0.1:8080/readyz
Tunnel readiness:  http://127.0.0.1:8081/readyz
Tunnel admin UI:   http://127.0.0.1:8081/ui
```

See the complete [`bootstrap guide`](docs/bootstrap.md) for the exact admin-console
steps, rerun guarantees, non-interactive mode, target-only usage, macOS
Gatekeeper guidance, and troubleshooting.

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

All tools are marked read-only, idempotent, and open-world in MCP metadata: they
do not modify targets, but they do read external systems.

## Security model

The gateway applies several independent boundaries:

1. Its own Tailscale identity should be `tag:chatgpt-ssh`.
2. Tailscale policy grants that tag TCP/22 and Tailscale SSH access only to
   `tag:chatgpt-managed` targets.
3. The gateway independently filters the netmap with `TS_ALLOWED_TAGS` and
   `TS_ALLOWED_HOSTS`.
4. Shared nodes are excluded unless `TS_ALLOW_SHARED_NODES=true`.
5. Execution requires an online, unexpired Linux peer with a Tailscale IP and
   control-plane-distributed SSH host keys.
6. The remote login is a dedicated non-root account, `chatgpt` by default.
7. Commands are opaque values constructible only by the fixed command package.
   No MCP argument or generic string can reach the SSH runner as a command.
8. Timeouts and a combined stdout/stderr byte limit bound each operation.
9. Logs contain operation metadata but never command output or secrets.

The bootstrap keeps local credentials in gitignored `.env` and `.env.tunnel`
files with mode `0600`. For a shared or production deployment, replace those
files with an external secrets manager.

Read [`docs/threat-model.md`](docs/threat-model.md) before enabling access to
sensitive hosts.

## Tailscale policy

The bootstrap displays [`docs/tailscale-policy.hujson`](docs/tailscale-policy.hujson)
and opens the Access Controls page. Merge the entries into the existing policy;
do not replace unrelated rules:

```jsonc
{
  "tagOwners": {
    "tag:chatgpt-ssh": ["autogroup:admin"],
    "tag:chatgpt-managed": ["autogroup:admin"],
  },
  "grants": [
    {
      "src": ["tag:chatgpt-ssh"],
      "dst": ["tag:chatgpt-managed"],
      "ip": ["tcp:22"],
    },
  ],
  "ssh": [
    {
      "action": "accept",
      "src": ["tag:chatgpt-ssh"],
      "dst": ["tag:chatgpt-managed"],
      "users": ["chatgpt"],
    },
  ],
}
```

Tagged devices can use Tailscale SSH only to other tagged devices, so targets
must actually carry `tag:chatgpt-managed`. The full policy file includes
commented human test rules for `ssh chatgpt+password@...` when your existing
policy does not already allow that verification. Tailscale access rules are
additive; review existing wildcard grants because the narrow rule does not
subtract permissions granted elsewhere.

## Target preparation

The target helper is independently usable and idempotent:

```bash
sudo ./scripts/bootstrap-target.sh --with-journal
```

It creates the dedicated account only when missing, never sets a Linux password,
optionally grants journal-read membership, and runs:

```bash
sudo tailscale set --ssh
```

After applying `tag:chatgpt-managed`, test over the target's Tailscale IP:

```bash
ssh chatgpt+password@100.x.y.z
```

Enter any non-empty value. The `+password` compatibility suffix still uses
Tailscale identity rather than a Linux password.

## Configuration

`./bootstrap` creates `.env` from `config/example.env` when needed and preserves
an existing file.

| Variable | Default | Purpose |
| --- | --- | --- |
| `MCP_LISTEN_ADDR` | `127.0.0.1:8080` | HTTP listener. |
| `MCP_BEARER_TOKEN` | empty | Optional static bearer token; required by default off loopback. |
| `MCP_ALLOW_UNAUTHENTICATED` | `false` | Explicitly allow a non-loopback listener without the built-in token. |
| `MCP_DISABLE_LOCALHOST_PROTECTION` | `false` | Disable MCP SDK loopback Host-header protection for a trusted tunnel only. |
| `MCP_MAX_REQUEST_BODY_BYTES` | `1048576` | Maximum MCP request body. |
| `TS_HOSTNAME` | `chatgpt-ssh-gateway` | Embedded node hostname. |
| `TS_STATE_DIR` | `./data/tsnet` | Persistent `tsnet` identity and state. |
| `TS_AUTHKEY` | empty | Tailscale auth key used for first enrollment. |
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

At least one target tag or host selector is always required. For
`TS_ALLOWED_HOSTS`, prefer a stable node ID, full MagicDNS name, or Tailscale IP.
A short selector matching multiple peers fails closed instead of admitting all
matches.

Service inspection is limited to exact unit names:

```dotenv
SSH_ALLOWED_SERVICES=tailscaled.service,nginx.service
```

## Manual gateway run

The bootstrap launcher is recommended. To run only the gateway manually:

```bash
set -a
. ./.env
set +a

go run ./cmd/gateway
```

The MCP endpoint is `http://127.0.0.1:8080/mcp`. Health checks are available at
`/healthz` and `/readyz`.

The loopback default intentionally permits OpenAI Secure MCP Tunnel or an
authenticated reverse proxy to provide the outer authentication layer. If the
MCP SDK rejects a trusted tunneled request with an invalid Host-header error,
`MCP_DISABLE_LOCALHOST_PROTECTION=true` is available, but use that exception
only while the listener remains private.

For a non-loopback listener, configuration fails closed unless either:

- `MCP_BEARER_TOKEN` contains at least 24 characters, or
- `MCP_ALLOW_UNAUTHENTICATED=true` is explicitly set.

The second option is dangerous and should be reserved for a private listener
behind an authenticating proxy.

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

When publishing a container port instead, bind it to host loopback and configure
a bearer token because the process itself listens on `0.0.0.0` inside the
container:

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

`make check` validates the bootstrap scripts, runs their idempotency tests,
checks Go formatting, runs the Go test suite and vet, and builds the gateway.

Useful targets:

```bash
make bootstrap        # interactive setup
make bootstrap-check  # read-only diagnostics
make up               # gateway + tunnel-client
```

The test suite covers configuration fail-closed behavior, target filtering and
ambiguous allowlist rejection, opaque command construction, local process
filtering, bearer authentication, strict host-key verification, combined output
limits, bootstrap reruns, target checks, and tunnel profile reuse.

## Current limitations

- Read-only inspection only; no arbitrary command execution.
- Linux targets and Linux inspection commands only.
- Tailscale SSH is required. Traditional SSH keys are not supported.
- Systemd is required for service status and service logs.
- One embedded Tailscale identity and one tailnet per process.
- Tailnet policy and machine-tag changes remain explicit admin-console steps.
- No built-in OAuth authorization server; use Secure MCP Tunnel or an
  authenticating reverse proxy.
- The Tailscale Admin API is not needed for v0.1. Inventory comes from the
  embedded node's netmap/LocalAPI, avoiding broad administrative credentials.

See [`docs/architecture.md`](docs/architecture.md) for component details and the
planned path to separately approved write actions.

## License

MIT
