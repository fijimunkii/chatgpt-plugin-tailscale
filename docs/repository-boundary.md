# Repository and open-source boundary

Status: normative product and architecture decision  
Last updated: 2026-08-10

TetherBound uses an **open Community runtime, optional hosted control plane** model.

The code that reaches, authenticates to, inspects, or operates customer machines must remain open source and independently usable. Self-hosting is not a fallback path; it is the primary TetherBound Community product.

## Canonical Community repository

`fijimunkii/chatgpt-plugin-tailscale` is the canonical public Community and self-hosted repository.

The repository name is intentionally descriptive and search-friendly. **TetherBound** is the product/brand and security model; **chatgpt-plugin-tailscale** is the open-source distribution people can clone, inspect, modify, and run themselves.

A future `tetherbound/tetherbound` or `fijimunkii/tetherbound` repository may exist as an umbrella, organization landing point, SDK, or broader multi-product repository, but Community users must not be forced to migrate merely for branding reasons.

This repository owns customer-side trust-boundary code, including:

- TetherBound Edge / gateway;
- TetherBound Agent;
- Linux, Windows, and macOS host adapters;
- Tailscale SSH transport;
- agent transport;
- local host discovery and identity;
- local policy evaluation;
- local audit behavior;
- normalized MCP tool contracts;
- agent and connector protocol definitions;
- pairing, certificate, and local revocation logic;
- bootstrap/setup tooling;
- installers and service wrappers;
- update verification logic;
- conformance and security tests;
- Community MCP endpoint and Secure MCP Tunnel integration;
- public architecture, threat-model, security, operator, and self-hosting documentation.

Community functionality must not require TetherBound Cloud, a private package, a private registry, a hosted license check, or a proprietary runtime component to perform its documented local function.

## Hosted Cloud repository

A future hosted implementation belongs in a separate private repository such as:

```text
tetherbound/cloud
```

or:

```text
tetherbound/control-plane
```

It may contain:

- hosted authenticated MCP endpoints;
- OAuth/OIDC integration;
- workspace, membership, and RBAC implementation;
- connector registry and presence service;
- regional relay infrastructure;
- hosted policy authoring/distribution UI;
- approval workflow UI and orchestration;
- hosted audit search and retention;
- billing and subscription logic;
- support/admin tooling;
- SaaS database migrations;
- production deployment configuration;
- internal observability and runbooks.

The hosted service consumes public Community protocols and contracts rather than replacing them with an incompatible private protocol.

## Dependency direction

The dependency direction is one-way:

```text
optional private Cloud
        |
        | consumes public contracts
        v
public protocol + tool contracts
        |
        v
public Edge / Agent / adapters / local policy
```

The reverse dependency is forbidden.

Specifically:

- public Edge must not import Cloud code;
- public Agent must not import Cloud code;
- Community builds must not need private package registries;
- local policy must not call Cloud to decide whether a locally allowed Community operation can execute;
- Cloud unavailability must not disable documented self-hosted operation;
- hosted policy may narrow local policy but may never widen it;
- protocol schemas required at the customer trust boundary must remain public and versioned.

## Protocol ownership

The following interfaces belong here and remain public:

- agent protocol schemas;
- connector/relay messages crossing into customer-controlled Edge;
- MCP tool request/response contracts;
- policy bundle schema consumed by Edge;
- audit event schema shared with Edge;
- capability identifiers;
- normalized error codes;
- compatibility/version-negotiation rules.

Cloud-internal messages may remain private only when they never cross the customer trust boundary.

## Licensing

Community is MIT licensed unless a deliberate future licensing review changes the license prospectively.

A licensing change must not silently turn already-open customer-side trust components into hosted-only dependencies.

Third-party dependencies must remain compatible with public distribution.

## Public-release hygiene

Before changing repository visibility to public:

1. review the reachable repository history for credentials, private infrastructure identifiers, and internal-only material;
2. confirm `.env`, `.env.tunnel`, runtime state, logs, binaries, and local editor files are ignored;
3. confirm examples contain placeholders only;
4. remove duplicated website/marketing artifacts whose source of truth is another public repository;
5. keep Cloud-only implementation out of this repository;
6. ensure `README.md`, `SECURITY.md`, `CONTRIBUTING.md`, support guidance, issue templates, CI, and license are public-ready;
7. keep CI green on the release candidate;
8. make support status and pre-1.0 limitations explicit;
9. publish a tagged prerelease only after an external self-host smoke test.

No real credential may appear in examples, fixtures, release artifacts, issue templates, or reachable Git history.

## Website

`tetherbound.dev` is the product site and is maintained separately from this runtime repository.

The website may link directly here as the first-class self-hosted Community project. The website and repository must distinguish implemented functionality from roadmap/hosted concepts.

## Security rationale

The split is primarily a trust decision, not a monetization trick.

Users should be able to audit the source that:

- joins or observes their tailnet;
- verifies machine identity;
- stores local node state;
- evaluates local permissions;
- opens SSH/agent sessions;
- invokes operating-system APIs;
- reads logs/process/service data;
- enforces time/output bounds;
- executes Operator commands;
- eventually executes approved privileged actions.

Publishing billing logic, SaaS dashboards, deployment topology, or internal support tooling does not materially improve auditability of that machine-access boundary.

## Self-hosted compatibility

Community remains independently functional without Cloud connectivity.

The public protocols should keep alternate/self-built control planes possible in principle. TetherBound does not need to open-source every future hosted collaboration or billing feature for the self-hosted runtime to remain genuine open source.

Enterprise self-hosting may become a supported product later, but the baseline remains:

- Community runtime is public;
- protocols crossing the trust boundary are public;
- users control their Tailscale identity and local policy;
- optional Cloud provides convenience rather than permission to operate.

## Contribution flow

Public contributors work directly in this repository.

When a change affects both public and private code:

1. define and land the public contract here first;
2. maintain compatibility for the documented support window;
3. update private Cloud to consume the public change;
4. never require a private PR to compile or test this repository.

Security fixes may be developed privately under responsible disclosure, then released publicly with coordinated patched code and an advisory.

## Release gates

Before the first public tagged release:

- no credentials or private infrastructure identifiers are found in the public candidate/history review;
- CI builds without private dependencies;
- source corresponding to released Edge/Agent binaries is public;
- protocol/tool schemas used by released binaries are public;
- security, contribution, and support policies are present;
- Linux support status is accurate;
- Operator mode is explicitly labeled experimental until design-partner smoke testing succeeds;
- Windows/macOS support is described as roadmap rather than shipped functionality.

Before hosted alpha:

- hosted implementation is separate from this repository;
- Cloud-to-Edge protocol used in alpha is public and versioned;
- Cloud cannot bypass local Edge policy;
- Community remains usable without Cloud connectivity;
- tenant/isolation and relay threat models are reviewed separately.

## Decision summary

1. `fijimunkii/chatgpt-plugin-tailscale` is the canonical public self-hosted TetherBound Community repository.
2. TetherBound remains the product/brand; the descriptive repository name remains first-class for discovery.
3. Customer-side Edge, Agent, protocols, adapters, setup tooling, local policy, and tests stay open source here.
4. Optional hosted Cloud implementation stays separate/private.
5. Public code never depends on private Cloud code.
6. Protocols crossing the customer trust boundary remain public and versioned.
7. Hosted policy may narrow but never widen local policy.
8. Community remains useful without TetherBound Cloud.
