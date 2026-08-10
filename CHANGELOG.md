# Changelog

All notable public changes to TetherBound Community / `chatgpt-plugin-tailscale` will be documented here.

The project is pre-1.0. Until the first tagged release, current work remains under **Unreleased**.

## Unreleased

### Added

- Embedded `tsnet` gateway identity for Tailscale-native connectivity.
- Strict Tailscale-distributed SSH host-key verification.
- Independent tag/host admission policy and shared-node exclusion by default.
- Seven typed, read-only MCP tools for Linux inspection.
- Idempotent bootstrap for target setup, gateway enrollment, and Secure MCP Tunnel configuration.
- Bounded command duration, request size, output size, log lines, and process rows.
- Experimental Operator mode with arbitrary shell execution behind local opt-in, admitted hosts, a dedicated OS account, and a time-bounded lease.
- Public threat model, architecture, cross-platform roadmap, tool contracts, contribution policy, security policy, and support guidance.

### Changed

- Self-hosted Community is now the first-class product path for this repository.
- `chatgpt-plugin-tailscale` remains the canonical descriptive open-source distribution name while TetherBound remains the product/brand.

### Security

- Operator mode remains disabled by default and does not add sudo/root privileges.
- Raw command text and remote output are not logged by default.
