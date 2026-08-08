# Contributing

This project brokers access to remote hosts, so changes are reviewed as security-boundary changes rather than ordinary feature additions.

## Development

Use the Go version declared in `go.mod`, then run:

```bash
make check
```

Keep changes small and include tests for both the allowed path and the rejection path.

## Design rules

- Do not add an MCP parameter that is executed as a shell command.
- Construct remote operations through the opaque `remote.Command` type.
- Keep target selection exact and fail closed on ambiguity.
- Keep service or resource selectors behind explicit local allowlists.
- Apply caller-provided text locally whenever possible; never interpolate it into a remote shell.
- Treat all host-derived values and output as untrusted data.
- Do not log credentials, shell commands, stdout, or stderr.
- Bound network dials, command duration, request size, and output size.
- Do not add write actions without a separate preparation and approval design.

Report security vulnerabilities through the process in [`SECURITY.md`](SECURITY.md), not a public issue.
