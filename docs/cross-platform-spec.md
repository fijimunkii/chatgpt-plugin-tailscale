# Cross-platform architecture and product specification

Status: proposed architecture  
Scope: community gateway, cross-platform host agent, and future hosted control plane  
Last updated: 2026-08-08

This specification evolves the current Linux/Tailscale SSH proof of concept into
a cross-platform private-infrastructure connector. It is intentionally
implementation-oriented: component boundaries, trust boundaries, protocols,
platform behavior, packaging, migration, tests, and release criteria are
specified far enough for work to be split into issues.

The sequencing is defined in [`roadmap.md`](roadmap.md). MCP inputs and outputs
are defined in [`tool-contracts-v1.md`](tool-contracts-v1.md).

## 1. Problem statement

A user should be able to ask an AI client to inspect explicitly allowed private
machines without:

- exposing SSH, WinRM, MCP, or an agent port to the public internet;
- placing an unrestricted SSH key or Windows password in the AI client;
- granting an arbitrary shell;
- losing the network and host boundaries already enforced by Tailscale;
- receiving platform-specific text that the model must guess how to parse.

The current implementation solves this for Linux targets with Tailscale SSH.
Tailscale SSH’s server component is not available on Windows, and on macOS it is
limited to the open-source `tailscale` + `tailscaled` distribution. Windows and
mainstream macOS targets therefore require a host-agent path.

## 2. Goals

### 2.1 Functional goals

- Run the gateway/edge connector on macOS, Linux, and Windows.
- Inspect Linux, Windows, and macOS targets through the same MCP tools.
- Preserve agentless Tailscale SSH as the preferred zero-install Linux path.
- Add a host agent that initiates an outbound session; no inbound target port is
  required.
- Return normalized, versioned, structured data.
- Detect and advertise per-host capabilities.
- Support both a fully local DIY topology and a hosted-control-plane topology.
- Preserve stable host identity while transport or hostname changes.
- Survive reboot and reconnect unattended where the underlying Tailscale client
  supports it.

### 2.2 Security goals

- No arbitrary command, script, or shell-expression input.
- No reusable Windows password or SSH key in the default Windows path.
- Tailscale identity and local agent identity must both be checked.
- Local edge policy must be authoritative even when a hosted service is used.
- Read-only tools must use the minimum OS access rights required.
- Remote output must be bounded and treated as untrusted.
- Revocation must work independently for users, connectors, agents, and
  certificates.
- Hosted infrastructure must never need direct network reachability into a
  customer tailnet.

### 2.3 Product goals

- DIY users can remain fully self-hosted.
- Hosted users do not need developer mode, a Platform tunnel, or `tunnel-client`.
- The edge connector and host agent remain open source and auditable.
- Public plugin requirements are isolated to the hosted MCP and identity layer.

## 3. Non-goals

The following remain outside the read-only v1 scope:

- interactive terminal or PTY support;
- RDP, VNC, or desktop control;
- generic PowerShell, shell, batch, or script execution;
- unrestricted file reads or writes;
- password management;
- automatic remediation;
- replacing monitoring, EDR, RMM, configuration-management, or backup products;
- unattended edits to Tailscale policy or machine tags.

## 4. Architecture overview

### 4.1 Current agentless Linux path

```text
ChatGPT
   |
   | MCP through OpenAI Secure MCP Tunnel
   v
Community gateway
   |
   | embedded tsnet identity + Tailscale policy + TCP/22
   v
Tailscale SSH server on Linux
   |
   v
fixed read-only Linux adapter
```

This remains supported.

### 4.2 Community cross-platform agent path

```text
ChatGPT
   |
   | MCP through OpenAI Secure MCP Tunnel
   v
Community gateway / private connector
   ^
   | long-lived, outbound agent session over the tailnet
   | Tailscale source identity + mTLS
   |
Host agent on Windows, macOS, or Linux
   |
   v
native typed host adapter
```

The host agent dials the connector. It does not listen on a LAN or public
interface. The connector’s embedded `tsnet` node exposes the private agent
session listener only inside the tailnet.

### 4.3 Hosted path

```text
ChatGPT or Codex
   |
   | public HTTPS MCP + OAuth 2.1
   v
Hosted MCP resource server
   |
   | tenant policy, routing, audit metadata
   v
Regional relay
   ^
   | outbound authenticated connector session
   |
Customer edge connector
   |                     ^
   | Tailscale SSH       | outbound host-agent sessions
   v                     |
Linux target       Windows/macOS/Linux agents
```

The cloud sends work only through an already established outbound connector
session. It receives no Tailscale auth key and cannot dial a tailnet directly.

## 5. Components

### 5.1 MCP gateway

Responsibilities:

- expose the MCP tools;
- authenticate the MCP caller in hosted mode;
- map caller identity to workspace policy;
- resolve stable host IDs;
- select a host transport;
- apply request bounds and tool annotations;
- normalize adapter errors;
- label remote content as untrusted;
- emit audit metadata.

In community mode, the gateway and edge connector are one process. In hosted
mode, the public MCP resource server is separate from the customer edge
connector.

### 5.2 Edge connector

Responsibilities:

- own the embedded `tsnet` identity;
- keep Tailscale node state locally;
- discover and dial agentless Linux targets;
- accept outbound host-agent sessions on a tailnet-only listener;
- verify Tailscale source identity and agent certificate;
- enforce local host and tool policy;
- route requests to the selected adapter or agent;
- maintain a local audit journal;
- maintain the outbound hosted relay session when hosted mode is enabled;
- cache the last valid signed hosted policy for bounded offline operation.

The edge connector must not accept arbitrary command payloads from the cloud or
MCP layer. Its internal API accepts only versioned typed operations.

### 5.3 Host agent

Responsibilities:

- run as an operating-system service;
- retain a stable local `agent_id` and private key;
- use the machine’s existing Tailscale connectivity to dial the connector;
- advertise platform, version, and capabilities;
- execute typed operations through a platform adapter;
- apply agent-local policy and bounds;
- return normalized structured results;
- rotate certificates and reconnect;
- keep a local security/audit journal.

The host agent has no MCP endpoint and no public listener.

### 5.4 Host adapter

A host adapter is an OS-specific implementation of the normalized operations.
It MUST NOT accept a raw command string.

Proposed Go interface, illustrative rather than source-compatible:

```go
type HostAdapter interface {
    Capabilities(context.Context) (CapabilitySet, error)
    InspectHost(context.Context, InspectHostRequest) (HostInspection, error)
    SystemSummary(context.Context, SystemSummaryRequest) (SystemSummary, error)
    DiskUsage(context.Context, DiskUsageRequest) (DiskUsageResult, error)
    ProcessList(context.Context, ProcessListRequest) (ProcessListResult, error)
    ServiceStatus(context.Context, ServiceStatusRequest) (ServiceStatusResult, error)
    ServiceLogs(context.Context, ServiceLogsRequest) (ServiceLogsResult, error)
}
```

Future write methods belong to a separate interface so a read-only build cannot
accidentally register write behavior.

### 5.5 Hosted control plane

Responsibilities:

- user and workspace identity;
- OAuth authorization server integration;
- public MCP resource server;
- connector pairing and registry;
- host inventory projections;
- central policy and approval workflow;
- relay routing and backpressure;
- audit metadata, billing, and support tooling;
- plugin review/demo environment.

## 6. Deployment modes

### 6.1 DIY local

- Gateway runs on a user-owned macOS, Linux, or Windows machine.
- Gateway uses OpenAI Secure MCP Tunnel or another private MCP transport.
- Linux targets may use Tailscale SSH.
- Other targets use the host agent.
- All policy, credentials, and audit data remain local.

### 6.2 Hosted with private edge

- Public MCP and OAuth run in our cloud.
- One or more edge connectors run in the customer environment.
- Host agents connect to an edge connector over the tailnet.
- The connector connects outbound to the hosted relay.
- Raw host output is streamed to the MCP response and is not retained by
  default.

### 6.3 Self-hosted control plane

- Same protocol and edge binaries as hosted mode.
- Customer deploys the MCP, identity integration, relay, database, and audit
  sink.
- Supported only after the hosted protocol and migrations stabilize.

### 6.4 Fully hosted tailnet node

A cloud-hosted connector that joins a customer tailnet is deferred. It creates a
larger trust boundary because our infrastructure would hold active node state
inside the customer tailnet. If implemented, it requires dedicated per-tenant
isolation, explicit consent, rapid revocation, encrypted node state, and a
separate threat model.

## 7. Platform support model

Support levels:

- **Tier 1:** release-blocking CI and integration tests; signed installer;
  documented support.
- **Tier 2:** binaries are produced and smoke-tested; best-effort support.
- **Experimental:** compile or community-tested only.

Initial target matrix:

| Component | Platform | Initial level | Notes |
| --- | --- | --- | --- |
| Gateway/connector | macOS arm64/amd64 | Tier 1 | User launch agent first; system deployment later. |
| Gateway/connector | Linux amd64/arm64 | Tier 1 | systemd package and container. |
| Gateway/connector | Windows amd64 | Tier 1 | Windows Service and MSI. |
| Gateway/connector | Windows arm64 | Tier 2 | Promote after hardware CI exists. |
| Host agent | Windows 11 / Server 2022+ amd64 | Tier 1 | First cross-platform target milestone. |
| Host agent | Windows 10 / Server 2016/2019 | Tier 2 | Align with current Tailscale support where feasible. |
| Host agent | Linux amd64/arm64 | Tier 1 after Windows | Agentless SSH remains available. |
| Host agent | macOS arm64/amd64 | Tier 1 after Windows | Headless Tailscale behavior must be documented. |

Mobile operating systems are not host targets.

## 8. Transport selection

Each host record contains an ordered list of available transports:

```text
agent
tailscale_ssh
openssh
winrm (future)
```

Default policy:

- Windows: `agent` only.
- macOS: `agent`, with Tailscale SSH only when explicitly detected and enabled.
- Linux: `tailscale_ssh` initially; prefer `agent` when richer capabilities or
  stricter local isolation are desired.

A transport is selected only when:

1. the host advertises or is discovered with that transport;
2. workspace policy permits it;
3. local connector policy permits it;
4. required capabilities and versions are present;
5. the target is online and authenticated.

No silent fallback to password authentication is permitted.

## 9. Agentless adapters

### 9.1 Tailscale SSH

The existing adapter remains the default zero-install Linux path:

- Tailscale network identity;
- Tailscale SSH authorization policy;
- `none` SSH authentication;
- strict host-key verification from Tailscale status;
- fixed commands constructed inside the adapter package;
- dedicated non-root account.

The adapter MUST continue to fail closed when the node is shared, expired,
offline, ambiguous, not Linux-compatible, or missing control-plane SSH host
keys.

### 9.2 Traditional OpenSSH

Traditional OpenSSH over Tailscale is an advanced fallback, not the default
Windows design.

Requirements if enabled:

- host-key pinning or an explicit SSH CA;
- key stored only in an OS secure store or external secret provider;
- exact username and host allowlist;
- fixed commands only;
- no password authentication;
- separate adapter and threat model;
- disabled in community defaults.

On Windows, Microsoft OpenSSH supports password and public-key authentication,
not Tailscale SSH identity. That makes it less attractive than the host agent.

### 9.3 WinRM / PowerShell Remoting

Deferred enterprise adapter. Reasons:

- credential and domain configuration complexity;
- broad remote-management semantics;
- greater risk of turning typed tools into a command wrapper;
- different behavior across workgroup, domain, certificate, and Kerberos setups.

If added, it must implement the same typed adapter interface and may not expose a
raw PowerShell tool.

## 10. Host-agent session topology

### 10.1 Direction

The host agent initiates and maintains the connection to the connector. This
avoids opening a listener on every target and works naturally on Windows.

Proposed private endpoint:

```text
connector MagicDNS name:7443
```

The connector listens through its embedded `tsnet.Server`; the listener is not
bound to the host LAN or public interface.

### 10.2 Tailnet policy

Current Linux SSH direction:

```text
tag:chatgpt-ssh -> tag:chatgpt-managed tcp:22
```

Agent-session direction:

```text
tag:chatgpt-managed -> tag:chatgpt-ssh tcp:7443
```

A future product rename may introduce new tag names, but existing tags must be
accepted through a documented migration period.

### 10.3 Layered authentication

An agent session is accepted only when all checks pass:

1. Connection arrived through the tailnet-only listener.
2. Connector resolves the source through Tailscale LocalAPI `WhoIs` or an
   equivalent identity-bearing listener API.
3. Source node ID and tags satisfy local policy.
4. TLS client certificate chains to the connector or workspace agent CA.
5. Certificate claims match the enrolled `agent_id`, workspace, and expected
   Tailscale node identity.
6. Certificate is not expired or revoked.
7. Agent protocol version is supported.

Tailscale identity is not replaced by mTLS; both are required.

## 11. Agent enrollment

### 11.1 Local/DIY enrollment

1. Operator runs `gateway agent invite`.
2. Gateway emits a one-time, short-lived pairing code and connector address.
3. Operator runs `host-agent enroll --connector ... --code ...` on the target.
4. Agent generates a private key locally and sends a CSR.
5. Connector verifies the pairing code and Tailscale source identity.
6. Connector issues a short-lived agent certificate and records the binding.
7. Pairing code is destroyed after one successful use.

### 11.2 Hosted enrollment

1. User signs in to the dashboard and creates a connector pairing code.
2. Edge connector pairs to the workspace using a device-code-style flow.
3. Host-agent enrollment remains local to the edge connector by default.
4. The cloud receives agent public metadata and certificate fingerprints, not
   private keys or Tailscale enrollment material.

### 11.3 Key storage

- Windows: Windows certificate/key storage protected by the service identity;
  DPAPI-protected fallback only when a certificate store cannot be used.
- macOS: Keychain for the service/user context.
- Linux: root- or service-user-owned `0600` state initially; TPM-backed storage
  is a later enhancement.

Private keys must never be printed, placed in process arguments, or copied into
the hosted control plane.

### 11.4 Rotation and revocation

- Agent certificates are short-lived and automatically renewed.
- Connector certificate and relay credentials rotate independently.
- Revocation can target a user, connector, agent, host, or certificate serial.
- Revocation state is pushed to connected edges and included in signed policy.
- An offline connector may use cached policy only until its configured
  `offline_valid_until` time.

## 12. Agent protocol

### 12.1 Transport

The preferred implementation is protobuf over a mutually authenticated
bidirectional gRPC stream using TLS 1.3. The stream runs over the private
tailnet connection for host agents and over outbound HTTPS for the hosted relay.

A WebSocket transport may be added for networks that cannot sustain HTTP/2, but
it must carry the same protobuf messages and security semantics.

### 12.2 Versioning

- Protocol versions use `major.minor`.
- A major mismatch rejects the session.
- Minor versions negotiate the intersection of capabilities.
- Every operation and result declares its schema version.
- Unknown fields are ignored only where protobuf compatibility permits.
- Feature use depends on explicit capabilities, never only on version number.

### 12.3 Session messages

Illustrative messages:

```text
AgentHello
ConnectorHello
Heartbeat
CapabilityUpdate
InvokeRequest
InvokeChunk
InvokeResult
CancelRequest
PolicyUpdate
RotateCertificate
Drain
Error
```

`AgentHello` fields:

- protocol version;
- agent version and build digest;
- stable `agent_id`;
- stable `host_id`;
- platform, architecture, and OS version;
- Tailscale node ID observed locally when available;
- capability set and tool schema versions;
- certificate serial;
- nonce.

`InvokeRequest` fields:

- globally unique request ID;
- workspace and connector routing context;
- stable host ID;
- exact tool name and schema version;
- typed protobuf `Any` or oneof arguments;
- deadline;
- policy digest;
- read/write classification;
- optional immutable approval reference for future actions.

`InvokeResult` fields:

- request ID;
- normalized status;
- typed result;
- platform details;
- truncation and redaction indicators;
- warnings;
- duration;
- adapter and capability versions.

Internal request IDs are retained in audit metadata. Public MCP responses should
not expose internal identifiers unless they are needed for a user-facing support
workflow.

### 12.4 Bounds

Defaults, configurable downward by policy:

- maximum concurrent reads per agent: 4;
- maximum queued reads per agent: 16;
- maximum request envelope: 256 KiB;
- maximum result envelope: 1 MiB;
- maximum log entries: 200;
- maximum process rows: 500, with lower MCP defaults;
- maximum single-operation duration: 30 seconds for v1;
- heartbeat interval: 20 seconds;
- reconnect backoff: exponential with jitter, 1 second to 60 seconds.

The connector may impose stricter bounds. The least permissive bound wins.

### 12.5 Cancellation and retry

- Read operations are idempotent and may be retried with the same idempotency
  key after a transport failure.
- The agent keeps a short bounded cache of completed request IDs to deduplicate
  retries.
- Cancellation is best effort and must stop any spawned fixed helper process.
- Future writes require a separate durable execution journal and single-use
  action nonce.

## 13. Stable host identity

Hostnames and IPs are selectors, not identity.

Each host agent generates and persists a random UUID `host_id` during first
enrollment. Connector records bind it to:

- `agent_id`;
- Tailscale node ID;
- public-key fingerprint;
- platform;
- optionally a one-way hash of an OS machine identifier for clone detection.

Raw Windows MachineGuid, Linux machine-id, or macOS hardware identifiers should
not be sent to the hosted service by default.

Agentless Linux hosts use the Tailscale stable node ID as the primary identity
until an agent is installed. Migration links the existing node record to the
new `host_id` after explicit verification.

## 14. Capability model

Capabilities are namespaced and versioned, for example:

```text
host.inspect@1
disk.usage@1
process.list@1
service.status@1
service.logs@1
log.windows_event@1
log.systemd_journal@1
log.macos_unified@1
action.service.restart@1 (future)
```

A host may be online but lack a capability. The MCP layer must return
`CAPABILITY_UNAVAILABLE`, not fabricate an empty successful result.

Capabilities include constraints:

- maximum rows or log entries;
- whether command lines are available;
- available service/log providers;
- whether privileged data is omitted;
- supported filter fields;
- adapter version.

## 15. Platform adapters

### 15.1 Windows

The default Windows adapter uses native Windows interfaces rather than invoking
arbitrary PowerShell.

Planned data sources:

- OS and memory: native system APIs, with a narrowly scoped CIM fallback only
  when needed;
- volumes: `GetLogicalDrives`, `GetVolumeInformation`, and
  `GetDiskFreeSpaceEx`-class APIs;
- processes: Tool Help/process APIs with access-denied handling per process;
- services: Service Control Manager query APIs;
- logs: Windows Event Log `EvtQuery`/render APIs;
- uptime/boot: native tick/boot-time APIs;
- agent lifecycle: Windows Service Control Manager.

Service identity:

- install as a dedicated virtual service account/service SID where practical;
- no interactive login;
- read-only permissions by default;
- optionally add only the required event-log read rights;
- do not run as LocalSystem merely to obtain richer process command lines;
- future privileged actions use a separately reviewed privilege boundary.

Behavior:

- service names are compared case-insensitively but returned with canonical
  names;
- inaccessible process details are marked unavailable per row;
- event log channels are exact-allowlisted;
- PowerShell is not a generic execution transport;
- fixed embedded PowerShell may be considered only as an explicit fallback with
  hash verification and no caller-provided script text.

The installer checks that Tailscale is connected and recommends Windows
unattended mode so the agent can reconnect after reboot without a logged-in
user.

### 15.2 Linux

Two implementations coexist:

- existing Tailscale SSH fixed-command adapter;
- local host-agent adapter.

The local adapter should progressively replace command parsing with native
interfaces:

- `/proc` and `/sys` for system/process facts;
- filesystem APIs for disk use;
- systemd D-Bus for service status where available;
- journal APIs or fixed `journalctl` fallback for logs;
- capability degradation on non-systemd distributions.

The Linux agent runs under a dedicated service account. Journal access is an
optional explicit permission.

### 15.3 macOS

Planned data sources:

- system facts: `sysctl` and native APIs;
- filesystem: `statfs`/volume APIs;
- processes: `libproc` or approved native interfaces;
- services: `launchd`/ServiceManagement-compatible inspection;
- logs: unified logging APIs or fixed bounded `log show` fallback;
- lifecycle: signed `launchd` service.

The mainstream macOS Tailscale GUI does not provide the same headless service
behavior as Linux. Packaging must clearly distinguish a user-session agent from
an always-on headless Mac deployment and test both supported modes.

## 16. Local policy

Local connector policy is authoritative and versioned. Illustrative YAML:

```yaml
version: 1
hosts:
  - match:
      tags: ["tag:chatgpt-managed"]
    allow:
      tools:
        - host.inspect
        - system.summary
        - disk.usage
        - process.list
        - service.status
        - service.logs
      services:
        - tailscaled.service
        - nginx.service
        - Tailscale
      log_sources:
        - systemd:tailscaled.service
        - windows:System
        - windows:Application
      limits:
        process_rows: 100
        log_entries: 200
        output_bytes: 262144
```

Rules:

- deny by default;
- exact service and log-source allowlists;
- no wildcard that expands to every write action;
- host selectors resolve to stable IDs before execution;
- hosted policy can narrow but not widen local policy;
- policy updates are atomic and retain the last valid version;
- signed hosted policy includes expiry and monotonic revision;
- local emergency deny takes effect without cloud connectivity.

## 17. Hosted identity and OAuth

A public plugin uses a stable public HTTPS MCP endpoint. Secure MCP Tunnel is
for private connections and developer testing, not public plugin distribution.

The hosted MCP resource server must:

- publish MCP protected-resource metadata;
- use an established OAuth 2.1/OIDC provider;
- support authorization-code flow with PKCE;
- validate token signature, issuer, audience/resource, expiry, and scopes on
  every request;
- define per-tool security schemes and scopes;
- support revocation and reauthorization;
- optionally validate OpenAI’s MCP client mTLS identity at the edge while still
  authenticating the end user with OAuth.

Proposed scopes:

```text
hosts:read
systems:read
processes:read
services:read
logs:read
actions:prepare (future)
actions:execute (future, normally not granted directly)
admin:connectors
admin:policy
admin:audit
```

## 18. Hosted data model

Minimum entities:

- `Workspace`
- `User`
- `Membership`
- `Role`
- `Connector`
- `ConnectorCredential`
- `Agent`
- `Host`
- `HostTransport`
- `Capability`
- `Policy`
- `PolicyRevision`
- `ToolInvocation`
- `Approval` (future)
- `AuditEvent`
- `Subscription`

Important relationships:

- a workspace owns connectors and policy;
- a connector owns or observes agents/hosts;
- a host can expose multiple transports;
- user scopes and workspace policy are evaluated before connector dispatch;
- connector local policy is evaluated again at the edge;
- tool invocation output is separate from audit metadata and is not retained by
  default.

## 19. Hosted relay semantics

- Connector opens an outbound mutually authenticated stream to a regional relay.
- Relay authenticates connector and workspace before registering presence.
- MCP server dispatches by connector ID and request ID.
- Relay applies bounded queues and rejects rather than silently dropping work.
- Connector reconnects with exponential backoff and resumes only pending
  idempotent reads.
- Messages are encrypted in transit; sensitive fields may additionally use
  application-layer envelope encryption.
- No relay operator can convert a typed operation into an arbitrary command.
- Connector verifies policy digest and tool schema before execution.

Availability behavior:

- If no connector is online, return a clear retriable `CONNECTOR_OFFLINE` error.
- Do not queue interactive requests for unbounded time.
- Hosted policy may be cached locally only until expiry.
- Future scheduled work requires a separate durable scheduler and explicit user
  consent.

## 20. Data minimization and retention

Community mode stores only local state and audit data chosen by the operator.

Hosted defaults:

- do not persist raw process, log, or command output;
- stream result payloads through the relay to the MCP response;
- retain metadata: actor, workspace, host, tool, policy decision, timestamps,
  duration, byte counts, truncation, and status;
- do not include access tokens, keys, environment values, or full command lines
  in logs;
- redact common secret formats before response as defense in depth;
- make any output retention opt-in, time-bounded, encrypted, and visible;
- support immediate deletion and retention-policy enforcement jobs.

Public plugin responses should omit internal tenant IDs, trace IDs, and agent
fingerprints unless directly necessary for the user’s request.

## 21. Prompt-injection boundary

Hostnames, process arguments, service descriptions, and logs are untrusted.

Controls:

- mark remote strings as untrusted in MCP content metadata/instructions;
- return structured fields rather than concatenated instructions;
- avoid automatically passing one tool’s text into a write action;
- strip terminal control sequences and invalid encodings;
- cap size and line length;
- redact secrets;
- require a new policy and approval decision for every future write;
- never interpret host output as connector configuration or policy.

## 22. Audit model

Local and hosted audit events share a documented schema:

- event ID;
- timestamp;
- actor type and stable pseudonymous ID;
- workspace/connector/host references;
- tool and schema version;
- transport and adapter;
- policy revision and decision;
- requested bounds;
- outcome, duration, bytes, truncation, and error code;
- approval reference for writes;
- software versions and build digests.

Raw output is not part of the default audit event.

Local audit must be append-oriented and rotate with bounded disk use. Enterprise
hosted mode may export to a customer SIEM.

## 23. Packaging and service lifecycle

### 23.1 Shared CLI design

The source of truth becomes a Go CLI:

```text
gateway setup
gateway check
gateway run
gateway service install|start|stop|status|uninstall
gateway agent invite
gateway diagnostics

host-agent enroll
host-agent check
host-agent run
host-agent service install|start|stop|status|uninstall
host-agent diagnostics
```

Bash and PowerShell wrappers call these commands but do not reimplement state
migration.

### 23.2 Paths

Proposed defaults:

Linux:

```text
/etc/chatgpt-plugin-tailscale/
/var/lib/chatgpt-plugin-tailscale/
/var/log/chatgpt-plugin-tailscale/
```

macOS system install:

```text
/Library/Application Support/chatgpt-plugin-tailscale/
/Library/Logs/chatgpt-plugin-tailscale/
```

macOS user install may use the corresponding `~/Library` directories.

Windows:

```text
%ProgramData%\chatgpt-plugin-tailscale\config
%ProgramData%\chatgpt-plugin-tailscale\state
%ProgramData%\chatgpt-plugin-tailscale\logs
```

Secrets use OS secure storage when available; configuration files contain
references, not secret values, in hosted production mode.

### 23.3 Installers

- macOS: signed/notarized universal package plus Homebrew formula/cask;
- Linux: signed DEB/RPM and static tarball; container for connector only;
- Windows: signed MSI and optional `winget` package;
- checksums, SBOM, provenance, and release signatures for every artifact.

### 23.4 Updates

- signed update manifest;
- stable, beta, and pinned channels;
- staged connector/agent rollout;
- compatibility check before restart;
- atomic binary replacement;
- automatic rollback when health checks fail;
- no unattended major-version migration.

## 24. Observability

Local health endpoints or service status expose:

- process health;
- tailnet/relay connectivity;
- connector and agent session counts;
- certificate expiry;
- policy revision and expiry;
- queue depth;
- recent normalized error counts;
- build version.

Metrics must not include host output, secrets, usernames, process arguments, or
raw IP addresses by default.

Diagnostic bundles:

- require explicit generation;
- redact keys and tokens;
- include config shape, versions, health, and recent error metadata;
- list every included file before archive creation.

## 25. Error model

Canonical errors are defined in the tool contract. Transport-specific errors
must map to stable categories such as:

- `HOST_NOT_FOUND`
- `HOST_OFFLINE`
- `CONNECTOR_OFFLINE`
- `AGENT_OFFLINE`
- `CAPABILITY_UNAVAILABLE`
- `POLICY_DENIED`
- `SERVICE_NOT_ALLOWED`
- `LOG_SOURCE_NOT_ALLOWED`
- `AUTHENTICATION_FAILED`
- `CERTIFICATE_EXPIRED`
- `VERSION_UNSUPPORTED`
- `DEADLINE_EXCEEDED`
- `OUTPUT_LIMIT_EXCEEDED`
- `PLATFORM_ACCESS_DENIED`
- `INTERNAL_ERROR`

User-facing messages must be actionable without exposing secrets or internal
stack traces.

## 26. Testing strategy

### 26.1 Unit and conformance

- adapter fixtures for every normalized state;
- schema golden tests;
- input validation and allowlist tests;
- output truncation/redaction tests;
- protocol compatibility tests;
- policy intersection tests;
- stable host identity and migration tests;
- Windows access-denied-per-process behavior;
- service and log normalization fixtures.

### 26.2 Cross-platform CI

- Go tests on `ubuntu-latest`, `windows-latest`, and `macos-latest`;
- cross-compilation for supported architecture pairs;
- installer build validation;
- service install/start/stop tests where hosted runners allow it;
- dependency and vulnerability scanning;
- SBOM/provenance validation.

### 26.3 Tailnet integration fleet

A private test tailnet should include:

- Linux Tailscale SSH target;
- Linux host-agent target;
- Windows host-agent target in unattended mode;
- macOS host-agent target;
- connector on each Tier 1 platform over time;
- deliberately denied and expired nodes;
- agent certificate rotation and revocation cases;
- network interruption and reboot cases.

No production customer credential belongs in CI. Use scoped ephemeral test
identity where possible.

### 26.4 Security testing

- fuzz protocol and result parsers;
- malformed Unicode and terminal escape payloads;
- prompt-injection log corpus;
- secret-pattern corpus;
- replay and duplicate request tests;
- certificate substitution and node-ID mismatch tests;
- hosted cross-tenant routing tests;
- policy downgrade and expired-cache tests;
- write-action mutation tests before writes ship.

## 27. Reliability targets

Initial design targets, to be validated before becoming public SLOs:

- connector detects a dropped relay or agent session within 60 seconds;
- healthy agent reconnects within 90 seconds after network restoration;
- interactive requests do not wait more than their explicit deadline;
- bounded queues reject with backpressure rather than exhaust memory;
- connector restart preserves identity and host mappings;
- a failed update rolls back without losing node or agent state;
- read requests are safe to retry and deduplicate.

## 28. Public plugin requirements

The hosted edition, not Secure MCP Tunnel, is the public distribution path.
Before submission it needs:

- publicly accessible production HTTPS MCP endpoint;
- verified publisher identity and domain;
- OAuth 2.1 protected-resource and authorization metadata;
- accurate per-tool security schemes and annotations;
- privacy policy, terms, support path, and data-retention disclosures;
- deterministic test prompts and expected results;
- a demo account and synthetic connector/hosts that require no private network
  or MFA during review;
- no unnecessary PII or internal IDs in tool responses;
- operational monitoring and incident response.

## 29. Migration from the current PR

The current implementation should evolve without a rewrite:

1. Keep existing MCP names and Linux behavior stable.
2. Define versioned output schemas and conformance fixtures.
3. Extract current catalog/runner behavior behind `HostTransport` and
   `HostAdapter` interfaces.
4. Implement an in-process adapter registry.
5. Add stable host IDs while preserving Tailscale node selectors.
6. Add connector tailnet listener and agent session manager.
7. Add Windows agent and adapter.
8. Replace Bash-only setup logic with the Go setup command; retain current
   scripts as wrappers and migration tests.
9. Add hosted relay as an optional connector mode.
10. Split public MCP/control-plane services only when hosted alpha begins.

Existing `.env`, `data/tsnet`, tags, and Linux targets must continue to work.

## 30. Proposed repository layout

```text
cmd/
  gateway/
  host-agent/
  control-plane/          # hosted, may move to a separate repository later

internal/
  adapter/
    linux/
    windows/
    darwin/
  agentproto/
  agentserver/
  agentclient/
  hostcatalog/
  policy/
  audit/
  relay/
  mcpapp/
  transport/
    tailscalessh/
    agent/
    openssh/              # optional

proto/
  agent/v1/
  relay/v1/
  tools/v1/

packaging/
  darwin/
  linux/
  windows/

docs/
  roadmap.md
  cross-platform-spec.md
  tool-contracts-v1.md
```

Hosted code may eventually live separately, but protocol definitions and edge
conformance tests should remain accessible with the open-source components.

## 31. Decisions made by this specification

1. Windows support is agent-first, not Tailscale SSH, WinRM, or password SSH.
2. Host agents initiate outbound sessions to the connector.
3. Tailscale source identity and mTLS are both required.
4. Linux retains the current agentless path.
5. MCP tool names remain stable while outputs become normalized and versioned.
6. Local policy is always authoritative.
7. Hosted cloud cannot directly dial the tailnet.
8. Raw output is not retained by default in hosted mode.
9. Core edge and protocol remain open source.
10. Write actions require a separate immutable approval protocol.

## 32. Open decisions requiring implementation spikes

- gRPC-only versus gRPC plus WebSocket fallback for agent and relay streams;
- exact Windows service identity and Event Log ACL installation behavior;
- whether macOS Tier 1 requires the standalone/open-source Tailscale client for
  always-on service operation;
- local CA per connector versus workspace agent CA with connector delegation;
- database and relay technology for hosted alpha;
- release signing service and key custody;
- how much host inventory metadata is projected to cloud when zero-retention is
  enabled;
- final product name and migration away from brand-specific tags/service IDs.

Each spike should produce an ADR, a minimal test, and an explicit go/no-go
criterion.

## 33. Reference material

- Tailscale SSH support and limitations:
  <https://tailscale.com/docs/features/tailscale-ssh>
- Tailscale Windows unattended mode:
  <https://tailscale.com/docs/how-to/run-unattended>
- `tsnet.Server`:
  <https://tailscale.com/docs/reference/tsnet-server-api>
- Tailscale Windows client support:
  <https://tailscale.com/docs/install/windows>
- Microsoft OpenSSH for Windows:
  <https://learn.microsoft.com/windows-server/administration/openssh/openssh-overview>
- Windows Service Control Manager status APIs:
  <https://learn.microsoft.com/windows/win32/api/winsvc/nf-winsvc-queryservicestatusex>
- Windows Event Log queries:
  <https://learn.microsoft.com/windows/win32/wes/querying-for-events>
- OpenAI Secure MCP Tunnel:
  <https://developers.openai.com/api/docs/guides/secure-mcp-tunnels>
- OpenAI plugin authentication:
  <https://developers.openai.com/plugins/build/auth>
- OpenAI MCP server review requirements:
  <https://developers.openai.com/plugins/deploy/app-review>
