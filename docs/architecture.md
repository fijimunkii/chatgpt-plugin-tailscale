# Architecture

This document describes the **implemented v0.1 Linux path** in PR #1. The
cross-platform connector, Windows/macOS host-agent design, hosted control plane,
and migration plan are specified in
[`cross-platform-spec.md`](cross-platform-spec.md). The staged delivery plan is
in [`roadmap.md`](roadmap.md), and normalized MCP schemas are in
[`tool-contracts-v1.md`](tool-contracts-v1.md).

## Implemented v0.1 topology

```text
ChatGPT custom app
        |
        | MCP Streamable HTTP
        | (Secure MCP Tunnel or authenticating reverse proxy)
        v
+------------------------------+
| Tailscale SSH Inspector      |
|                              |
|  MCP tool registry           |
|        |                     |
|  fixed command builders      |
|        |                     |
|  local target allowlist      |
|        |                     |
|  strict SSH runner           |
|        |                     |
|  embedded tsnet node         |
+--------|---------------------+
         |
         | WireGuard + Tailscale policy + Tailscale SSH
         v
  tag:chatgpt-managed Linux hosts
```

## Implemented request path

1. The HTTP layer accepts MCP Streamable HTTP at `/mcp`, caps request bodies,
   rejects browser-origin requests, and optionally verifies a static bearer
   token.
2. The MCP SDK validates typed tool arguments and serializes typed structured
   output.
3. The tool handler resolves the requested peer from a fresh LocalAPI status
   snapshot.
4. The catalog removes peers that do not match an allowed tag or exact host
   selector. Shared nodes are removed by default, and an explicit host selector
   matching multiple peers fails closed.
5. The target must be online, unexpired, Linux-based, have a Tailscale IP, and
   have SSH host keys present in the netmap.
6. A command builder returns an opaque `remote.Command` whose shell field cannot
   be populated outside that package. Inputs can select only a host, an exact
   allowlisted service, a bounded integer, or a local literal process filter.
7. `tsnet.Server.Dial` opens TCP/22 over the tailnet.
8. `x/crypto/ssh` performs the SSH handshake with authentication method `none`,
   as expected by Tailscale SSH.
9. The SSH host-key callback accepts only keys distributed in the peer's
   Tailscale status. There is no insecure callback and no trust-on-first-use
   path.
10. stdout and stderr share one bounded capture budget. The session and dial
    have independent deadlines.
11. The MCP response marks remote content as untrusted and structured audit logs
    record operation metadata without command output or credentials.

## Why Windows needs a different path

Tailscale SSH’s server component does not run on Windows. Windows can originate
Tailscale SSH connections, but it cannot be the destination for this v0.1
adapter. Requiring Windows OpenSSH, a reusable key, or WinRM would weaken the
current default security model and create substantial setup variance.

The planned Windows path is therefore an open-source host agent that:

- runs as a Windows Service;
- uses the machine’s existing Tailscale connection;
- initiates an outbound session to the connector’s tailnet-only listener;
- is authenticated by both Tailscale source identity and mTLS;
- executes native, typed, read-only Windows adapters;
- requires no Windows login password, SSH key, or raw PowerShell command.

The same agent model becomes the default macOS target path and an optional Linux
path, while agentless Tailscale SSH remains available for Linux.

## Planned cross-platform community topology

```text
ChatGPT custom app
        |
        | OpenAI Secure MCP Tunnel
        v
Community gateway / edge connector
        ^                    |
        | outbound agent     | Tailscale SSH
        | session over       v
        | Tailscale       Linux hosts
        |
Windows, macOS, and Linux host agents
```

The host agent initiates the connection. It does not expose an MCP endpoint or a
public/LAN management listener.

## Planned hosted topology

```text
ChatGPT / Codex
      |
      | public HTTPS MCP + OAuth 2.1
      v
Hosted MCP + policy + audit metadata
      |
      | regional relay
      v
outbound customer edge connector
      |
      +-- Tailscale SSH -> Linux
      |
      +-- agent sessions -> Windows/macOS/Linux
```

The hosted cloud never receives a Tailscale auth key and cannot dial the
customer tailnet. The edge connector remains the local enforcement point. Raw
host output is not retained by default.

## Why the Admin API is not in v0.1

The gateway only needs its own control-plane view of peers and their Tailscale
SSH host keys. `tsnet` plus LocalAPI status provides that data without giving the
process broad tailnet-administration scopes. The Admin API remains useful for
later workflows such as onboarding, tag lifecycle, policy validation, and
fleet-level health, but it is not part of the command path.

The bootstrap intentionally leaves policy merging and machine tagging as
explicit admin-console actions rather than silently taking administrative
control of the tailnet.

## Planned host abstraction

The current Linux catalog and SSH runner will move behind two interfaces:

- `HostTransport`: discovery, identity, reachability, authentication, and request
  delivery;
- `HostAdapter`: typed host facts, disks, processes, services, and logs.

Initial transports:

```text
tailscale_ssh
agent
```

Reserved optional transports:

```text
openssh
winrm
```

Every adapter must pass the same normalized contract and capability tests. No
transport accepts a caller-supplied command string.

## Planned write path

Write operations must not be added as a generic shell. The intended design is:

1. A narrow typed action such as `restart_service` or `deploy_release`.
2. A preparation call that resolves the target, validates policy, computes the
   exact immutable action, and returns a short-lived signed approval token.
3. A separate execution call that accepts only the immutable action reference.
4. Tool metadata marked as write/destructive where appropriate.
5. Explicit ChatGPT confirmation and, where configured, out-of-band
   administrator approval.
6. Local edge revalidation immediately before execution.
7. Replay prevention and append-only audit metadata.

Arbitrary shell access remains intentionally outside the design.

## Primary design references

- [`roadmap.md`](roadmap.md)
- [`cross-platform-spec.md`](cross-platform-spec.md)
- [`tool-contracts-v1.md`](tool-contracts-v1.md)
- [`threat-model.md`](threat-model.md)
- [`bootstrap.md`](bootstrap.md)
