# Support

`chatgpt-plugin-tailscale` / TetherBound Community is a pre-1.0 open-source project. Support is best-effort and community-oriented; there is no SLA for the self-hosted edition.

## Where to ask

Use GitHub Issues for:

- bootstrap/install problems;
- Tailscale policy or target-admission questions;
- bounded-tool behavior;
- Operator-mode behavior;
- compatibility problems;
- feature requests.

Choose the closest issue form so the report includes the information maintainers need.

## Before posting

Run:

```bash
./bootstrap --check
```

and include the non-secret parts of the result that matter to the problem.

Also include, when relevant:

- OS and architecture of the gateway host;
- OS of the target host;
- Tailscale version;
- Go version;
- whether you are using Bounded or Operator mode;
- the exact TetherBound commit/release;
- the failing tool name;
- a redacted error message.

## Never post

Do not put any of the following in a public issue:

- Tailscale auth keys or OAuth secrets;
- OpenAI/API keys or tunnel credentials;
- MCP bearer tokens;
- `tsnet` state files;
- SSH private keys;
- private customer data;
- unredacted private host inventories/IPs if you consider them sensitive;
- logs or command output containing credentials/secrets.

Replace sensitive identifiers with stable placeholders when needed to explain a topology.

## Security issues

Do not use a public issue for a vulnerability or suspected boundary bypass. Follow [`SECURITY.md`](SECURITY.md).

## Hosted product

TetherBound Cloud is a separate future/optional product direction. Community issues should remain focused on the self-hosted runtime and its public protocols. Hosted-service support terms will be defined separately if/when Cloud is released.
