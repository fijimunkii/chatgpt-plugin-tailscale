# Operator mode

Status: experimental / design-partner preview  
Last updated: 2026-08-10

Operator mode is TetherBound's deliberately broad access profile for users who want an AI assistant to work freely inside a machine rather than select individual read-only tools.

The boundary moves outward from **operation-by-operation permission** to **an explicitly admitted host + a dedicated OS account + a time-bounded gateway process**.

## What it enables

When Operator mode is locally enabled, TetherBound registers one additional MCP tool:

```text
operator_exec(host, command, cwd?)
```

The command is arbitrary POSIX shell text. It runs through `/bin/sh -lc` on the selected Linux host as the configured `SSH_USER`.

The tool can therefore, within that user's ordinary OS permissions:

- create, edit, and delete files;
- clone and modify repositories;
- run build/test/development tools;
- start or stop user-owned processes;
- use Docker only if that OS account already has Docker access;
- make network requests the OS account is permitted to make;
- invoke any other command already available to that account.

This is intentionally not a typed read-only operation.

## Local activation

Operator mode is **off by default** and is intentionally not persisted in `config/example.env`.

For a local design-partner session:

```bash
OPERATOR_MODE=true make up
```

The default lease is one hour. The tool refuses new commands after the lease expires.

An optional process-local override can change the lease:

```bash
OPERATOR_MODE=true OPERATOR_LEASE_DURATION=30m make up
```

Accepted lease duration is 5 minutes through 24 hours.

### Current v0.1 lease caveat

The lease is tied to one gateway process start. If an operator deliberately starts a new gateway process again with `OPERATOR_MODE=true`, that new process receives a fresh lease. For the design-partner preview, do not persist `OPERATOR_MODE=true` in `.env` or a service manager. A later release should use an absolute locally issued lease so automatic process restarts cannot extend access.

## What remains bounded

Operator mode does **not** bypass the rest of TetherBound's transport and host boundary.

A command still requires all of the following:

1. the gateway is currently inside a locally enabled Operator lease;
2. the requested host resolves exactly through the TetherBound catalog;
3. the host matches `TS_ALLOWED_TAGS` or `TS_ALLOWED_HOSTS`;
4. shared nodes remain excluded unless explicitly enabled;
5. the host is online and ready for Tailscale SSH;
6. the host's Tailscale-distributed SSH host key verifies;
7. Tailscale network and SSH policy permit the dedicated gateway identity;
8. execution stays inside the configured SSH command timeout;
9. captured stdout/stderr stays inside the configured byte limit.

Operator mode does not create another public network listener or distribute a reusable SSH private key.

## Privilege boundary

TetherBound does not add privilege elevation.

The initial implementation runs as the existing dedicated SSH account (currently `chatgpt` by default). If that account is non-root and is not a sudoer, Operator mode is non-root.

Do **not** make the default account `NOPASSWD: ALL` merely to make Operator mode more capable. Root/Administrator delegation should become a separate explicit access level with its own lease and approval semantics.

Remember that some existing group memberships are effectively privileged. For example, Docker daemon access commonly provides a path to host-level control. Treat membership in `docker`, hypervisor, device-management, or similar groups as elevated access when evaluating a target.

## MCP annotations

`operator_exec` is advertised as:

- not read-only;
- not idempotent;
- potentially destructive;
- open-world.

These annotations help MCP clients present appropriate UX, but they are descriptive metadata, not the security boundary. Enforcement remains local in TetherBound and Tailscale.

## Logging and audit

The gateway records operation metadata such as:

- operation ID;
- operation name (`operator_exec`);
- admitted host identity;
- configured SSH user;
- exit code;
- duration;
- truncation state;
- captured output byte count.

The gateway deliberately does **not** log raw command text or raw command output by default. Commands often contain credentials, tokens, source snippets, URLs, or other sensitive values. Richer audit capture should be an explicit policy choice rather than an accidental secret-retention system.

## Prompt-injection risk

Operator mode materially increases the consequence of model mistakes and prompt injection. Logs, files, repository text, web responses, process arguments, issue content, and other machine data can contain malicious instructions intended to influence an AI assistant.

For Operator targets:

- use a dedicated OS account;
- avoid placing human SSH keys and broad cloud credentials in that account;
- prefer development/staging machines before production;
- keep root separate;
- use host admission to keep the delegated blast radius explicit;
- expire the Operator session when broad access is no longer needed.

The product goal is not to claim arbitrary shell execution is harmless. The goal is to make the delegated blast radius understandable and revocable.

## Product model

TetherBound now has two useful permission shapes:

### Bounded mode

Fixed typed operations, exact service allowlists, read-only inspection, and narrow privileges.

### Operator mode

Broad shell access within a selected machine/account boundary for users who explicitly prefer autonomy over operation-level controls.

This produces a simpler product principle:

> You choose the boundary. TetherBound keeps the AI inside it.
