# Architecture

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

## Request path

1. The HTTP layer accepts MCP Streamable HTTP at `/mcp`, caps request bodies, rejects browser-origin requests, and optionally verifies a static bearer token.
2. The MCP SDK validates typed tool arguments and serializes typed structured output.
3. The tool handler resolves the requested peer from a fresh LocalAPI status snapshot.
4. The catalog removes peers that do not match an allowed tag or exact host selector. Shared nodes are removed by default, and an explicit host selector matching multiple peers fails closed.
5. The target must be online, unexpired, Linux-based, have a Tailscale IP, and have SSH host keys present in the netmap.
6. A command builder returns an opaque `remote.Command` whose shell field cannot be populated outside that package. Inputs can select only a host, an exact allowlisted service, a bounded integer, or a local literal process filter.
7. `tsnet.Server.Dial` opens TCP/22 over the tailnet.
8. `x/crypto/ssh` performs the SSH handshake with authentication method `none`, as expected by Tailscale SSH.
9. The SSH host-key callback accepts only keys distributed in the peer's Tailscale status. There is no insecure callback and no trust-on-first-use path.
10. stdout and stderr share one bounded capture budget. The session and dial have independent deadlines.
11. The MCP response marks remote content as untrusted and includes an operation ID that is also recorded in structured audit logs.

## Why the Admin API is not in v0.1

The gateway only needs its own control-plane view of peers and their Tailscale SSH host keys. `tsnet` plus LocalAPI status provides that data without giving the process broad tailnet-administration scopes. The Admin API remains useful for later workflows such as onboarding, tag lifecycle, policy validation, and fleet-level health, but it is not part of the command path.

## Planned write path

Write operations should not be added as a generic shell. The intended design is:

1. A narrow typed action such as `restart_service` or `deploy_release`.
2. A preparation call that resolves the target, validates policy, computes the exact immutable action, and returns a short-lived signed approval token.
3. A separate execution call that accepts only that token.
4. Tool metadata marked as write/destructive where appropriate.
5. Explicit ChatGPT confirmation, stronger OS authorization, and session recording.
6. Replay prevention and an append-only audit sink.

Arbitrary shell access is intentionally outside the initial implementation.
