# Bootstrap guide

The supported first-run path is:

```bash
./bootstrap
```

The bootstrap wizard codifies the full setup we otherwise have to coordinate
across Tailscale, the Linux target, the local gateway, OpenAI Secure MCP Tunnel,
and ChatGPT. It is deliberately safe to rerun.

## What the wizard does

The wizard:

1. Checks local prerequisites.
2. Shows the exact Tailscale policy fragment and opens the Access Controls page.
3. Optionally copies and runs the idempotent target helper over an existing
   administrator SSH connection.
4. Creates or preserves `.env`, prompts for the gateway's Tailscale enrollment
   credential when needed, and enforces mode `0600`.
5. Creates or preserves `.env.tunnel` for the OpenAI runtime key and tunnel ID,
   also with mode `0600`.
6. Creates a `tunnel-client` profile with its health listener on
   `127.0.0.1:8081`, avoiding the gateway's port `8080`.
7. Offers to launch the gateway and tunnel together.

The wizard does **not** silently edit your tailnet policy or assign a tag to an
existing machine. Those are high-impact administrative changes and remain
explicit confirmation steps in the Tailscale admin console.

## Rerun behavior

Running `./bootstrap` again is safe:

- Existing `.env` and `.env.tunnel` files are updated by key without deleting
  unrelated settings.
- Repeated keys are collapsed to one entry.
- An existing `tsnet` identity under `TS_STATE_DIR` is reused.
- An existing Linux `chatgpt` account is preserved.
- Existing journal group membership is preserved.
- `tailscale set --ssh` is safe to repeat.
- A matching `tunnel-client` profile is reused.
- A mismatched profile is left untouched unless you explicitly approve a backup
  and replacement or pass `--reconfigure-tunnel`.
- `run-local.sh` will not start duplicate gateway or tunnel processes. It leaves
  processes it did not start untouched.

Use the read-only diagnostics at any time:

```bash
./bootstrap --check
```

## Prerequisites

On the gateway machine:

- Bash
- Go 1.26.5 or later
- `curl`
- OpenAI `tunnel-client`
- `ssh` and `scp` only when using remote target setup

On each target:

- Linux
- Tailscale installed and connected
- `systemd` for service-status and service-log tools
- An existing administrator path that does not depend on the new `chatgpt`
  account

Download the currently supported `tunnel-client` build from OpenAI Platform:

**Organization settings → Tunnels**

On macOS, the release is a command-line binary rather than an application. If
Gatekeeper blocks it, verify the release checksum, attempt to run it once, then
approve that individual binary under **System Settings → Privacy & Security →
Open Anyway**. Do not disable Gatekeeper globally.

## Step 1: Tailscale policy

The wizard displays [`tailscale-policy.hujson`](tailscale-policy.hujson) and
opens the Tailscale Access Controls page.

Merge the entries into the corresponding existing top-level sections. Do not
replace unrelated policy:

```jsonc
{
  "tagOwners": {
    "tag:chatgpt-ssh": ["autogroup:admin"],
    "tag:chatgpt-managed": ["autogroup:admin"],
  },

  "grants": [
    {
      "src": ["tag:chatgpt-ssh"],
      "dst": ["tag:chatgpt-managed"],
      "ip": ["tcp:22"],
    },
  ],

  "ssh": [
    {
      "action": "accept",
      "src": ["tag:chatgpt-ssh"],
      "dst": ["tag:chatgpt-managed"],
      "users": ["chatgpt"],
    },
  ],
}
```

The network grant permits only TCP/22 from the gateway tag to managed targets.
The SSH rule permits only the dedicated local `chatgpt` account. The policy
file also contains commented optional rules for your own Tailscale login. Enable
both optional human rules when your existing policy does not already permit the
manual `ssh chatgpt+password@...` verification; use `check` mode for the human
SSH rule.

Tailscale access rules are additive. An existing wildcard grant can still give
the gateway tag broader reach, so review any existing allow-all rules rather
than assuming the narrow rule subtracts access.

## Step 2: Prepare a Linux target

Pass an existing administrator SSH destination to the wizard:

```bash
./bootstrap --target ubuntu@server.example
```

To let the MCP `service_logs` tool read the system journal where supported:

```bash
./bootstrap --target ubuntu@server.example --with-journal
```

The wizard copies `scripts/bootstrap-target.sh` to a temporary path, runs it
through `sudo`, and removes the temporary copy. The target helper:

- Creates `chatgpt` only when it does not already exist.
- Uses `/home/chatgpt` and `/bin/bash`.
- Does not set a Linux password.
- Optionally adds the user to `systemd-journal`, falling back to `adm` when
  appropriate.
- Runs `tailscale set --ssh`.
- Prints the target's Tailscale IPv4 address and next steps.

You can also run the target helper manually:

```bash
scp scripts/bootstrap-target.sh ubuntu@server:/tmp/bootstrap-target.sh
ssh -t ubuntu@server 'sudo bash /tmp/bootstrap-target.sh --with-journal'
```

Read-only target verification:

```bash
sudo ./scripts/bootstrap-target.sh --check --with-journal
```

After preparing the target, assign `tag:chatgpt-managed` in the Tailscale
**Machines** page. Do not apply this tag to a human laptop; a Tailscale tag
replaces the machine's user identity.

Test with the target's Tailscale IP:

```bash
ssh chatgpt+password@100.x.y.z
```

Enter any non-empty value at the password prompt. The `+password` compatibility
suffix still uses Tailscale identity; it does not authenticate against a Linux
password. Verify:

```bash
whoami
id
```

`whoami` should return `chatgpt`.

## Step 3: Enroll the embedded gateway node

The gateway process creates a separate Tailscale node named
`chatgpt-ssh-gateway`. It is independent of the normal Tailscale client on your
Mac or server.

For the first setup, generate a Tailscale auth key with:

- Tag: `tag:chatgpt-ssh`
- Reusable: off
- Ephemeral: off
- Pre-approved: on when device approval is enabled
- A short key lifetime is sufficient

The wizard securely prompts for the `tskey-auth-...` value and stores it in the
gitignored `.env` file with mode `0600`. Alternatively, supply a Tailscale OAuth
client secret with permission to create auth keys carrying the same tag.

On the first successful start, `tsnet` persists a node identity under:

```text
data/tsnet
```

Subsequent starts reuse that identity and do not need the one-off auth key. Once
you have confirmed a restart succeeds, you may clear the consumed key from
`.env`:

```dotenv
TS_AUTHKEY=
```

Do not delete `data/tsnet` unless you intend to register a replacement gateway
node with a new enrollment credential.

## Step 4: Configure OpenAI Secure MCP Tunnel

The wizard asks for:

- `CONTROL_PLANE_TUNNEL_ID`: the `tunnel_...` identifier from OpenAI Platform.
- `CONTROL_PLANE_API_KEY`: a Runtime API key whose principal has Tunnels
  **Read** and **Use**.

The runtime key is not a model API key used by the MCP gateway. It authenticates
`tunnel-client` to OpenAI's tunnel control plane.

The values are stored locally in:

```text
.env.tunnel
```

That file is gitignored and set to mode `0600`.

The wizard creates a named profile using the HTTP/no-auth starter:

```bash
tunnel-client init \
  --sample sample_mcp_remote_no_auth \
  --profile tailscale-ssh \
  --tunnel-id tunnel_... \
  --mcp-server-url http://127.0.0.1:8080/mcp \
  --health-listen-addr 127.0.0.1:8081
```

The explicit `8081` health listener avoids the common collision where both the
MCP gateway and `tunnel-client` try to bind `127.0.0.1:8080`.

If the profile already exists and matches, the wizard reuses it. To intentionally
replace a mismatched profile:

```bash
./bootstrap --reconfigure-tunnel
```

The old YAML file is backed up with a UTC timestamp before replacement.

## Step 5: Run both processes

After bootstrap:

```bash
./scripts/run-local.sh
```

This command:

1. Loads `.env` and `.env.tunnel`.
2. Reuses an already healthy gateway when present; otherwise builds and starts
   `bin/gateway`.
3. Waits for `http://127.0.0.1:8080/readyz`.
4. Reuses an already healthy tunnel client when present; otherwise starts the
   configured profile.
5. Waits for `http://127.0.0.1:8081/readyz`.
6. Prints the local admin UI URL.
7. On Ctrl-C, stops only the processes it started.

Logs are written to:

```text
data/logs/gateway.log
data/logs/tunnel-client.log
```

Health and status URLs:

```text
Gateway health:       http://127.0.0.1:8080/healthz
Gateway readiness:    http://127.0.0.1:8080/readyz
Tunnel health:        http://127.0.0.1:8081/healthz
Tunnel readiness:     http://127.0.0.1:8081/readyz
Tunnel admin UI:      http://127.0.0.1:8081/ui
```

## Step 6: Add it to ChatGPT

While both local processes are healthy:

1. Enable developer mode in ChatGPT web.
2. Open **Plugins**.
3. Select the plus button and create a developer app.
4. Choose **Tunnel** and select the tunnel ID configured above.
5. Scan tools.
6. Confirm these tools appear:

```text
list_hosts
inspect_host
system_summary
disk_usage
service_status
service_logs
process_list
```

A useful first prompt is:

```text
Using Tailscale SSH Inspector, list the hosts I am allowed to inspect.
```

## Non-interactive bootstrap

For automation, provide values through the environment and explicitly accept or
skip the manual policy step:

```bash
export TS_AUTHKEY='tskey-auth-...'
export CONTROL_PLANE_TUNNEL_ID='tunnel_0123456789abcdef0123456789abcdef'
export CONTROL_PLANE_API_KEY='sk-...'
export BOOTSTRAP_ACCEPT_POLICY=1

./bootstrap --non-interactive --skip-target --no-open
```

Secrets are still written only to the gitignored, mode-`0600` local environment
files. Use an external secrets manager instead when deploying this as a shared
or production service.

## Options

```text
--check                 Read-only diagnostics
--non-interactive       Never prompt
--target USER@HOST      Configure one Linux target over existing admin SSH
--with-journal          Grant target journal-read membership
--profile NAME          Select the tunnel-client profile name
--reconfigure-tunnel    Back up and recreate a mismatched profile
--skip-policy           Skip the policy walkthrough
--skip-target           Skip target preparation and tagging walkthrough
--no-open               Do not open browser pages
```

## Troubleshooting

### `.env: no such file or directory`

Run `./bootstrap`. It creates `.env` from `config/example.env` without replacing
an existing file.

### SSH asks for a password

First make sure the target has Tailscale SSH enabled:

```bash
sudo tailscale set --ssh
```

Then connect over the Tailscale IP using the compatibility suffix:

```bash
ssh chatgpt+password@100.x.y.z
```

Enter any non-empty value. Do not set a Linux password for `chatgpt`.

### `bind: address already in use` on `127.0.0.1:8080`

Port `8080` belongs to the MCP gateway. The generated tunnel profile uses
`127.0.0.1:8081` for `tunnel-client` health and its admin UI. Rerun with
`--reconfigure-tunnel` if an older profile still uses `8080`.

### Tunnel profile file is missing

Run `./bootstrap`. It creates:

```text
~/.config/tunnel-client/tailscale-ssh.yaml
```

or the corresponding path under `XDG_CONFIG_HOME`.

### `list_hosts` returns no targets

Check that:

- The target is online.
- The target carries `tag:chatgpt-managed`.
- `tailscale set --ssh` has been run on the target.
- The gateway appears as a separate machine carrying `tag:chatgpt-ssh`.
- Both the network grant and SSH rule are in the tailnet policy.
- The target is not a shared-to node, unless you intentionally enabled
  `TS_ALLOW_SHARED_NODES=true`.

### Service logs are denied

Rerun the target helper with journal access:

```bash
./bootstrap --target ADMIN@HOST --with-journal
```

Then establish a new SSH session so the supplemental group membership is
applied.
