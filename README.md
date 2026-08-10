# TetherBound

**Give AI access. Keep it bounded.**

[![CI](https://github.com/fijimunkii/chatgpt-plugin-tailscale/actions/workflows/ci.yml/badge.svg)](https://github.com/fijimunkii/chatgpt-plugin-tailscale/actions/workflows/ci.yml)

TetherBound is a security-first MCP gateway for giving AI assistants controlled
access to explicitly allowed private infrastructure. The implemented v0.1 path
supports Linux targets over
[Tailscale SSH](https://tailscale.com/docs/features/tailscale-ssh); the committed
roadmap extends the same model to Windows and macOS through an open-source host
agent.

TetherBound now supports two access shapes:

- **Bounded mode (default):** seven fixed, typed, read-only operations with exact
  allowlists and hard result limits.
- **Operator mode (experimental):** an explicit, locally started lease that gives
  the AI arbitrary POSIX shell access on admitted Linux hosts as the configured
  OS account, without TetherBound adding root or sudo privileges.

The boundary therefore does not have to mean approving every individual
operation. In Operator mode it moves outward to **the admitted host + OS account
+ local lease**. Both modes still use the embedded Tailscale identity, local
host admission, and strict Tailscale-distributed SSH host-key verification.
TetherBound does **not** store SSH private keys.

TetherBound is an independent community project and is not affiliated with or
endorsed by Tailscale or OpenAI.

> [!IMPORTANT]
> This is an initial implementation intended for private testing. Remote
> hostnames, logs, process arguments, files, repository content, and command
> output are untrusted data and can contain prompt-injection text. Operator mode
> materially increases the consequences of model mistakes because it is a real
> shell. Keep its host/account blast radius deliberate and review the threat
> model before deployment.

## Product validation site

The repository includes an incubation copy of the dependency-free validation
site in [`site/`](site/). The public deployment source of truth now lives in the
separate `tetherbound-site` repository and is published at
[`tetherbound.dev`](https://tetherbound.dev/).

The site explains the product in non-implementation language, separates shipped
features from roadmap concepts, and collects structured Cloud interest without
asking for infrastructure secrets.

Preview the incubation copy locally:

```bash
python3 -m http.server 4173 --directory site
```

Then open `http://127.0.0.1:4173/`.

Pages:

- [`Landing page`](site/index.html) — **Give AI access. Keep it bounded.**
- [`Security`](site/security.html) — customer-facing trust boundary.
- [`Community`](site/community.html) — current open-source / DIY direction.
- [`Cloud`](site/cloud.html) — proposed hosted edition and validation CTA.
- [`Docs portal`](site/docs.html) — implementation and roadmap docs.

Product-learning documents:

- [`Product positioning`](docs/product-positioning.md)
- [`Validation plan`](docs/validation-plan.md)
- [`Product roadmap`](docs/roadmap.md)
- [`Cross-platform architecture and hosted product spec`](docs/cross-platform-spec.md)
- [`Normalized MCP tool contracts v1`](docs/tool-contracts-v1.md)
- [`Operator mode`](docs/operator-mode.md)
- [`Threat model`](docs/threat-model.md)

The runtime repository/binary/tag names are intentionally not migrated in this
branding change. Those identifiers should move only through a dedicated
backward-compatible migration.

## Quick start

The supported setup path is the interactive, rerunnable bootstrap wizard:

```bash
./bootstrap
```

It walks through all five systems involved in the current Linux setup:

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

### Bounded mode

After bootstrap, launch the default read-only gateway and tunnel:

```bash
make up
```

The launcher reuses already healthy bounded processes, waits for both readiness
checks, and stops only processes it started.

### Operator mode — experimental

For a design-partner session that deliberately grants arbitrary shell access to
the admitted Linux machines as `SSH_USER`:

```bash
make operator-up
```

The default Operator lease is one hour. To shorten it:

```bash
OPERATOR_LEASE_DURATION=30m make operator-up
```

`make operator-up` fails closed when another gateway is already running. Stop
the existing gateway first so the launcher cannot silently reuse a bounded
process. Operator mode is intentionally not persisted in `config/example.env`.
After launch, rescan the MCP tools in ChatGPT so `operator_exec` appears.

The local endpoints are:

```text
MCP gateway:       http://127.0.0.1:8080/mcp
Gateway readiness: http://127.0.0.1:8080/readyz
Tunnel readiness:  http://127.0.0.1:8081/readyz
Tunnel admin UI:   http://127.0.0.1:8081/ui
```

See the complete [`bootstrap guide`](docs/bootstrap.md) for the exact admin-console
steps, rerun guarantees, non-interactive mode, target-only usage, macOS
Gatekeeper guidance, and troubleshooting. Read [`docs/operator-mode.md`](docs/operator-mode.md)
before enabling broad access.

## Cross-platform and hosted direction

Tailscale SSH can be used *from* Windows, but its server component does not run
on Windows. The planned Windows path therefore does not ask users to configure
a Windows password, reusable SSH key, WinRM credential, or unrestricted
PowerShell endpoint.

Instead, a small open-source TetherBound Agent will:

- run as a Windows Service;
- use the Windows machine's existing Tailscale connection;
- initiate an outbound session to the private connector;
- authenticate with both Tailscale source identity and mTLS;
- implement native typed reads for system facts, disks, processes, services,
  and Windows Event Logs;
- return the same normalized bounded MCP contracts as Linux.

The same agent model will support macOS and optionally Linux. Agentless
Tailscale SSH remains the zero-install Linux path. The gateway/TetherBound Edge
itself is planned for macOS, Linux, and Windows.

The proposed TetherBound Cloud edition provides a hosted authenticated MCP
endpoint and outbound customer edge relay. The customer connector keeps the
Tailscale identity and local policy at the edge; the cloud does not directly
dial the tailnet. Cloud is currently a product-validation concept, not a shipped
service.

The immediate validation milestone is a real external Linux Operator-mode
install. The next implementation milestone after that remains a Windows host
agent tested beside the current Linux path.

## Included tools

| Tool | Current behavior |
| --- | --- |
| `list_hosts` | Lists only peers admitted by `TS_ALLOWED_TAGS` or `TS_ALLOWED_HOSTS`. |
| `inspect_host` | Returns tailnet metadata and readiness for one allowed peer. |
| `system_summary` | Fixed Linux read of kernel, uptime, load, memory, swap, and boot ID. |
| `disk_usage` | Fixed `df` inspection with bounded output. |
| `service_status` | Fixed `systemctl show` for an exact service allowlist. |
| `service_logs` | Fixed, line-bounded `journalctl` read for an exact service allowlist. |
| `process_list` | Fixed `ps` read; optional filtering happens locally in the gateway. |
| `operator_exec` | **Experimental / opt-in.** Arbitrary POSIX shell text on one admitted Linux host during an active local Operator lease. |

The first seven tools are the default Bounded surface and are marked read-only
and idempotent. `operator_exec` is registered only when Operator mode is locally
enabled and is marked writable, non-idempotent, destructive, and open-world.
MCP annotations are descriptive metadata; the actual boundary is enforced by
TetherBound and Tailscale.

## Security model

The implemented gateway applies several independent boundaries:

1. Its own Tailscale identity should be `tag:chatgpt-ssh`.
2. Tailscale policy grants that tag TCP/22 and Tailscale SSH access only to
   `tag:chatgpt-managed` targets.
3. The gateway independently filters the netmap with `TS_ALLOWED_TAGS` and
   `TS_ALLOWED_HOSTS`.
4. Shared nodes are excluded unless `TS_ALLOW_SHARED_NODES=true`.
5. Execution requires an online Linux peer with a Tailscale IP and
   control-plane-distributed SSH host keys.
6. The remote login is a dedicated non-root account, `chatgpt` by default.
7. In Bounded mode, caller text cannot become a remote command; the fixed command
   package constructs the allowed operations.
8. In Operator mode, arbitrary shell text is the deliberate capability, but it
   remains restricted to an active locally started lease and the same admitted
   host + SSH-account boundary. TetherBound does not add sudo or root.
9. Timeouts and a combined stdout/stderr byte limit bound each operation in both
   modes.
10. Gateway logs contain operation metadata but do not retain raw command text
    or raw command output by default.

The planned host-agent path adds a second authenticated transport rather than
weakening these controls: agents dial outbound, Tailscale identity and mTLS are
both required, and local policy remains authoritative.

The bootstrap keeps local credentials in gitignored `.env` and `.env.tunnel`
files with mode `0600`. For a shared or production deployment, replace those
files with an external secrets manager.

Read [`docs/threat-model.md`](docs/threat-model.md) and
[`docs/operator-mode.md`](docs/operator-mode.md) before enabling access to
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

The future host-agent session adds a separately scoped reverse-direction grant
from managed targets to the connector's private agent-session port. It does not
replace the current TCP/22 rule for Linux.

## Target preparation

The current Linux target helper is independently usable and idempotent:

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

Windows and macOS target setup described in the roadmap is not implemented in
v0.1 yet.

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
| `SSH_USER` | `chatgpt` | Remote OS account used by both access modes. |
| `SSH_ALLOWED_SERVICES` | empty | Exact systemd unit allowlist for Bounded service tools; empty disables those tools. |
| `SSH_DIAL_TIMEOUT` | `10s` | Tailnet TCP dial deadline. |
| `SSH_COMMAND_TIMEOUT` | `15s` code fallback; `2m` in `config/example.env` | End-to-end command deadline. |
| `SSH_MAX_OUTPUT_BYTES` | `262144` | Combined stdout/stderr capture limit. |
| `SSH_MAX_LOG_LINES` | `200` | Maximum journal lines per call. |
| `SSH_MAX_PROCESSES` | `100` | Maximum returned process rows. |
| `OPERATOR_MODE` | `false` | Registers `operator_exec` when explicitly enabled locally. Intentionally not persisted by the example config. |
| `OPERATOR_LEASE_DURATION` | `1h` | Operator lease from process start; accepted range 5m–24h. |
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

The bootstrap launcher is recommended. To run only the gateway manually in
Bounded mode:

```bash
set -a
. ./.env
set +a

go run ./cmd/gateway
```

For a manual Operator-mode process:

```bash
set -a
. ./.env
set +a
OPERATOR_MODE=true go run ./cmd/gateway
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
make up               # bounded gateway + tunnel-client
make operator-up      # experimental Operator lease + tunnel-client
```

The test suite covers configuration fail-closed behavior, target filtering and
ambiguous allowlist rejection, bounded command construction, Operator command
construction and lease behavior, local process filtering, bearer authentication,
strict host-key verification, combined output limits, bootstrap reruns, target
checks, and tunnel profile reuse.

## Current limitations

- Operator mode is experimental and Linux/Tailscale-SSH-only today.
- Operator leases are process-relative in v0.1; a deliberately restarted process
  with Operator mode enabled receives a fresh lease. Do not persist it in a
  service manager for the design-partner preview.
- TetherBound does not add root/Administrator access. Existing privileged group
  membership (for example Docker daemon access) is part of the delegated blast
  radius and must be evaluated separately.
- Operator mode has no PTY, interactive stdin, or persistent terminal session;
  each `operator_exec` call is a bounded command invocation.
- The implemented target adapter is Linux-only today.
- Tailscale SSH is required for the implemented target path.
- Systemd is required for current bounded service status and service logs.
- One embedded Tailscale identity and one tailnet per process.
- Tailnet policy and machine-tag changes remain explicit admin-console steps.
- The current bootstrap source of truth is Bash; a cross-platform Go setup CLI
  is planned.
- No built-in OAuth authorization server; use Secure MCP Tunnel or an
  authenticating reverse proxy for DIY use.
- The Tailscale Admin API is not needed for v0.1. Inventory comes from the
  embedded node's netmap/LocalAPI, avoiding broad administrative credentials.

See [`docs/architecture.md`](docs/architecture.md) for the implemented request
path, [`docs/operator-mode.md`](docs/operator-mode.md) for the broad-access
preview, and [`docs/cross-platform-spec.md`](docs/cross-platform-spec.md) for the
planned cross-platform and hosted architecture.

## License

MIT