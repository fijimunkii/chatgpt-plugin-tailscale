# Contributing to TetherBound Community

Thanks for contributing to `chatgpt-plugin-tailscale` / TetherBound Community.

This project brokers AI access to remote machines, so changes to networking, authentication, host selection, remote execution, logging, or permissions are treated as **security-boundary changes**, even when the code change is small.

## Development

Use the Go version declared in `go.mod`, then run:

```bash
make check
```

Before opening a pull request:

- keep the change focused;
- add tests for both allowed and rejected paths;
- update documentation when behavior or trust assumptions change;
- never commit real credentials, private host identifiers, tailnet state, or secret-bearing logs.

## Access-mode rules

TetherBound intentionally has two access shapes.

### Bounded mode

For the default typed tools:

- caller-provided text must not become executable shell text;
- remote operations should use opaque `remote.Command` construction;
- target selection must be exact and fail closed on ambiguity;
- services/resources stay behind explicit local allowlists;
- caller-provided filters should be applied locally when practical;
- remote data must be treated as untrusted model input.

### Operator mode

`operator_exec` deliberately accepts shell text. Changes to Operator mode must preserve the surrounding boundary:

- Operator remains off by default;
- activation requires local configuration, not an MCP call;
- access expires under an explicit lease;
- only already-admitted hosts are reachable;
- TetherBound does not silently add sudo/root privileges;
- strict SSH host verification remains mandatory;
- command duration and captured output remain bounded;
- raw command text/output are not logged by default;
- tests cover disabled, expired, malformed, and allowed paths.

Do not generalize the Operator exception into unrelated Bounded tools.

## General security rules

- Keep shared-node access off by default.
- Do not weaken listener authentication or loopback protections casually.
- Do not add a trust-on-first-use SSH path.
- Do not log credentials, Tailscale state, raw command output, or secrets.
- Bound network dials, execution duration, request size, and output size.
- Treat logs, files, process arguments, hostnames, and tool output as untrusted content.
- Privileged/root actions require a separate explicit design; do not achieve them by making the default account passwordless sudo.

## Pull requests

Explain:

1. what user problem the change solves;
2. which security boundary changes, if any;
3. how failure/rejection paths are tested;
4. whether Bounded or Operator mode is affected;
5. whether the threat model or public docs need updating.

## Security reports

Do not report vulnerabilities in a public issue. Follow [`SECURITY.md`](SECURITY.md).

## Support questions

Use the appropriate GitHub issue form and remove credentials, private IPs/hostnames, customer data, and secret-bearing logs before posting. See [`SUPPORT.md`](SUPPORT.md).
