# Self-host bootstrap guide

The supported first-run path is:

```bash
./bootstrap
```

The wizard coordinates the pieces required for the current Linux target path: Tailscale policy, target preparation, the embedded TetherBound gateway identity, OpenAI Secure MCP Tunnel, and ChatGPT custom MCP app connectivity. It is designed to be safe to rerun.

## What the wizard does

It:

1. checks local prerequisites;
2. shows the Tailscale policy fragment and opens the Access Controls page;
3. optionally configures a Linux target over an existing administrator SSH path;
4. creates or preserves `.env` for local gateway configuration;
5. creates or preserves `.env.tunnel` for Secure MCP Tunnel configuration;
6. creates/reuses the `tunnel-client` profile with health on `127.0.0.1:8081` while the MCP gateway uses `127.0.0.1:8080`;
7. optionally starts the gateway and tunnel.

It does **not** silently replace tailnet policy or assign machine tags. Those remain explicit administrative actions.

## Rerun behavior

Running `./bootstrap` again preserves existing matching resources:

- `.env` and `.env.tunnel` are updated by key rather than recreated blindly;
- repeated environment keys are collapsed;
- an existing `tsnet` identity under `TS_STATE_DIR` is reused;
- an existing target `chatgpt` account is preserved;
- journal group membership is preserved;
- `tailscale set --ssh` is safe to repeat;
- a matching tunnel profile is reused;
- a mismatched profile is replaced only after explicit approval or `--reconfigure-tunnel`.

Diagnostics are read-only:

```bash
./bootstrap --check
```

## Prerequisites

Gateway machine:

- Bash
- Go version declared in `go.mod`
- `curl`
- OpenAI `tunnel-client`
- `ssh` and `scp` only when using remote target setup

Target machine:

- Linux
- Tailscale installed and connected
- `systemd` for the current service-status/log tools
- an existing administrator path independent of the new `chatgpt` account

For ChatGPT, private/local MCP servers are connected through Secure MCP Tunnel. ChatGPT currently exposes custom MCP integrations as **apps** in developer mode; UI and plan availability can change, so consult current OpenAI documentation when the UI differs from this guide.

On macOS, if Gatekeeper blocks a downloaded `tunnel-client` binary, verify its checksum and approve that individual binary through **System Settings → Privacy & Security → Open Anyway**. Do not disable Gatekeeper globally.

## 1. Tailscale policy

The wizard displays [`tailscale-policy.hujson`](tailscale-policy.hujson).

Merge the relevant entries into your existing policy:

```jsonc
{
  "tagOwners": {
    "tag:chatgpt-ssh": ["autogroup:admin"],
    "tag:chatgpt-managed": ["autogroup:admin"]
  },
  "grants": [
    {
      "src": ["tag:chatgpt-ssh"],
      "dst": ["tag:chatgpt-managed"],
      "ip": ["tcp:22"]
    }
  ],
  "ssh": [
    {
      "action": "accept",
      "src": ["tag:chatgpt-ssh"],
      "dst": ["tag:chatgpt-managed"],
      "users": ["chatgpt"]
    }
  ]
}
```

The network grant permits the gateway identity to reach managed targets on TCP/22. The SSH rule permits the dedicated `chatgpt` OS account.

Tailscale rules are additive. Existing broad grants can widen effective reachability, so review the rest of the policy too.

## 2. Prepare a Linux target

Using an existing administrator SSH path:

```bash
./bootstrap --target ubuntu@server.example
```

To grant the dedicated account journal-read membership for `service_logs`:

```bash
./bootstrap --target ubuntu@server.example --with-journal
```

The helper:

- creates `chatgpt` only if missing;
- uses `/home/chatgpt` and `/bin/bash`;
- does not set a Linux password;
- optionally adds journal-read membership;
- runs `tailscale set --ssh`;
- prints the target's Tailscale address and next steps.

You can run it directly on a target:

```bash
sudo ./scripts/bootstrap-target.sh --with-journal
```

Read-only verification:

```bash
sudo ./scripts/bootstrap-target.sh --check --with-journal
```

Then apply `tag:chatgpt-managed` to the target in Tailscale Machines.

Manual Tailscale SSH compatibility test:

```bash
ssh chatgpt+password@100.x.y.z
```

Enter any non-empty value at the password prompt. The suffix still uses Tailscale identity; it does not authenticate against a Linux password.

Verify:

```bash
whoami
id
```

`whoami` should return `chatgpt`.

## 3. Enroll the embedded gateway

The gateway creates a separate Tailscale node named `chatgpt-ssh-gateway` by default. It is distinct from the ordinary Tailscale client running on the same machine.

For first setup, create a one-off Tailscale auth key carrying:

```text
tag:chatgpt-ssh
```

A non-reusable, non-ephemeral, short-lived key is appropriate. If device approval is enabled, pre-approve as appropriate for your environment.

The wizard stores the enrollment credential in the gitignored `.env` file with mode `0600` where supported. It can alternatively use a Tailscale OAuth client capable of creating auth keys with the required tag.

After successful enrollment, persistent node state lives under:

```text
data/tsnet
```

The gateway reuses that identity on restart. After confirming restarts work, a consumed one-off key can be cleared from `.env`:

```dotenv
TS_AUTHKEY=
```

Do not delete `data/tsnet` unless you intend to replace the gateway's Tailscale identity.

## 4. Configure OpenAI Secure MCP Tunnel

The wizard asks for the tunnel ID and the runtime credential used by `tunnel-client`.

They are stored only in the gitignored local file:

```text
.env.tunnel
```

The profile is created with the private MCP endpoint and a separate health listener:

```bash
tunnel-client init \
  --sample sample_mcp_remote_no_auth \
  --profile tailscale-ssh \
  --tunnel-id tunnel_... \
  --mcp-server-url http://127.0.0.1:8080/mcp \
  --health-listen-addr 127.0.0.1:8081
```

If a profile already matches, it is reused. To intentionally recreate a mismatched profile:

```bash
./bootstrap --reconfigure-tunnel
```

The existing profile is backed up before replacement.

## 5. Start TetherBound

### Bounded mode

```bash
make up
```

or:

```bash
./scripts/run-local.sh
```

### Operator mode

Operator mode is an explicit local design-partner feature:

```bash
make operator-up
```

The default lease is one hour. A shorter lease can be supplied for testing:

```bash
OPERATOR_LEASE_DURATION=30m make operator-up
```

If a gateway is already running, Operator startup fails rather than reusing a process whose access mode is ambiguous.

Status endpoints:

```text
Gateway health:       http://127.0.0.1:8080/healthz
Gateway readiness:    http://127.0.0.1:8080/readyz
Tunnel health:        http://127.0.0.1:8081/healthz
Tunnel readiness:     http://127.0.0.1:8081/readyz
Tunnel admin UI:      http://127.0.0.1:8081/ui
```

Local logs:

```text
data/logs/gateway.log
data/logs/tunnel-client.log
```

Raw Operator commands and remote output are not intentionally written to the gateway audit log.

## 6. Create the ChatGPT custom MCP app

Current ChatGPT terminology is **Apps**, not the historical Plugins name used by this repository.

With the gateway and Secure MCP Tunnel healthy:

1. enable developer mode in ChatGPT web if your plan/workspace requires it;
2. open **Settings → Apps** and choose **Create** (the exact placement may vary as the beta UI evolves);
3. configure the custom MCP app using the Secure MCP Tunnel connection;
4. scan/refresh the tool definitions;
5. confirm the expected tools appear.

Bounded mode should expose:

```text
list_hosts
inspect_host
system_summary
disk_usage
service_status
service_logs
process_list
```

An active Operator-mode gateway additionally exposes:

```text
operator_exec
```

A useful first prompt is:

```text
Using TetherBound, list the hosts I am allowed to inspect.
```

OpenAI currently notes that ChatGPT does not connect directly to a local MCP server; Secure MCP Tunnel is the supported path for a private-network/on-premises/developer-machine MCP server. Write/modify MCP capabilities such as Operator mode may also be subject to ChatGPT plan/workspace controls and confirmation behavior.

## Non-interactive bootstrap

For automation, provide values through the environment and explicitly accept or skip the manual policy step:

```bash
export TS_AUTHKEY='tskey-auth-...'
export CONTROL_PLANE_TUNNEL_ID='tunnel_0123456789abcdef0123456789abcdef'
export CONTROL_PLANE_API_KEY='sk-...'
export BOOTSTRAP_ACCEPT_POLICY=1

./bootstrap --non-interactive --skip-target --no-open
```

Those are placeholders. Never commit live values.

For shared or production deployment, prefer an external secret manager over long-lived plaintext environment files.

## Options

```text
--check                 Read-only diagnostics
--non-interactive       Never prompt
--target USER@HOST      Configure one Linux target over existing admin SSH
--with-journal          Grant target journal-read membership
--profile NAME          Select the tunnel-client profile name
--reconfigure-tunnel    Back up and recreate a mismatched profile
--skip-policy           Skip the policy walkthrough
--skip-target           Skip target preparation/tagging walkthrough
--no-open               Do not open admin-console pages
```

## Troubleshooting

### `.env` is missing

Run:

```bash
./bootstrap
```

### SSH asks for a password

Ensure the target has Tailscale SSH enabled:

```bash
sudo tailscale set --ssh
```

Then use:

```bash
ssh chatgpt+password@100.x.y.z
```

Enter any non-empty value. Do not set a Linux password for `chatgpt` merely for this test.

### Port 8080 is already in use

`8080` belongs to the MCP gateway. The generated tunnel profile uses `8081` for tunnel health/admin UI. Rerun with `--reconfigure-tunnel` if an old profile still conflicts.

### No hosts appear

Check that:

- the target is online;
- it carries `tag:chatgpt-managed` or matches `TS_ALLOWED_HOSTS`;
- `tailscale set --ssh` has been run;
- the gateway appears as its separate `tag:chatgpt-ssh` node;
- both Tailscale network and SSH policy permit the path;
- the target is not shared-to unless `TS_ALLOW_SHARED_NODES=true` was deliberately enabled.

### Service logs are denied

Grant journal-read access and start a new SSH session:

```bash
./bootstrap --target ADMIN@HOST --with-journal
```

### Operator tool does not appear

Stop any existing gateway, then start a fresh Operator lease:

```bash
make operator-up
```

Refresh/rescan the custom MCP app's tools in ChatGPT. Operator mode is intentionally not persisted in the example `.env`.
