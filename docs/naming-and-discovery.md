# Naming and discovery strategy

Status: product decision  
Last updated: 2026-08-10

## Canonical brand

The product and project brand is **TetherBound**.

Tagline:

> Give AI access. Keep it bounded.

User-facing releases, packages, documentation, and the future public Community repository should use the TetherBound name.

## Incubation repository

The private incubation repository intentionally retains its historical descriptive name:

```text
fijimunkii/chatgpt-plugin-tailscale
```

Do not rename this repository merely for branding consistency.

The name captures the project's origin and remains useful internal/historical context. The incubation repository is not the permanent public distribution surface and will remain private after Community extraction.

The word `plugin` in this repository name is historical. Current user-facing architecture and documentation should describe the integration as MCP-based rather than presenting TetherBound as an unrestricted ChatGPT plugin.

## Public Community repository

The future clean public runtime repository should use the canonical product name, expected to be:

```text
fijimunkii/tetherbound
```

or, if a TetherBound GitHub organization exists by then:

```text
tetherbound/tetherbound
```

This repository owns the public Edge, Agent, adapters, protocol/tool contracts, local policy, setup tooling, tests, packaging, and public operator/security documentation defined in `repository-boundary.md`.

## Discovery language

TetherBound should deliberately use descriptive phrases around the brand so people searching for the problem can find it.

Primary discovery phrases:

- ChatGPT + Tailscale
- ChatGPT Tailscale MCP
- Tailscale MCP server / gateway
- ChatGPT MCP
- Codex + Tailscale
- secure AI access to a tailnet
- bounded AI infrastructure access
- MCP gateway for private infrastructure

These phrases are discovery and explanatory language, not alternate product names.

A concise public description is:

> TetherBound is a security-first MCP layer for giving ChatGPT, Codex, and other MCP clients bounded access to machines over Tailscale.

## Brand boundary

Use:

- **TetherBound** as the product name;
- **Built on Tailscale** to describe the network/identity substrate;
- **ChatGPT, Codex, and other MCP clients** to describe compatible AI clients;
- explicit independent-project/trademark language where Tailscale is prominent.

Avoid:

- renaming the product to `ChatGPT Plugin for Tailscale`;
- presenting `chatgpt-plugin-tailscale` as the eventual public/canonical repository;
- using Tailscale branding as part of the TetherBound logo;
- wording that implies Tailscale or OpenAI affiliation, endorsement, or partnership;
- treating `plugin` as the primary current architecture term.

## Repository summary

```text
fijimunkii/chatgpt-plugin-tailscale   private incubation/history
fijimunkii/tetherbound                future public Community runtime
fijimunkii/tetherbound-site           public product/validation website
<tetherbound cloud repo>              future private hosted control plane
```

The descriptive incubation name and canonical TetherBound brand therefore coexist intentionally: one preserves origin and context; the other provides a durable product identity.
