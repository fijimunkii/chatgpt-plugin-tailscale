# Repository and open-source boundary

Status: normative product and architecture decision  
Last updated: 2026-08-10

TetherBound uses an **open edge, private hosted control plane** model. The code that reaches, authenticates to, inspects, or eventually operates customer machines must remain open source and independently usable. The hosted SaaS may remain proprietary because it provides convenience, collaboration, policy management, relay infrastructure, billing, and operations rather than the trusted machine-access primitive itself.

This document is normative for the roadmap and cross-platform architecture. If another planning document conflicts with this boundary, this document wins until an explicit architecture decision changes it.

## Repository model

### Incubation repository — private

The current `fijimunkii/chatgpt-plugin-tailscale` repository remains private during incubation.

Its purpose is to preserve prototype history while the project finishes:

- the Linux proof of concept;
- cross-platform abstractions;
- the Windows host-agent MVP;
- packaging and setup decisions;
- branding and validation work;
- the clean extraction plan for the public Community repository.

The incubation repository is **not** the permanent Community distribution repository. It may contain planning material and early hosted-product work that should not automatically become part of the public source history.

### Community repository — public

Before the first broadly promoted Community release, create a clean public repository, expected to be named approximately:

```text
tetherbound/tetherbound
```

or, while the project remains under a personal GitHub account:

```text
fijimunkii/tetherbound
```

The public repository owns all code that participates in the customer-side trust boundary:

- TetherBound Edge / gateway;
- TetherBound Agent;
- Linux, Windows, and macOS host adapters;
- Tailscale SSH transport;
- agent transport;
- local host discovery and stable identity;
- local policy evaluation;
- local audit journal;
- normalized MCP tool contracts;
- agent and connector protocol definitions;
- pairing, certificate, and local revocation logic;
- bootstrap/setup tooling;
- installers and service wrappers;
- update verification logic;
- conformance and security tests;
- Community MCP endpoint and Secure MCP Tunnel integration;
- public architecture, threat-model, security, and operator documentation.

The public repository must be usable without TetherBound Cloud. No Community feature may require a private package, private API, hosted license check, or proprietary runtime component to perform its documented local function.

### Cloud/control-plane repository — private

Hosted implementation lives in a separate private repository, expected to be named approximately:

```text
tetherbound/cloud
```

or:

```text
tetherbound/control-plane
```

It may contain:

- public hosted MCP resource server;
- OAuth/OIDC integration;
- workspace, membership, and RBAC implementation;
- connector registry and presence service;
- regional relay infrastructure;
- hosted policy authoring and distribution UI;
- approval workflow UI and orchestration;
- hosted audit search and retention services;
- billing and subscription logic;
- support/admin tooling;
- SaaS database migrations;
- production deployment configuration;
- internal observability and runbooks;
- plugin review/demo environment.

The private repository must consume the public edge/protocol contracts rather than reimplementing them in an incompatible private form.

## Dependency direction

The dependency rule is deliberately one-way:

```text
private Cloud
     |
     | depends on published/open contracts
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
- local policy must not call Cloud to decide whether a locally allowed Community operation can execute;
- protocol schemas needed for interoperability must not exist only in the private repository;
- the hosted service may narrow local policy but may never widen it;
- Cloud unavailability must not disable documented Community-only operation;
- a Community build must be reproducible without credentials for private package registries.

CI for the public repository should include a dependency-boundary check where practical.

## Protocol ownership

The following belong in the public repository and are versioned as public interfaces:

- agent protocol schemas;
- connector/relay message schemas required by customer-controlled Edge;
- MCP tool request and response contracts;
- policy bundle schema consumed by Edge;
- audit event schema shared with Edge;
- capability identifiers;
- normalized error codes;
- compatibility/version-negotiation rules.

The hosted relay implementation can remain private, but the messages that cross from Cloud into customer-controlled Edge must be inspectable and documented.

Private protocol extensions are allowed only when they are Cloud-internal and never cross the customer trust boundary.

## Licensing

The current proof of concept is MIT licensed. The intended Community repository should retain a permissive open-source license unless a deliberate licensing review changes that decision before public extraction.

Any change away from a permissive license must be explicit and must not silently make previously open customer-side trust components source-available-only or hosted-dependent.

Third-party dependencies must be reviewed for compatibility with public distribution.

## Public extraction from the incubation repository

Do not simply flip the current incubation repository from private to public.

Use an intentional extraction before Community launch:

1. Freeze a known-good cross-platform Community baseline.
2. Inventory every tracked file for secrets, private identifiers, internal notes, and Cloud-only material.
3. Run automated secret scanning across the full candidate public history or choose a clean-history extraction.
4. Create the public `tetherbound` repository.
5. Prefer a curated/squashed initial history unless preserving prototype history provides clear value and has been reviewed completely.
6. Move public runtime code, tests, packaging, protocol definitions, and appropriate documentation.
7. Replace prototype naming such as `chatgpt-plugin-tailscale` in user-facing package/service identifiers through a documented compatibility migration.
8. Establish public issue templates, contribution guidance, security disclosure instructions, CODEOWNERS, release signing, and CI.
9. Publish reproducible signed artifacts from the public repository.
10. Keep the incubation repository private and archived as historical/internal material once the migration is complete.

No credential from the incubation environment may be copied into the public repository, release artifacts, examples, test fixtures, or Git history.

## Website and documentation

`tetherbound.dev` is the product domain.

During incubation, the validation site may be published from the private incubation repository if the GitHub plan supports private-repository Pages. Otherwise, use a small public site repository or wait for the public Community repository.

Long term, public product/security/Community documentation should be generated or deployed from public source where practical. Cloud-specific operational documentation and internal runbooks remain private.

The website must distinguish:

- **Working / Community** features;
- **Roadmap** features;
- **Cloud / validation** features.

Repository visibility must never be used to imply a feature is shipped.

## Security rationale

The split is not primarily a monetization mechanism. It is a trust-boundary decision.

Users should be able to audit the code that:

- joins or observes their tailnet;
- accepts agent connections;
- verifies machine identity;
- stores local credentials or node state;
- evaluates local permissions;
- invokes operating-system APIs;
- reads logs/process/service data;
- enforces output bounds;
- executes future approved actions.

Keeping those components open reduces the need for customers to trust an opaque binary with privileged private-network access.

Conversely, publishing billing logic, SaaS dashboards, deployment topology, or internal support tooling does not materially improve the security auditability of the customer-side boundary and is not required for Community independence.

## Hosted self-hosting compatibility

The public protocol should make a compatible self-hosted control plane possible in principle, but TetherBound is not required to ship every proprietary Cloud feature as open source.

Enterprise self-hosting may be a supported product later. Until then:

- protocol compatibility is public;
- Community remains independently functional;
- unsupported third-party/self-built control planes are allowed to interoperate at their own risk;
- Cloud-specific UI, billing, and operations remain proprietary.

## Contribution flow

Public contributors work against the Community repository.

When a change affects both public and private code:

1. land or publish the public contract first;
2. maintain backward compatibility for the documented support window;
3. update the private Cloud implementation to consume the public change;
4. never require a private PR to compile or test the public repository.

Security fixes affecting the customer-side trust boundary should be developed privately under responsible disclosure when necessary, then released publicly with coordinated patched binaries and an advisory.

## Release gates

Before the public Community repository is announced:

- no secrets or private infrastructure identifiers exist in its reachable Git history;
- public CI builds without private dependencies or credentials other than ordinary release-signing infrastructure;
- Edge and Agent source used to produce release binaries is present in the public repository;
- protocol/tool schemas used by the released binaries are public;
- security and contribution policies are present;
- package/service naming migration from the prototype is documented;
- Linux proof-of-concept behavior remains supported;
- Windows MVP or its exact support status is documented without overclaiming.

Before hosted alpha:

- the Cloud repository is separate;
- the Cloud-to-Edge protocol used in alpha is public and versioned;
- the private Cloud implementation cannot bypass local Edge policy;
- Community functionality continues to operate without Cloud connectivity;
- tenant/isolation and relay threat models are reviewed separately from the public Edge threat model.

## Decision summary

1. The current repository remains a private incubation repository.
2. Community/Edge/Agent/protocol/local-policy code moves to a clean public `tetherbound` repository before broad Community release.
3. Hosted control-plane implementation lives in a separate private repository.
4. Code that reaches or operates customer machines stays open source.
5. Protocols crossing the customer trust boundary stay public and versioned.
6. Public code must never depend on private Cloud code.
7. Hosted policy may narrow but never widen local policy.
8. Community remains independently usable without TetherBound Cloud.
9. The current incubation repository should be archived privately after public extraction, not simply made public wholesale.
