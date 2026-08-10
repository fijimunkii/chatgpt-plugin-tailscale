# TetherBound product positioning

Status: working positioning for product validation  
Last updated: 2026-08-10

This document defines how TetherBound should describe itself while the product is
still being validated. It is intentionally stricter than a marketing brainstorm:
implemented capabilities, roadmap capabilities, and hypotheses must remain
visibly separate.

## Brand

**Name:** TetherBound  
**Primary line:** **Give AI access. Keep it bounded.**

Supporting description:

> TetherBound gives AI assistants controlled, auditable access to private
> infrastructure without public management ports, shared SSH keys, or an
> unrestricted shell.

The name should carry two ideas:

- **Tethered:** connected, observable, revocable, and attached to an explicit
  trust boundary.
- **Bounded:** limited by host, identity, tool, service, data, output, time, and
  eventually approval policy.

Do not explain the name at length on the homepage. The product behavior should
make the metaphor obvious.

## Category

Primary category language:

> **Private infrastructure access for AI**

Secondary category language:

> **A bounded operations layer between AI assistants and private systems**

Avoid positioning the product primarily as:

- an SSH client;
- an MCP wrapper;
- a VPN;
- an RMM product;
- an AI SRE replacement;
- a generic agent framework.

SSH, MCP, Tailscale, gRPC, and platform APIs are implementation details that
belong in technical material. The user-facing problem is that an AI assistant
cannot safely inspect systems that are intentionally private.

## Problem statement

People increasingly use AI assistants for debugging, operations, and technical
analysis, but their most important machines are often intentionally unreachable
from the public internet. The common workarounds are poor trust boundaries:

- expose a management endpoint publicly;
- put a reusable SSH key into third-party configuration;
- grant a generic remote shell;
- copy logs and command output manually into a chat;
- assemble multiple identity, tunnel, policy, and MCP components by hand.

TetherBound should make the safe path easier than the unsafe shortcut.

## Core value proposition

### Functional

Ask an AI assistant questions about real private machines from the AI interface
you already use.

Examples:

- Which allowed machines are online?
- Is a server running low on disk?
- Why is an allowlisted service unhealthy?
- What changed around a recent reboot?
- Show bounded recent errors for an approved service.
- Which processes are consuming the most memory?

### Security

The assistant receives capabilities, not a shell.

A request must pass multiple independent boundaries:

1. caller/workspace authorization where applicable;
2. network identity and reachability policy;
3. local TetherBound host and tool policy;
4. exact tool schema and bounds;
5. target/platform authorization;
6. future approval policy for writes.

### Operational

The Community edition codifies a difficult setup while leaving the entire edge
self-hosted. The proposed Cloud edition removes platform plumbing while keeping
the final trust decision at the customer edge.

## Primary audiences

### 1. Self-hosters and homelab operators

They already run Tailscale, Linux servers, NAS devices, Docker hosts, or mixed
home infrastructure. They are comfortable installing software but do not want
to give an AI agent an unrestricted shell.

Best message:

> Ask ChatGPT about your private machines without opening them to the internet.

### 2. Independent developers and small technical teams

They operate VPSs, staging/production systems, remote build hosts, GPU machines,
or small internal fleets. They value diagnosis from the same AI surface they use
for development.

Best message:

> Give your AI assistant the visibility it needs without handing it your SSH
> keys.

### 3. Consultants, agencies, and MSP-adjacent operators

They manage several environments and need strict separation, explicit host
selection, auditability, and eventually approval workflows.

Best message:

> Define exactly what AI can inspect in each environment and keep the boundary
> enforceable at the edge.

### 4. Infrastructure and SRE teams

Longer-term audience. They care less about setup convenience and more about
identity, policy, audit, approvals, retention, and integration with existing
operational systems.

Best message:

> A typed, policy-enforced operations layer for AI—not another unrestricted
> remote shell.

## Messaging hierarchy

### Homepage headline

> **Give AI access. Keep it bounded.**

### Homepage subhead

> TetherBound connects AI assistants to the systems you already run while
> keeping access narrow, explicit, and auditable. No public SSH ports. No shared
> private keys. No unrestricted shell.

### Three proof points

1. **No unrestricted shell.** Caller text does not become an arbitrary command.
2. **Private by default.** The current Community path keeps target connectivity
   on the tailnet and uses an outbound MCP tunnel.
3. **Local policy stays authoritative.** A hosted service may narrow local
   permissions but must never widen them.

### Product distinction

> Most remote-management bridges start with credentials and end with a shell.
> TetherBound starts with the exact operation you are willing to allow.

## Community vs Cloud

### TetherBound Community

Promise:

> Own the whole boundary.

Current product:

- open-source gateway;
- Linux read-only target inspection through Tailscale SSH;
- seven typed MCP tools;
- local host/service allowlists and hard bounds;
- idempotent setup workflow;
- OpenAI Secure MCP Tunnel integration for ChatGPT development use.

Roadmap:

- native cross-platform gateway packaging;
- Windows/macOS/Linux host agent;
- normalized cross-platform tool outputs;
- signed binaries and installers.

Community should remain useful without a TetherBound account or TetherBound
Cloud.

### TetherBound Cloud

Promise:

> The easy path to the same hard boundary.

Hypothesized value:

- stable hosted MCP endpoint and user authentication;
- edge connector pairing and health;
- centralized policy that cannot widen the local policy;
- audit metadata and team controls;
- simpler connector updates and lifecycle;
- public ChatGPT/Codex distribution when platform requirements are met;
- later approval workflows for typed actions.

Cloud must not be described as available until a usable hosted alpha exists.
Use **in development**, **proposed**, **early access**, or **validation** depending
on context.

## Claims discipline

Public copy must distinguish three states.

### Implemented

Use language such as:

- available now;
- working;
- implemented;
- tested end-to-end.

Current examples:

- Linux read-only inspection through Tailscale SSH;
- seven fixed MCP tools;
- local host and exact service allowlists;
- bootstrap workflow;
- private ChatGPT connectivity through OpenAI Secure MCP Tunnel.

### Committed roadmap

Use language such as:

- planned;
- roadmap;
- designed for;
- proposed architecture.

Examples:

- Windows host agent;
- macOS host agent;
- native installers and services;
- normalized cross-platform contracts.

### Product hypothesis

Use language such as:

- exploring;
- validating;
- proposed Cloud edition;
- early-access concept.

Examples:

- hosted pricing;
- customer willingness to run an edge connector;
- centralized team policy;
- default retention expectations;
- willingness to approve future write actions.

Never present a roadmap item as an existing security guarantee.

## Security language

Preferred:

- bounded;
- controlled;
- typed;
- explicit;
- least privilege;
- customer-controlled edge;
- local policy;
- exact allowlist;
- read-only;
- auditable;
- revocable;
- no generic shell.

Use cautiously:

- zero trust — only when discussing specific architecture properties rather than
  as a broad marketing label;
- secure — pair it with the mechanism;
- private — clarify whether it means network reachability, retention, or data
  custody.

Avoid absolute claims:

- perfectly secure;
- impossible to compromise;
- zero risk;
- data never leaves your network;
- we cannot see your data.

For the proposed Cloud edition, raw result data necessarily transits the hosted
relay on its way to the AI client. The design goal is **no raw output retention
by default**, not "the cloud never handles data."

## Objections and responses

### “Isn’t this just SSH from ChatGPT?”

No. SSH is the first Linux transport. The product boundary is the typed policy
layer: the assistant receives explicit capabilities rather than a generic remote
command channel. Windows and mainstream macOS use a planned native agent.

### “Why not give the AI a restricted SSH key?”

A restricted key still makes command policy, host selection, output bounds,
audit semantics, cross-platform behavior, and future approvals somebody else’s
problem. TetherBound makes those first-class product concepts.

### “Why do I need TetherBound if I already have Tailscale?”

Tailscale provides network identity, reachability, and strong access controls.
TetherBound uses those boundaries and adds an AI-facing typed tool surface,
local host/tool policy, normalized output, and the operational integration layer.

### “Does the AI get root / Administrator?”

Not in the read-only design. Linux uses a dedicated non-root account. The
Windows agent is designed around native read APIs and should not run as
LocalSystem merely to collect richer data.

### “Can it restart services?”

Not in the initial read-only release. Future writes are planned as individually
typed prepare → approve → execute actions. Arbitrary shell remains out of scope.

### “Does Cloud put a node inside my tailnet?”

The preferred hosted design does not. A customer-controlled edge connector joins
the tailnet locally and opens an outbound authenticated relay session. A fully
hosted tailnet node is explicitly deferred because it creates a larger trust
boundary.

## Visual tone

TetherBound should feel:

- technical but not hacker-themed;
- calm rather than alarmist;
- precise rather than futuristic;
- infrastructure-grade without looking like an enterprise firewall appliance;
- open and inspectable rather than mysterious.

Avoid padlock stock art, neon terminal screenshots, matrix imagery, generic AI
brains, and claims that imply autonomous control. Diagrams should emphasize the
bounded path and the customer-controlled edge.

## Product vocabulary

Preferred product names:

- **TetherBound Community** — open-source/self-hosted edition.
- **TetherBound Cloud** — proposed hosted control plane.
- **TetherBound Edge** — customer-side connector in the hosted model.
- **TetherBound Agent** — cross-platform host agent.
- **TetherBound Policy** — generic reference to the local policy model; not
  necessarily a separately sold product.

Do not rename the repository, binary, environment variables, or Tailscale tags
until a dedicated migration change is planned. Brand changes should not break
the working proof of concept.
