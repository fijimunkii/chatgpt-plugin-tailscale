# Security policy

TetherBound Community connects AI clients to private infrastructure. Please treat potential authentication, authorization, host-selection, remote-execution, secret-handling, or isolation failures as security-sensitive.

## Reporting a vulnerability

**Do not open a public GitHub issue for a vulnerability.**

Use GitHub's private vulnerability reporting / security-advisory flow for this repository when available. If private reporting is not available in the UI, contact the repository owner privately rather than publishing exploit details.

Include only the minimum information needed to reproduce the issue:

- affected commit or release;
- Bounded or Operator mode;
- deployment mode and relevant **non-secret** configuration;
- minimal reproduction steps;
- expected and observed security boundary;
- impact assessment;
- whether you believe exploitation has occurred.

Do not send live Tailscale auth keys, OAuth client secrets, OpenAI/API keys, MCP bearer tokens, `tsnet` state, private SSH material, customer data, or unredacted secret-bearing command output unless a secure exchange method has been explicitly arranged.

## High-priority vulnerability classes

Examples include:

- reaching a host that does not satisfy local admission policy;
- bypassing Tailscale/SSH host verification;
- enabling or extending Operator mode remotely without local authorization;
- escaping the configured OS-account boundary through TetherBound itself;
- arbitrary shell injection through a Bounded-mode tool;
- exposing credentials or raw remote output through logs;
- listener/authentication bypass that exposes the MCP endpoint unintentionally;
- cross-tenant or Cloud-to-Edge authorization bypass once hosted components exist.

A model following malicious instructions contained in content it was legitimately allowed to read is an important product risk, but is not automatically a software vulnerability. Reports showing a concrete TetherBound boundary bypass caused by prompt injection are security-relevant.

## Supported versions

The project is pre-1.0. Until tagged support policy is established, security fixes target the latest release candidate / default branch. Users should update promptly after a security fix.

## Disclosure

Please allow reasonable time for diagnosis, patching, release preparation, and coordinated disclosure before publishing details. The project will credit reporters who want attribution when a public advisory is issued.
