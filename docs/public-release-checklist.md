# Public release checklist

This is the maintainer checklist for opening `fijimunkii/chatgpt-plugin-tailscale` and cutting the first Community prerelease.

## Repository hygiene

- [ ] Review reachable Git history for credentials and private infrastructure identifiers.
- [ ] Confirm `.env`, `.env.tunnel`, `data/`, logs, binaries, and editor state are ignored.
- [ ] Confirm all key/tunnel/IP examples are obvious placeholders or documentation ranges.
- [ ] Confirm no Cloud implementation or internal operations material is present.
- [ ] Confirm the duplicate website source has been removed; `tetherbound-site` is the site source of truth.
- [ ] Confirm repository description/homepage/topics are set for discoverability.

Recommended description:

> Self-hosted MCP gateway for giving ChatGPT, Codex, and other AI clients bounded or operator access to machines over Tailscale. TetherBound Community.

Recommended homepage:

```text
https://tetherbound.dev
```

Recommended topics:

```text
chatgpt
tailscale
mcp
model-context-protocol
openai
codex
self-hosted
homelab
infrastructure
ssh
```

## Security and Community

- [ ] `SECURITY.md` is present and private vulnerability reporting is enabled in repository settings when available.
- [ ] `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, and `SUPPORT.md` are present.
- [ ] Bug, self-hosting, and feature issue forms are enabled.
- [ ] Blank issues are disabled to reduce accidental secret disclosure.
- [ ] Branch protection/ruleset requires CI on `main`.
- [ ] Dependabot is enabled.

## Build and tests

- [ ] CI is green.
- [ ] Go formatting, tests, vet, and production build pass.
- [ ] Bootstrap idempotency tests pass.
- [ ] Linux amd64 and arm64 cross-builds pass.
- [ ] macOS amd64 and arm64 gateway cross-builds pass before advertising binaries for those platforms.
- [ ] Container build passes.

## Live smoke tests

- [ ] Fresh self-host install on a machine that did not develop the project.
- [ ] Fresh Tailscale target enrollment.
- [ ] ChatGPT custom MCP app tool scan through Secure MCP Tunnel.
- [ ] `list_hosts`, `inspect_host`, `system_summary`, and `disk_usage` succeed.
- [ ] Service tools fail closed for non-allowlisted units.
- [ ] Operator mode is absent/disabled by default.
- [ ] `make operator-up` exposes `operator_exec` only for the active local lease.
- [ ] Operator execution runs as the dedicated non-root account.
- [ ] Operator lease expiration rejects new commands.
- [ ] Raw Operator command text/output is not present in gateway logs.

## Documentation claims

- [ ] Linux target support is labeled working.
- [ ] Operator mode is labeled experimental until external smoke testing succeeds.
- [ ] Windows/macOS target Agents are labeled roadmap.
- [ ] TetherBound is clearly described as independent from Tailscale Inc. and OpenAI.
- [ ] OpenAI UI terminology matches current custom MCP app / Secure MCP Tunnel documentation.

## First tag

Recommended first public tag:

```text
v0.1.0-alpha.1
```

Before tagging:

- [ ] merge the release candidate to `main`;
- [ ] run CI on `main`;
- [ ] complete the external self-host smoke test;
- [ ] update `CHANGELOG.md` with the release date;
- [ ] create release notes emphasizing pre-1.0 status and Operator-mode risk;
- [ ] link the public repository from `tetherbound.dev`.

Do not call the release stable/GA until the external install path and Operator-mode safety assumptions have been exercised outside the development environment.
