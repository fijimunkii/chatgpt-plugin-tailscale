# Threat model

## Assets

- Availability and confidentiality of hosts in the tailnet.
- Tailscale node credentials and persistent `tsnet` state.
- Service logs, process arguments, hostnames, system metadata, and Operator-mode command output returned to ChatGPT.
- Integrity of the MCP gateway and its deployment configuration.
- Integrity and confidentiality of files, processes, repositories, credentials, and network access available to an Operator-mode OS account.

## Trust boundaries

- ChatGPT and the MCP transport authenticate at Secure MCP Tunnel or an external proxy boundary; a built-in static bearer check is optional.
- Tailscale authenticates the gateway node and target nodes and enforces network and SSH policy.
- The gateway independently enforces the admitted target catalog.
- Bounded mode additionally enforces typed tools, exact service allowlists, and fixed command construction.
- Operator mode deliberately broadens the operation boundary to arbitrary shell access while retaining host admission, Tailscale SSH verification, the configured OS account, execution timeout, output limit, and a locally enabled lease.
- The target OS account controls local data visibility and write capability.
- Remote output is never trusted as an instruction source.

## Primary threats and mitigations

### Prompt injection in remote data

Logs, process arguments, DNS names, unit descriptions, files, repository content, web responses, and other remote data can contain text that asks the model to call additional tools, reveal data, or take destructive actions.

Mitigations:

- Every bounded inspection tool description and structured result labels remote content as untrusted.
- Bounded mode exposes no arbitrary command or generic file-read capability.
- Operator mode is explicitly treated as a higher-risk delegation because a model can act on malicious instructions using the delegated OS account.
- Host admission and the OS-account boundary remain deterministic even if the model mishandles untrusted content.
- Operators should avoid placing human SSH keys, broad cloud credentials, production secrets, or other unnecessary authority in an Operator-mode account.
- Root/Administrator elevation is not added by TetherBound Operator mode.

### Command injection

#### Bounded mode

Mitigations:

- MCP callers never provide a command string to bounded tools.
- Bounded commands are opaque values whose shell text can be constructed only inside the fixed command package.
- Service values require conservative syntax and an exact configuration allowlist.
- Log counts and process limits are parsed integers with hard bounds.
- Process filtering is literal and occurs after the remote command completes.
- Shell quoting is retained even for validated service names.

#### Operator mode

Arbitrary shell interpretation is the intended capability, not an injection bug. The relevant security question becomes whether untrusted data can escape the delegated machine/account boundary.

Mitigations:

- `operator_exec` is registered only when Operator mode was enabled locally at gateway startup.
- Operator mode is off by default and is not persisted in `config/example.env`.
- The tool requires the same exact admitted-host resolution as bounded tools.
- An optional working directory is shell-quoted before use.
- Commands execute as the configured SSH user; TetherBound does not add sudo or root.
- Execution time and captured output remain bounded.
- MCP annotations mark the tool writable, non-idempotent, destructive, and open-world.

### Lateral movement

Mitigations:

- The gateway has its own tag-based Tailscale identity.
- The sample grant permits only TCP/22 from that identity to one target tag.
- Tagged sources can SSH only to tagged targets under Tailscale's policy restrictions.
- The local catalog repeats the target allowlist check, excludes shared nodes by default, and rejects ambiguous explicit host selectors.
- The default remote account is non-root and TetherBound does not install a sudo rule.
- Operator mode can use whatever network access and local groups the delegated OS account already has. Membership in privileged groups such as `docker` should be treated as effectively elevated access.

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
- Operator commands share the same command timeout and output-byte boundary.
- The initial server does not retain interactive PTYs or persistent shell sessions.

### Credential disclosure

Mitigations:

- The gateway does not store SSH private keys.
- Auth keys, OAuth secrets, bearer tokens, raw command text, and command output are not written to gateway logs by default.
- `TS_STATE_DIR` should be mode 0700 and backed by a protected persistent volume.
- Bounded tools do not accept environment injection.
- Operator mode can intentionally read credentials visible to its OS account; operators must treat that account's credential set as part of the delegated blast radius.

### Operator lease renewal

The initial Operator lease is process-scoped. A new gateway process deliberately started with `OPERATOR_MODE=true` receives a fresh lease.

Mitigations for the design-partner preview:

- `OPERATOR_MODE` is not included as a persisted default in `config/example.env`.
- Operators should enable it only for an explicit local launch, for example `OPERATOR_MODE=true make up`.
- Do not configure a service manager to persistently restart the gateway with Operator mode enabled.
- A later release should replace process-relative leases with an absolute locally issued lease so automatic restarts cannot extend delegated access.

### MCP endpoint exposure

Mitigations:

- The default listener is loopback-only.
- A non-loopback listener requires a sufficiently long bearer token unless the operator explicitly overrides the fail-closed check.
- Browser-origin requests are rejected.
- The MCP SDK's localhost Host-header defense remains enabled unless explicitly disabled for a trusted tunnel.
- Public deployment should use an OAuth-capable, authenticating edge rather than relying solely on a shared static token.

## Out of scope for the design-partner preview

- Compromise of ChatGPT, Tailscale control infrastructure, the gateway host kernel, or a target host.
- Making arbitrary shell execution intrinsically safe; Operator mode is intentionally powerful and reduces safety to the delegated host/account/network boundary.
- Automatic root/Administrator delegation.
- Multi-user authorization inside one gateway process.
- Guaranteed redaction of secrets already visible to the remote OS account.
- Persistent interactive PTYs, stdin streaming, or terminal multiplexing.
