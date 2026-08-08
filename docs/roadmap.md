# Product roadmap

Status: planning baseline for the project after the Linux proof of concept  
Last updated: 2026-08-08

This document defines how the current Linux-only Tailscale SSH inspector can
become a cross-platform private-infrastructure connector for ChatGPT, Codex, and
other MCP clients. It is a sequence of product milestones, not a calendar
commitment. Dates depend on maintainer capacity, external platform behavior, and
security review.

The detailed target architecture is in
[`cross-platform-spec.md`](cross-platform-spec.md), and the normalized MCP
contracts are in [`tool-contracts-v1.md`](tool-contracts-v1.md).

## Product thesis

People increasingly want an AI assistant to answer questions about machines
that are intentionally unreachable from the public internet. Existing choices
usually require one of the following:

- exposing an MCP or SSH endpoint publicly;
- placing a reusable SSH key in a third-party configuration;
- granting an unrestricted remote shell;
- assembling several unrelated tools and identity systems by hand.

The product should instead provide a narrow, inspectable path:

```text
AI client -> authenticated MCP -> policy -> private connector -> typed host tool
```

The durable product is not “SSH in ChatGPT.” It is a safe operations layer for
private infrastructure, with Tailscale as the first and preferred network and
identity substrate.

## Product principles

1. **Private by default.** No inbound public ports are required.
2. **No generic shell.** User input must not become a command string.
3. **Two-sided authorization.** Cloud or MCP policy and local edge policy must
   both allow an operation.
4. **Typed operations.** Tools return normalized structured data across
   operating systems.
5. **Least privilege.** Read-only host inspection ships before write actions.
6. **Local custody.** Tailscale enrollment material, host credentials, and host
   policy remain on customer-controlled machines whenever possible.
7. **Open edge.** The code that reaches and operates customer machines remains
   reviewable and self-hostable.
8. **Explicit risk.** Every write action is prepared, approved, and executed as
   an immutable action rather than inferred from a conversational sentence.
9. **Cross-platform parity without pretending platforms are identical.** Tools
   expose normalized fields plus platform-specific details and capability flags.
10. **Rerunnable onboarding.** Installers and setup flows detect existing state,
    preserve user changes, and fail closed.

## Editions

### Community / DIY

The community edition remains fully usable without our hosted service:

- open-source gateway and host agent;
- OpenAI Secure MCP Tunnel for private ChatGPT development apps;
- direct MCP support for other compatible clients;
- Tailscale SSH for zero-install Linux targets;
- cross-platform host agent for Windows, macOS, and Linux;
- local policy, audit log, and configuration;
- signed release binaries and packages;
- no product-imposed usage fee or host limit.

### Hosted

The hosted edition removes the hardest operational steps:

- a stable public MCP endpoint;
- OAuth user and workspace authentication;
- a published ChatGPT/Codex plugin;
- an outbound edge connector instead of `tunnel-client`;
- connector and host health;
- centralized policy, approvals, and audit metadata;
- managed updates and support;
- optional retention, redaction, and enterprise controls.

The hosted service must not require a long-lived Tailscale OAuth secret in our
cloud. The customer edge connector should join the tailnet locally and maintain
an outbound relay to the hosted control plane.

### Enterprise / self-hosted control plane

Later enterprise options may include:

- customer-hosted control plane or bring-your-own-cloud deployment;
- SSO, SCIM, RBAC, and approval groups;
- customer-managed encryption keys;
- regional data residency and configurable retention;
- SIEM and compliance audit export;
- private package repositories and managed rollout rings.

## Core user journeys

### DIY individual

1. Install the gateway on macOS, Linux, or Windows.
2. Run one setup command.
3. Apply a generated Tailscale policy fragment.
4. Use agentless Tailscale SSH for Linux and install the host agent on Windows
   or macOS.
5. Connect the local MCP endpoint through Secure MCP Tunnel.
6. Ask ChatGPT to inspect the allowed machines.

### Hosted individual or small team

1. Install the public plugin and sign in.
2. Pair one edge connector with a short-lived device code.
3. Apply the generated Tailscale policy fragment.
4. Install host agents where agentless Linux access is unavailable or
   undesirable.
5. Select hosts and tools in the dashboard.
6. Ask ChatGPT questions without developer mode, a Platform tunnel, or local
   terminal sessions.

### Enterprise workspace

1. An administrator deploys signed connector and agent packages through MDM or
   configuration management.
2. Users and groups are provisioned through the identity provider.
3. Policy maps environments, hosts, tools, and approval requirements to groups.
4. ChatGPT users receive only the tools and hosts authorized for their identity.
5. Audit events flow to the organization’s SIEM.

## Platform strategy

Tailscale SSH’s server component is not available on Windows. The cross-platform
product therefore uses two target transports:

| Target path | Linux | Windows | macOS |
| --- | --- | --- | --- |
| Tailscale SSH adapter | Primary zero-install option | Not supported | Optional only with open-source `tailscale` + `tailscaled` |
| Host agent | Supported | Primary option | Primary option |
| Traditional OpenSSH adapter | Advanced fallback | Advanced fallback | Advanced fallback |
| WinRM / PowerShell Remoting | N/A | Deferred enterprise adapter | N/A |

The gateway or edge connector itself must run on macOS, Linux, and Windows. The
core implementation remains Go, with platform-specific service wrappers and
native host adapters.

## Milestone 0 — Linux proof of concept

Status: implemented in PR #1.

Scope:

- embedded `tsnet` gateway identity;
- Linux target discovery and local allowlists;
- strict Tailscale SSH host-key verification;
- seven read-only MCP tools;
- bounded output and deadlines;
- idempotent bootstrap for the current macOS/Linux path;
- OpenAI Secure MCP Tunnel integration;
- live end-to-end validation from ChatGPT.

Exit criteria:

- `list_hosts`, host inspection, disk, process, service, and log reads work on an
  explicitly tagged Linux target;
- no SSH private key or unrestricted command tool exists;
- tests, vet, build, and container build pass.

## Milestone 1 — Cross-platform build and installation foundation

Goal: make the connector a normal installable product before adding more target
platforms.

Deliverables:

- build matrix for `darwin`, `linux`, and `windows` on `amd64` and `arm64` where
  the dependency graph supports it;
- a Go-native `setup` command that replaces Bash as the source of truth;
- thin Bash and PowerShell launchers for familiar install experiences;
- platform configuration and state directories;
- `launchd`, `systemd`, and Windows Service integration;
- graceful upgrade and rollback hooks;
- release manifest, SHA-256 checksums, SBOMs, provenance, and signature
  verification;
- Apple Developer ID signing and notarization plan;
- Windows Authenticode signing and SmartScreen reputation plan;
- Homebrew, direct tar/zip, MSI, and Linux package installation plans;
- CI compile and unit-test matrix for all supported platforms.

Exit criteria:

- a new user can install, configure, start, stop, inspect, and uninstall the
  gateway on each Tier 1 desktop platform without cloning the repository;
- a rerun does not replace credentials, node state, or custom policy;
- service startup survives reboot;
- release artifacts can be verified independently.

## Milestone 2 — Host-agent protocol and Windows read-only MVP

Goal: satisfy the concrete mixed Linux/Windows household or small-fleet use
case without requiring Windows SSH credentials.

Deliverables:

- `host-agent` Windows service;
- outbound agent session from the Windows host to the private connector over
  the tailnet;
- connector-side verification of Tailscale source identity plus agent mTLS;
- one-time pairing and certificate rotation;
- Windows native adapters for:
  - host and OS facts;
  - uptime and memory;
  - filesystem volume usage;
  - process listing;
  - Service Control Manager status;
  - bounded Windows Event Log reads;
- normalized tool output matching the Linux tool contracts;
- Windows MSI and idempotent PowerShell bootstrap;
- Tailscale unattended-mode check and guidance;
- local policy controlling allowed services and log channels;
- agent capability discovery and version negotiation.

Windows MVP exit criteria:

- a Windows 11 or Windows Server test machine appears beside Linux hosts in
  `list_hosts`;
- all seven read-only MCP tools either return normalized Windows data or a
  precise `CAPABILITY_UNAVAILABLE` response;
- the agent reconnects after reboot without an interactive login;
- no Windows account password, reusable SSH key, WinRM credential, or arbitrary
  PowerShell command is required;
- uninstall removes the service and binaries but preserves or explicitly asks
  about identity and audit state.

## Milestone 3 — Agent parity on Linux and macOS

Goal: use one host model across all desktop/server operating systems while
retaining agentless Linux access.

Deliverables:

- Linux host agent using native/procfs/system interfaces where practical;
- macOS host agent using native system, process, filesystem, service, and
  unified-log interfaces;
- capability-aware selection between `tailscale_ssh` and `agent` transports;
- per-host transport preference and fallback policy;
- macOS `launchd` package and notarized installer;
- Linux DEB/RPM packages and systemd unit;
- agent-side local audit journal;
- migration tooling from an SSH-only host record to an agent-backed record.

Exit criteria:

- the same MCP prompt can inspect a mixed Linux, Windows, and macOS fleet;
- transport changes do not change the host’s stable identity;
- unsupported platform fields are omitted or explicitly marked unavailable,
  never fabricated;
- agent and agentless results pass the same conformance suite.

## Milestone 4 — Community read-only release

Goal: make the DIY project dependable for people who did not build it.

Deliverables:

- stable configuration schema and migration framework;
- one-command installers and uninstallers;
- upgrade channels and signed auto-update metadata, with auto-update opt-in;
- diagnostics bundle with secret redaction;
- documented backup and restore of connector and agent identity;
- integration test tailnet covering Linux, Windows, and macOS;
- compatibility matrix and support policy;
- example Ansible, PowerShell DSC, and MDM deployment snippets;
- security policy, disclosure process, and initial independent review;
- release notes and semantic versioning.

Exit criteria:

- time to first healthy host is under fifteen minutes for a typical technical
  user following the primary path;
- failed bootstrap steps are resumable and actionable;
- no known critical or high-severity issue remains open for the release;
- clean install, upgrade, rollback, and uninstall tests pass on Tier 1 systems.

## Milestone 5 — Hosted control-plane alpha

Goal: prove that users will pay to avoid developer-mode tunnel setup while
keeping machine access at the edge.

Deliverables:

- multi-tenant public MCP endpoint;
- OAuth 2.1 user authorization compatible with MCP clients;
- workspace, membership, connector, host, and policy models;
- one-time connector pairing;
- outbound connector-to-cloud relay with reconnect and backpressure;
- encrypted credential references and signed policy bundles;
- connector health and host inventory dashboard;
- audit metadata with host output retention disabled by default;
- tenant isolation tests;
- a synthetic demo fleet for plugin review and support;
- invitation-only alpha onboarding.

Exit criteria:

- a user installs the connector, pairs it, and uses the hosted MCP endpoint
  without OpenAI Secure MCP Tunnel or developer mode;
- cloud infrastructure cannot directly dial the customer tailnet;
- revoking a connector or user prevents new tool calls promptly;
- read-only calls are idempotent and tolerate relay reconnects;
- audit events identify actor, tool, host, policy decision, duration, and result
  without storing secrets or raw output by default.

## Milestone 6 — Hosted beta and public plugin submission

Goal: turn the alpha into a distributable product.

Deliverables:

- public HTTPS MCP endpoint with production availability controls;
- plugin OAuth metadata, PKCE-compatible authorization, scope enforcement, and
  token revocation;
- onboarding, billing, limits, support, privacy policy, and terms;
- test prompts and deterministic demo-account responses;
- accurate MCP tool annotations and security schemes;
- regional relay routing;
- customer-visible audit and retention controls;
- abuse prevention, rate limits, and incident-response runbooks;
- submission to the ChatGPT/Codex plugin directory.

Exit criteria:

- external beta users can onboard without maintainer intervention;
- the plugin passes all documented review and test cases;
- P95 read-tool dispatch latency, excluding host execution time, is measured and
  within the product SLO;
- no cross-tenant data access is possible in automated isolation tests;
- support can diagnose connector failures without requesting secrets.

## Milestone 7 — Read-only general availability

Goal: establish a trusted cross-platform infrastructure inspector.

Deliverables:

- support and deprecation policy;
- production SLOs and status page;
- restore and disaster-recovery exercises;
- external penetration test and remediation;
- enterprise SSO/RBAC baseline;
- SIEM export;
- hosted and self-hosted compatibility guarantees;
- agent and connector fleet update controls.

Exit criteria:

- Windows, Linux, and macOS Tier 1 paths meet the published compatibility
  matrix;
- connector upgrades can be staged and rolled back;
- security incident, revocation, and key-rotation drills are complete;
- data retention behavior is externally documented and enforced by tests.

## Milestone 8 — Controlled write actions

Goal: add high-value operations without adding a shell.

Initial action candidates:

- restart an exact allowlisted service;
- restart an exact allowlisted container;
- trigger an existing deployment or runbook;
- rotate an application log through a named action;
- apply an approved structured configuration value;
- reboot a non-production host under a dedicated policy.

Every action follows:

```text
prepare -> policy evaluation -> human approval -> execute immutable action
```

Required controls:

- immutable action hash and short expiry;
- single-use nonce and replay prevention;
- explicit target, effect, and rollback information;
- ChatGPT confirmation plus optional out-of-band administrator approval;
- local edge revalidation immediately before execution;
- accurate write/destructive MCP annotations;
- append-only audit events;
- per-environment maintenance windows and concurrency limits.

Exit criteria:

- there is still no arbitrary command, script, or unbounded file-write tool;
- prepared action mutation is cryptographically detectable;
- retries cannot execute the same action twice unintentionally;
- local policy can deny an action even when the hosted control plane approves it.

## Milestone 9 — Enterprise operations platform

Potential scope:

- reusable organization runbooks;
- multi-person and separation-of-duty approvals;
- temporary elevation and just-in-time policy;
- change windows and deployment verification;
- incident timelines and evidence bundles;
- integrations with monitoring, paging, source control, and ticketing systems;
- customer-managed control planes and regional isolation;
- policy-as-code and signed organization policy repositories.

This milestone should be driven by real customer workflows rather than broad
feature accumulation.

## Workstreams and sequencing

### A. Core host abstraction

1. Freeze v1 normalized tool contracts.
2. Introduce `HostTransport` and `HostAdapter` interfaces.
3. Move Linux SSH behavior behind the interfaces.
4. Add capability discovery and conformance tests.
5. Implement Windows, Linux-agent, and macOS adapters.

### B. Private networking and identity

1. Add connector listener for outbound host-agent sessions.
2. Bind agent enrollment to Tailscale node identity.
3. Add mTLS pairing, rotation, and revocation.
4. Add hosted connector relay.
5. Add signed policy distribution and offline policy cache.

### C. Packaging and lifecycle

1. Go-native setup and migration command.
2. Cross-platform service wrappers.
3. Signed release pipeline.
4. Package managers and MDM examples.
5. Update rings, rollback, and diagnostics.

### D. Hosted product

1. Tenant identity and public MCP resource server.
2. Connector pairing and relay.
3. Policy and audit models.
4. Dashboard and plugin onboarding.
5. Billing, support, and publication.

### E. Security and assurance

1. Threat-model each transport and deployment mode.
2. Fuzz parsers and protocol frames.
3. Prompt-injection and output-minimization tests.
4. Tenant-isolation and replay tests.
5. External review before GA and before write actions.

## Open-source and hosted boundary

The intended boundary is open edge, hosted convenience:

Open source:

- gateway/edge connector;
- host agent;
- host adapters and tool contracts;
- local policy evaluator;
- agent protocol and conformance suite;
- DIY bootstrap and Secure MCP Tunnel integration.

Hosted:

- multi-tenant identity and OAuth service;
- public MCP endpoint;
- global relay and connector registry;
- hosted policy UI, approvals, audit search, billing, and support operations.

The wire protocol should remain documented so users can self-host a compatible
control plane and inspect what crosses the boundary.

## Success metrics

### Activation

- median time from install to first successful host inspection;
- percentage of setup sessions that complete without maintainer intervention;
- percentage of Windows installs surviving reboot and reconnecting unattended;
- percentage of users who inspect more than one host in the first week.

### Reliability

- connector session availability;
- successful tool-call rate by platform and adapter;
- reconnect time after network loss or reboot;
- update success and rollback rate;
- P50/P95/P99 dispatch and execution latency.

### Trust and safety

- number of tool calls denied by local policy after upstream approval;
- secret-redaction escape rate in test fixtures;
- cross-tenant isolation failures: target zero;
- duplicate write executions: target zero;
- mean time to revoke a user, connector, agent, or certificate.

### Product value

- weekly active connectors and hosts;
- repeated incident/diagnostic workflows;
- successful answers that avoid a manual SSH session;
- conversion from community to hosted;
- retention after thirty and ninety days.

## Major risks

| Risk | Mitigation |
| --- | --- |
| Product is perceived as “AI with root” | Keep read-only default, no shell, dedicated identities, and visible policy. |
| Windows parity becomes a PowerShell wrapper | Implement native typed adapters; use embedded fixed scripts only as an explicit fallback. |
| Hosted relay becomes a new remote-access backdoor | Outbound connector, mTLS, tenant isolation, local policy, revocation, no cloud tailnet credential. |
| Prompt injection in logs and process arguments | Mark remote data untrusted, minimize fields, bound output, avoid model-driven follow-up actions. |
| Tailscale or OpenAI ships overlapping functionality | Differentiate on host operations, policy, approvals, audit, and cross-client support. |
| Onboarding remains too technical | Move logic into a signed setup binary, generate policy, test each boundary, and resume safely. |
| Open-core trust concerns | Keep agent, connector, protocol, and local policy open and independently usable. |
| Platform API differences produce misleading parity | Capability flags, normalized enums, raw platform fields, and conformance fixtures. |

## Explicit non-goals through read-only GA

- arbitrary shell execution;
- interactive SSH or terminal emulation;
- RDP or desktop control;
- unrestricted file browsing or file upload;
- password vaulting;
- replacing a monitoring, EDR, RMM, or configuration-management platform;
- automatic remediation without a deterministic action contract;
- silently editing a customer’s tailnet policy or machine tags.

## Recommended next implementation slice

The highest-value next slice is Milestone 2’s narrow Windows path:

1. define the host adapter and v1 tool schemas;
2. create a Windows service that opens an outbound authenticated agent session;
3. implement system summary, disk usage, process list, service status, and Event
   Log reads with native Windows APIs;
4. add a PowerShell bootstrap wrapper and MSI build;
5. run the same ChatGPT prompts against one Linux and one Windows target.

That proves the cross-platform architecture against the exact friend use case
before investing in the hosted control plane.

## External constraints and references

- Tailscale SSH server support and limitations:
  <https://tailscale.com/docs/features/tailscale-ssh>
- Tailscale unattended operation on Windows:
  <https://tailscale.com/docs/how-to/run-unattended>
- `tsnet.Server`:
  <https://tailscale.com/docs/reference/tsnet-server-api>
- Microsoft OpenSSH for Windows:
  <https://learn.microsoft.com/windows-server/administration/openssh/openssh-overview>
- OpenAI Secure MCP Tunnel:
  <https://developers.openai.com/api/docs/guides/secure-mcp-tunnels>
- OpenAI plugin authentication:
  <https://developers.openai.com/plugins/build/auth>
- OpenAI MCP server review requirements:
  <https://developers.openai.com/plugins/deploy/app-review>
