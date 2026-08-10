# ChatGPT Plugin for Tailscale

**TetherBound Community — Give AI access. Keep it bounded.**

[![CI](https://github.com/fijimunkii/chatgpt-plugin-tailscale/actions/workflows/ci.yml/badge.svg)](https://github.com/fijimunkii/chatgpt-plugin-tailscale/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

`chatgpt-plugin-tailscale` is the open-source, self-hosted TetherBound Community project: an MCP gateway that lets ChatGPT, Codex, and other MCP clients reach machines on your Tailscale network without exposing a new public management port or distributing reusable SSH private keys.

The repository name is intentionally literal and discoverable. **TetherBound** is the product and security model; **chatgpt-plugin-tailscale** is the first-class self-hosted distribution.

> **Status:** pre-1.0 / design-partner preview. Linux targets over Tailscale SSH are implemented. Bounded mode has been exercised end to end; Operator mode is experimental. Windows and macOS target support are roadmap items.

TetherBound is an independent community project and is not affiliated with or endorsed by Tailscale Inc. or OpenAI. Tailscale is a trademark of Tailscale Inc.

## Why this exists

Giving an AI assistant a raw SSH key is easy. Deciding what blast radius you actually want is harder.

TetherBound keeps the private-network and machine-identity layer in Tailscale, then gives you two useful access shapes:

| Mode | What the AI gets | Default |
| --- | --- | --- |
| **Bounded** | Seven typed, read-only infrastructure tools with local allowlists and hard limits | Yes |
| **Operator** | Arbitrary shell commands on explicitly admitted hosts as a dedicated non-root OS account, under a time-bounded local lease | No |

The product principle is simple:

> **You choose the boundary. TetherBound keeps the AI inside it.**

## Architecture

```text
ChatGPT / Codex / MCP client
            |
            | MCP
            v
      TetherBound gateway
      host admission + policy
            |
            | Tailscale SSH
            v
        your tailnet
            |
            v
   explicitly admitted Linux hosts
```

For ChatGPT, a private/local MCP server can be connected through **Secure MCP Tunnel** rather than exposing the MCP endpoint publicly. In current ChatGPT terminology this is a custom MCP app; the historical repository name keeps “plugin” for discoverability.

## What works today

The gateway embeds its own tagged Tailscale node with `tsnet`, discovers peers from local tailnet state, verifies Tailscale-distributed SSH host keys, and connects with Tailscale SSH as a dedicated OS account.

### Bounded mode

The default MCP surface contains:

| Tool | Behavior |
| --- | --- |
| `list_hosts` | Lists only peers admitted by `TS_ALLOWED_TAGS` or `TS_ALLOWED_HOSTS`. |
| `inspect_host` | Returns tailnet metadata and readiness for one admitted peer. |
| `system_summary` | Reads kernel, uptime, load, memory, swap, and boot ID. |
| `disk_usage` | Reads bounded filesystem usage. |
| `service_status` | Reads selected properties for an exactly allowlisted systemd service. |
| `service_logs` | Reads bounded journal output for an exactly allowlisted service. |
| `process_list` | Reads a bounded process snapshot; optional filtering happens locally. |

Caller text does not become shell text in Bounded mode.

### Operator mode

For users who explicitly want broad autonomy on a machine, Operator mode adds:

```text
operator_exec(host, command, cwd?)
```

It is **off by default** and must be enabled locally for a time-bounded gateway session. It uses the same admitted-host catalog, Tailscale SSH path, strict host-key verification, OS-account boundary, command timeout, and output limit as the bounded tools.

TetherBound does **not** add sudo/root privileges. If the configured account already has powerful group memberships such as Docker access, treat that as elevated access.

Read [`docs/operator-mode.md`](docs/operator-mode.md) before enabling it.

## Quick start

### Requirements

You need:

- a Tailscale tailnet you can configure;
- at least one Linux target running Tailscale;
- Go as declared in [`go.mod`](go.mod);
- the OpenAI `tunnel-client` if connecting a private/local gateway to ChatGPT through Secure MCP Tunnel;
- a ChatGPT configuration that supports custom MCP apps for the capabilities you want to test.

Clone the repository and run the rerunnable bootstrap:

```bash
git clone https://github.com/fijimunkii/chatgpt-plugin-tailscale.git
cd chatgpt-plugin-tailscale
./bootstrap
```

The wizard walks through:

1. the minimal Tailscale tags and grants;
2. a dedicated `chatgpt` account on the Linux target;
3. enabling Tailscale SSH on that target;
4. enrolling the embedded gateway identity;
5. configuring Secure MCP Tunnel for ChatGPT;
6. validating the local setup.

It preserves existing configuration and is safe to rerun.

Run diagnostics at any time:

```bash
./bootstrap --check
```

### Start in Bounded mode

```bash
make up
```

### Start an Operator session

```bash
make operator-up
```

The default Operator lease is one hour. To use a shorter local lease:

```bash
OPERATOR_LEASE_DURATION=30m make operator-up
```

If another gateway is already running, `make operator-up` fails closed instead of silently reusing a bounded process.

The local services are:

```text
MCP gateway:       http://127.0.0.1:8080/mcp
Gateway readiness: http://127.0.0.1:8080/readyz
Tunnel readiness:  http://127.0.0.1:8081/readyz
Tunnel admin UI:   http://127.0.0.1:8081/ui
```

See [`docs/bootstrap.md`](docs/bootstrap.md) for the complete setup and troubleshooting guide.

## Tailscale policy

The bootstrap displays [`docs/tailscale-policy.hujson`](docs/tailscale-policy.hujson). Merge the relevant entries into your existing policy rather than replacing unrelated rules.

The basic model is:

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

Tailscale policy is additive. Existing broad grants can widen the effective network boundary, so review the rest of the tailnet policy too.

## Security model

The gateway deliberately layers controls rather than treating the MCP client as the security boundary:

1. the gateway has its own Tailscale identity;
2. Tailscale controls which targets it can reach over TCP/22 and SSH;
3. the gateway independently filters peers by local tag/host admission;
4. shared nodes are excluded by default;
5. target SSH host keys must match keys distributed through Tailscale;
6. the remote login uses a dedicated OS account (`chatgpt` by default);
7. Bounded tools use fixed command builders and explicit service allowlists;
8. Operator mode exists only after explicit local opt-in and expires;
9. command execution and captured output are bounded;
10. raw command text and remote output are not written to gateway logs by default.

Remote logs, files, process arguments, repository content, and command output are **untrusted model input**. Operator mode materially increases the consequence of prompt injection or model mistakes. Start with development or disposable hosts and keep root credentials out of the delegated account.

Read [`docs/threat-model.md`](docs/threat-model.md) and [`SECURITY.md`](SECURITY.md) before exposing sensitive infrastructure.

## Self-hosting is the primary Community product

TetherBound Community is designed to remain fully useful without TetherBound Cloud. The code that reaches, authenticates to, inspects, or operates customer machines belongs in this public repository.

A future hosted TetherBound Cloud may provide convenience around authentication, connector lifecycle, team policy, relay infrastructure, audit search, and billing. Cloud must not be required to run the documented Community features and may never widen a local Edge policy denial.

See [`docs/repository-boundary.md`](docs/repository-boundary.md).

## Current platform support

| Component | Linux | macOS | Windows |
| --- | --- | --- | --- |
| TetherBound gateway / Edge | Primary tested path | Development-friendly / not yet packaged | Roadmap |
| Agentless target via Tailscale SSH | **Working** | Not a supported target path yet | Tailscale SSH server unavailable |
| TetherBound Agent target | Roadmap | Roadmap | Next major target milestone |

Windows is agent-first: the planned Agent will initiate an outbound tailnet session and expose normalized operations without requiring reusable Windows passwords, WinRM credentials, or a generic PowerShell endpoint.

## Configuration

`./bootstrap` creates a gitignored `.env` from [`config/example.env`](config/example.env) and stores tunnel configuration separately in `.env.tunnel`, both mode `0600` where supported.

Important controls include:

| Variable | Default | Purpose |
| --- | --- | --- |
| `TS_ALLOWED_TAGS` | `tag:chatgpt-managed` | Target tags admitted by local policy. |
| `TS_ALLOWED_HOSTS` | empty | Optional exact host IDs/names/Tailscale IPs. |
| `TS_ALLOW_SHARED_NODES` | `false` | Whether shared-to nodes may be admitted. |
| `SSH_USER` | `chatgpt` | Remote OS account. |
| `SSH_ALLOWED_SERVICES` | empty | Exact service allowlist for Bounded service tools. |
| `SSH_COMMAND_TIMEOUT` | `15s` code default; `2m` example config | Command deadline. |
| `SSH_MAX_OUTPUT_BYTES` | `262144` | Combined stdout/stderr capture limit. |
| `OPERATOR_MODE` | `false` | Registers `operator_exec` when explicitly enabled locally. |
| `OPERATOR_LEASE_DURATION` | `1h` | Operator lease from gateway process start; 5m–24h. |
| `MCP_LISTEN_ADDR` | `127.0.0.1:8080` | MCP listener; loopback by default. |

At least one target tag or host selector is always required. Ambiguous short host selectors fail closed.

## Development

```bash
make check
```

This validates bootstrap scripts, Go formatting, tests, vet, and the production gateway build.

Useful targets:

```bash
make bootstrap
make bootstrap-check
make up
make operator-up
make build
make docker
```

Contributions are welcome. Read [`CONTRIBUTING.md`](CONTRIBUTING.md) first because changes to remote execution are treated as security-boundary changes.

## Project docs

- [`docs/bootstrap.md`](docs/bootstrap.md) — complete self-host setup
- [`docs/operator-mode.md`](docs/operator-mode.md) — broad-access profile and risks
- [`docs/threat-model.md`](docs/threat-model.md) — security assumptions and mitigations
- [`docs/architecture.md`](docs/architecture.md) — implemented request path
- [`docs/tool-contracts-v1.md`](docs/tool-contracts-v1.md) — normalized tool direction
- [`docs/cross-platform-spec.md`](docs/cross-platform-spec.md) — Windows/macOS/Agent architecture
- [`docs/roadmap.md`](docs/roadmap.md) — implementation sequence
- [`docs/repository-boundary.md`](docs/repository-boundary.md) — open Community / optional hosted boundary

Product site: **https://tetherbound.dev**

## Support and security

General setup problems and feature ideas belong in GitHub Issues after removing secrets and private infrastructure details. See [`SUPPORT.md`](SUPPORT.md).

Do **not** file a public issue for a vulnerability. Follow [`SECURITY.md`](SECURITY.md).

## License

MIT — see [`LICENSE`](LICENSE).
