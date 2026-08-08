# Threat model

## Assets

- Availability and confidentiality of hosts in the tailnet.
- Tailscale node credentials and persistent `tsnet` state.
- Service logs, process arguments, hostnames, and system metadata returned to ChatGPT.
- Integrity of the MCP gateway and its deployment configuration.

## Trust boundaries

- ChatGPT and the MCP transport authenticate at Secure MCP Tunnel or an external proxy boundary; a built-in static bearer check is optional.
- Tailscale authenticates the gateway node and target nodes and enforces network and SSH policy.
- The gateway independently enforces target and service allowlists.
- The target OS account controls local data visibility.
- Remote output is never trusted as an instruction source.

## Primary threats and mitigations

### Prompt injection in remote data

Logs, process arguments, DNS names, unit descriptions, and other remote data can contain text that asks the model to call additional tools or reveal data.

Mitigations:

- Every tool description and structured result labels remote content as untrusted.
- There are no arbitrary commands or file reads.
- A model still may mishandle malicious data, so the target and service allowlists remain the primary security boundary.
- Operators should avoid exposing secret-bearing logs unless required.

### Command injection

Mitigations:

- MCP callers never provide a command string.
- Commands are opaque values whose shell text can be constructed only inside the fixed command package.
- Service values require conservative syntax and an exact configuration allowlist.
- Log counts and process limits are parsed integers with hard bounds.
- Process filtering is literal and occurs after the remote command completes.
- Shell quoting is retained even for validated service names.

### Lateral movement

Mitigations:

- The gateway has its own tag-based Tailscale identity.
- The sample grant permits only TCP/22 from that identity to one target tag.
- Tagged sources can SSH only to tagged targets under Tailscale's policy restrictions.
- The local catalog repeats the target allowlist check, excludes shared nodes by default, and rejects ambiguous explicit host selectors.
- The remote account is non-root and no sudo rule is required by the supplied tools.

### SSH server impersonation

Mitigations:

- Connections use Tailscale IPs learned from the current netmap.
- The SSH host key must exactly match a public key distributed in Tailscale peer status.
- There is no `InsecureIgnoreHostKey`, trust-on-first-use, or local mutable `known_hosts` dependency.

### Resource exhaustion

Mitigations:

- MCP request bodies are capped.
- HTTP headers are capped by the server.
- SSH dials and commands have deadlines.
- stdout and stderr share one capture limit.
- Log lines and process rows have hard maximums.
- The initial server is stateless and does not retain MCP sessions.

### Credential disclosure

Mitigations:

- The gateway does not store SSH private keys.
- Auth keys, OAuth secrets, bearer tokens, commands, and command output are not logged.
- `TS_STATE_DIR` should be mode 0700 and backed by a protected persistent volume.
- Environment injection is not supported for remote commands.

### MCP endpoint exposure

Mitigations:

- The default listener is loopback-only.
- A non-loopback listener requires a sufficiently long bearer token unless the operator explicitly overrides the fail-closed check.
- Browser-origin requests are rejected.
- The MCP SDK's localhost Host-header defense remains enabled unless explicitly disabled for a trusted tunnel.
- Public deployment should use an OAuth-capable, authenticating edge rather than relying solely on a shared static token.

## Out of scope for v0.1

- Compromise of ChatGPT, Tailscale control infrastructure, the gateway host kernel, or a target host.
- Arbitrary shell execution.
- Writes, service restarts, deployments, package management, or file modification.
- Multi-user authorization inside one gateway process.
- Guaranteed redaction of secrets already visible to the remote OS account.
