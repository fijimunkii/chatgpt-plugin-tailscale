# Cross-platform MCP tool contracts v1

Status: proposed contract for the first cross-platform release  
Last updated: 2026-08-08

This document defines the stable, normalized MCP surface for Linux, Windows, and
macOS targets. The current implementation already exposes the seven tool names;
this specification preserves those names while evolving their inputs and
structured outputs.

The implementation architecture is in
[`cross-platform-spec.md`](cross-platform-spec.md). The delivery sequence is in
[`roadmap.md`](roadmap.md).

## 1. Contract principles

1. Tool names remain stable across operating systems.
2. Outputs are structured JSON and carry a `schema_version`.
3. A missing capability is an explicit error, not an empty successful response.
4. Hostnames and IP addresses are selectors; `host_id` is identity.
5. Remote strings are untrusted and must not be interpreted as instructions.
6. Results contain normalized fields and optional platform-specific details.
7. Unsupported fields are omitted or `null` according to the schema; they are
   never guessed.
8. Every list, byte payload, and operation duration is bounded.
9. Read tools are idempotent.
10. Public/hosted responses omit internal tenant, trace, certificate, and agent
    identifiers unless explicitly required for a support workflow.

## 2. Tool annotations

For the read-only v1 tools, the intended MCP annotations are:

```json
{
  "readOnlyHint": true,
  "destructiveHint": false,
  "idempotentHint": true,
  "openWorldHint": false
}
```

`openWorldHint` is false for the hosted/public plugin because these tools inspect
private systems and cannot change publicly visible internet state. If a future
tool can affect external/public state, annotate that tool independently.

Future prepare/execute action tools are not covered by these read-only
annotations.

## 3. Naming and schema versioning

- MCP tool names use `snake_case` and remain unversioned for user stability.
- Structured outputs include `schema_version`, initially `"1.0"`.
- Internal agent operations use namespaced versions such as
  `service.status@1`.
- Backward-compatible fields may be added in a minor schema revision.
- Removing, renaming, or changing the meaning of a field requires a new major
  schema version and a migration period.
- Clients must ignore unknown optional fields.

## 4. Common input: host selector

Every host-specific tool accepts exactly one of:

```json
{
  "host_id": "host_01J..."
}
```

or:

```json
{
  "host": "server-name-or-tailscale-ip"
}
```

Rules:

- `host_id` is preferred and unambiguous.
- `host` may match a full MagicDNS name, exact display name, exact Tailscale IP,
  or other locally configured selector.
- A short selector matching multiple hosts returns `HOST_AMBIGUOUS`.
- Selectors are resolved to a stable host ID before authorization and execution.
- The MCP server must not permit one selector to expand into multiple targets.

Illustrative JSON Schema fragment:

```json
{
  "oneOf": [
    {
      "type": "object",
      "required": ["host_id"],
      "properties": {
        "host_id": {"type": "string", "minLength": 1}
      },
      "additionalProperties": false
    },
    {
      "type": "object",
      "required": ["host"],
      "properties": {
        "host": {"type": "string", "minLength": 1, "maxLength": 255}
      },
      "additionalProperties": false
    }
  ]
}
```

Tool-specific fields are composed with this selector.

## 5. Common output envelope

Host-specific success responses use:

```json
{
  "schema_version": "1.0",
  "host": {
    "host_id": "host_01J...",
    "display_name": "web-01",
    "platform": "linux",
    "os_name": "Ubuntu",
    "os_version": "24.04",
    "architecture": "arm64"
  },
  "collected_at": "2026-08-08T18:00:00Z",
  "transport": "tailscale_ssh",
  "remote_content_untrusted": true,
  "truncated": false,
  "warnings": [],
  "data": {}
}
```

### 5.1 `host`

```json
{
  "host_id": "string",
  "display_name": "string",
  "platform": "linux | windows | darwin | unknown",
  "os_name": "string | null",
  "os_version": "string | null",
  "architecture": "amd64 | arm64 | 386 | arm | unknown"
}
```

### 5.2 `transport`

Allowed values:

```text
agent
tailscale_ssh
openssh
winrm
unknown
```

`openssh` and `winrm` are reserved for optional/future adapters.

### 5.3 Trust and truncation

- `remote_content_untrusted` is always `true` for host-derived data.
- `truncated` means at least one result field was bounded.
- Each bounded collection may also include its own truncation information.
- `warnings` are machine-readable objects, not free-form instructions.

Warning shape:

```json
{
  "code": "PROCESS_DETAILS_PARTIAL",
  "message": "Some process owners or command lines were not accessible.",
  "affected_items": 12
}
```

## 6. Common error contract

Tool failures return MCP `isError: true` and a structured payload:

```json
{
  "schema_version": "1.0",
  "error": {
    "code": "HOST_OFFLINE",
    "message": "The selected host is currently offline.",
    "retryable": true,
    "retry_after_seconds": null,
    "details": {}
  }
}
```

Canonical error codes:

| Code | Meaning | Retryable |
| --- | --- | --- |
| `INVALID_ARGUMENT` | Input failed schema or semantic validation. | No |
| `HOST_NOT_FOUND` | No permitted host matches the selector. | No |
| `HOST_AMBIGUOUS` | Selector matches more than one permitted host. | No |
| `HOST_OFFLINE` | Host is known but not currently reachable. | Yes |
| `CONNECTOR_OFFLINE` | Required edge connector is offline. | Yes |
| `AGENT_OFFLINE` | Agent-backed host has no active session. | Yes |
| `CAPABILITY_UNAVAILABLE` | Host/adapter does not implement this operation. | No |
| `POLICY_DENIED` | User, workspace, connector, or agent policy denied the call. | No |
| `SERVICE_NOT_ALLOWED` | Service is not in the exact allowlist. | No |
| `SERVICE_NOT_FOUND` | Allowed service does not exist on the host. | No |
| `LOG_SOURCE_NOT_ALLOWED` | Requested log source is not allowlisted. | No |
| `AUTHENTICATION_FAILED` | Transport or agent authentication failed. | Usually no |
| `CERTIFICATE_EXPIRED` | Agent certificate expired and could not rotate. | Yes |
| `VERSION_UNSUPPORTED` | Protocol/tool schema versions cannot interoperate. | No |
| `PLATFORM_ACCESS_DENIED` | OS denied access to requested read data. | No |
| `DEADLINE_EXCEEDED` | Operation exceeded its deadline. | Yes |
| `OUTPUT_LIMIT_EXCEEDED` | Result could not be represented within hard limits. | No |
| `INTERNAL_ERROR` | Unexpected internal failure. | Maybe |

`details` must not contain stack traces, credentials, full file paths containing
usernames, raw tokens, or internal tenant IDs in public responses.

## 7. `list_hosts`

Purpose: list only hosts the authenticated caller and local connector policy may
inspect.

### 7.1 Input

```json
{
  "online_only": false,
  "platform": null,
  "capability": null,
  "include_addresses": false,
  "limit": 100,
  "cursor": null
}
```

Fields:

| Field | Type | Default | Rules |
| --- | --- | --- | --- |
| `online_only` | boolean | `false` | Exclude known offline hosts when true. |
| `platform` | enum/null | null | `linux`, `windows`, or `darwin`. |
| `capability` | string/null | null | Exact capability such as `service.logs@1`. |
| `include_addresses` | boolean | `false` | Return Tailscale addresses only when requested and authorized. |
| `limit` | integer | 100 | 1–500. |
| `cursor` | string/null | null | Opaque pagination cursor. |

### 7.2 Output

```json
{
  "schema_version": "1.0",
  "hosts": [
    {
      "host_id": "host_01J...",
      "display_name": "windows-desktop",
      "dns_name": "windows-desktop.tailnet-name.ts.net",
      "platform": "windows",
      "os_name": "Windows 11 Pro",
      "os_version": "10.0.26100",
      "architecture": "amd64",
      "online": true,
      "last_seen": "2026-08-08T18:00:00Z",
      "connection": {
        "preferred_transport": "agent",
        "available_transports": ["agent"],
        "connector_online": true,
        "agent_version": "0.3.0"
      },
      "capabilities": [
        "host.inspect@1",
        "disk.usage@1",
        "process.list@1",
        "service.status@1",
        "service.logs@1"
      ],
      "tailscale_addresses": null
    }
  ],
  "next_cursor": null,
  "truncated": false
}
```

Privacy:

- IP addresses are omitted unless `include_addresses=true` and policy permits.
- Raw Tailscale node IDs, certificate fingerprints, and internal connector IDs
  are not returned.
- Hosts excluded by policy are indistinguishable from nonexistent hosts.

## 8. `inspect_host`

Purpose: return identity, connection, capability, and security-relevant status
for one permitted host without performing a broad system scan.

### 8.1 Input

```json
{
  "host_id": "host_01J...",
  "include_addresses": false
}
```

### 8.2 Output `data`

```json
{
  "identity": {
    "display_name": "windows-desktop",
    "dns_name": "windows-desktop.tailnet-name.ts.net",
    "platform": "windows",
    "os_name": "Windows 11 Pro",
    "os_version": "10.0.26100",
    "kernel_version": "10.0.26100",
    "architecture": "amd64",
    "timezone": "America/New_York"
  },
  "connection": {
    "online": true,
    "last_seen": "2026-08-08T18:00:00Z",
    "preferred_transport": "agent",
    "available_transports": ["agent"],
    "agent_version": "0.3.0",
    "agent_protocol_version": "1.0",
    "certificate_expires_at": "2026-08-09T18:00:00Z"
  },
  "capabilities": [
    {
      "name": "service.logs",
      "version": 1,
      "constraints": {
        "max_entries": 200,
        "providers": ["windows_event_log"]
      }
    }
  ],
  "policy": {
    "read_only": true,
    "allowed_service_count": 2,
    "allowed_log_source_count": 2
  },
  "addresses": null
}
```

Do not return policy contents, certificate fingerprints, enrollment secrets, or
raw local account identifiers.

## 9. `system_summary`

Purpose: provide a small cross-platform health snapshot.

### 9.1 Input

Host selector only.

### 9.2 Output `data`

```json
{
  "boot_time": "2026-08-07T13:22:12Z",
  "uptime_seconds": 102468,
  "cpu": {
    "logical_processors": 12,
    "utilization_percent": 17.4,
    "sample_window_seconds": 1,
    "load_average_1m": null,
    "load_average_5m": null,
    "load_average_15m": null
  },
  "memory": {
    "total_bytes": 34359738368,
    "available_bytes": 18123407360,
    "used_bytes": 16236331008,
    "used_percent": 47.25
  },
  "swap": {
    "total_bytes": 4294967296,
    "used_bytes": 536870912,
    "used_percent": 12.5
  },
  "platform_details": {
    "windows_edition": "Professional"
  }
}
```

Normalization:

- `uptime_seconds` is monotonic-derived where possible.
- CPU utilization requires a declared sample window. If unavailable, it is
  `null`; do not substitute load average.
- Unix load averages are returned only on platforms that expose them.
- `available_bytes` follows the best platform-native estimate and may not equal
  “free” memory.
- `used_bytes` is `total - available` unless a platform provides a more precise
  normalized definition; the adapter documents any difference in
  `platform_details`.
- Swap/pagefile fields may be `null` when not meaningful or unavailable.

## 10. `disk_usage`

Purpose: return bounded filesystem/volume capacity information.

### 10.1 Input

```json
{
  "host_id": "host_01J...",
  "include_pseudo": false,
  "limit": 100
}
```

### 10.2 Output `data`

```json
{
  "volumes": [
    {
      "volume_id": "C:",
      "display_name": "Windows",
      "mount_point": "C:\\",
      "filesystem": "NTFS",
      "device": null,
      "total_bytes": 1000204886016,
      "free_bytes": 502345678848,
      "available_bytes": 502345678848,
      "used_bytes": 497859207168,
      "used_percent": 49.78,
      "read_only": false,
      "remote": false,
      "pseudo": false
    }
  ],
  "returned": 1,
  "available": 1,
  "truncated": false
}
```

Rules:

- byte values use base-2 byte counts, not formatted strings;
- percentages are 0–100 and rounded only for display by the client;
- Windows drive letters and volume GUID paths are normalized into
  `volume_id`/`mount_point`;
- Linux/macOS pseudo filesystems are excluded by default;
- remote/network mounts are identified where possible but not probed beyond the
  operation deadline;
- inaccessible volumes produce warnings rather than aborting the entire result
  unless none can be read.

## 11. `process_list`

Purpose: return a bounded snapshot of processes. It is not a continuous process
monitor.

### 11.1 Input

```json
{
  "host_id": "host_01J...",
  "filter": "nginx",
  "limit": 100,
  "include_command_line": false,
  "sort_by": "memory_bytes",
  "sort_direction": "desc"
}
```

Fields:

| Field | Type | Default | Rules |
| --- | --- | --- | --- |
| `filter` | string/null | null | Literal case-insensitive local filter; never sent into a shell. Max 128 chars. |
| `limit` | integer | 100 | 1–500 and capped by policy/capability. |
| `include_command_line` | boolean | false | Requires explicit policy; likely to contain secrets. |
| `sort_by` | enum | `pid` | `pid`, `name`, `memory_bytes`, `cpu_percent`, `start_time`. |
| `sort_direction` | enum | `asc` | `asc` or `desc`. |

### 11.2 Output `data`

```json
{
  "processes": [
    {
      "pid": 4120,
      "parent_pid": 732,
      "name": "nginx.exe",
      "executable_path": "C:\\Program Files\\nginx\\nginx.exe",
      "owner": "HOST\\svc-nginx",
      "state": "running",
      "start_time": "2026-08-08T12:00:00Z",
      "cpu_percent": 1.2,
      "cpu_sample_window_seconds": 1,
      "memory_bytes": 73400320,
      "thread_count": 8,
      "command_line": null,
      "details_complete": true
    }
  ],
  "returned": 1,
  "matched": 1,
  "truncated": false,
  "details_denied_count": 0
}
```

Rules:

- filter and sorting occur locally in the adapter, not through command
  interpolation;
- command line is off by default and passes secret redaction when enabled;
- owner, path, command line, CPU, start time, or memory may be inaccessible for
  individual processes; set those fields to `null` and
  `details_complete=false`;
- Windows protected processes must not require the agent to run as LocalSystem
  merely to fill optional fields;
- PID reuse means process rows are snapshots, not durable identities;
- no environment variables are returned.

## 12. `service_status`

Purpose: return the state of one exact allowlisted service.

### 12.1 Input

```json
{
  "host_id": "host_01J...",
  "service": "Tailscale"
}
```

Rules:

- `service` must be an exact allowlisted service identifier;
- Windows comparison is case-insensitive but does not accept display-name
  substring matching;
- Linux expects an exact unit such as `tailscaled.service`;
- macOS expects the exact supported service/label identifier;
- aliases may be configured explicitly in policy and resolve before execution.

### 12.2 Normalized state enum

```text
unknown
stopped
start_pending
stop_pending
running
continue_pending
pause_pending
paused
failed
```

### 12.3 Output `data`

```json
{
  "service": {
    "service_id": "Tailscale",
    "display_name": "Tailscale",
    "description": "Private network service",
    "state": "running",
    "enabled": true,
    "process_id": 1880,
    "started_at": null,
    "exit_code": 0,
    "platform_state": "SERVICE_RUNNING",
    "platform_substate": null
  }
}
```

Normalization examples:

| Platform | Native state | Normalized |
| --- | --- | --- |
| systemd | `active` + `running` | `running` |
| systemd | `failed` | `failed` |
| Windows SCM | `SERVICE_START_PENDING` | `start_pending` |
| Windows SCM | `SERVICE_PAUSED` | `paused` |
| launchd | running process | `running` |
| launchd | loaded but not running | `stopped` or platform-specific detail |

`enabled` may be `null` when startup configuration cannot be determined
reliably.

## 13. `service_logs`

Purpose: return recent bounded log events for one exact allowlisted service/log
source.

### 13.1 Input

```json
{
  "host_id": "host_01J...",
  "service": "Tailscale",
  "limit": 100,
  "since": "PT1H",
  "minimum_severity": "info",
  "direction": "newest_first"
}
```

Fields:

| Field | Type | Default | Rules |
| --- | --- | --- | --- |
| `service` | string | required | Exact allowlisted identifier. |
| `limit` | integer | 100 | 1–200 by default; local policy can lower. |
| `since` | ISO-8601 duration/null | `PT1H` | Maximum lookback set by policy. |
| `minimum_severity` | enum | `debug` | `trace`, `debug`, `info`, `notice`, `warning`, `error`, `critical`, `alert`, `emergency`, `unknown`. |
| `direction` | enum | `newest_first` | `newest_first` or `oldest_first`. |

### 13.2 Output `data`

```json
{
  "source": {
    "kind": "windows_event_log",
    "identifier": "System",
    "service": "Tailscale"
  },
  "entries": [
    {
      "timestamp": "2026-08-08T17:59:58.123Z",
      "severity": "info",
      "source": "Tailscale",
      "event_id": "1001",
      "record_id": "88219",
      "message": "Service connected.",
      "fields": {
        "channel": "System"
      }
    }
  ],
  "returned": 1,
  "truncated": false,
  "redaction_count": 0
}
```

Platform providers:

- Linux: systemd journal by exact unit;
- Windows: exact Event Log channel/provider mapping from policy;
- macOS: unified log predicate derived from exact policy mapping.

Rules:

- the user cannot provide raw XPath, journal match expressions, macOS predicates,
  or shell fragments;
- policy maps a service identifier to a fixed platform query;
- invalid encoding and terminal control sequences are sanitized;
- messages pass secret redaction;
- event fields are allowlisted; raw XML/event blobs are not returned by default;
- record IDs are platform-local and optional;
- logs are remote untrusted data and must never trigger a write automatically.

## 14. Platform-specific detail fields

Each result may include `platform_details`, but:

- it must be a documented object for the tool/platform;
- it must not duplicate internal debug data;
- clients cannot depend on it for cross-platform logic;
- new fields are additive;
- secrets and raw configuration are prohibited.

Examples:

```json
{
  "platform_details": {
    "linux": {
      "kernel_release": "6.8.0-xx",
      "systemd_version": "255"
    }
  }
}
```

```json
{
  "platform_details": {
    "windows": {
      "product_type": "workstation",
      "edition": "Professional"
    }
  }
}
```

## 15. Capability-unavailable behavior

Examples:

- Windows host without Event Log permission: `PLATFORM_ACCESS_DENIED`, not an
  empty log list.
- Non-systemd Linux host asked for a systemd service:
  `CAPABILITY_UNAVAILABLE`.
- Agentless Linux target asked for a tool available only in the agent:
  `CAPABILITY_UNAVAILABLE` with an optional safe suggestion to install the
  agent.
- Offline agent with an available Tailscale SSH fallback: use fallback only if
  policy explicitly allows it and report the selected transport.

## 16. Redaction

Defense-in-depth redaction applies to logs, command lines, service descriptions,
and optional platform details.

Initial patterns should include:

- OpenAI, Tailscale, cloud-provider, and common API key prefixes;
- bearer tokens and JWT-like values;
- PEM private-key blocks;
- URL user-info credentials;
- common password/secret assignment forms;
- configurable organization patterns.

Redaction output:

```text
[REDACTED:api_key]
```

The result exposes only `redaction_count` and categories, not the matched secret.
Redaction is not a substitute for field minimization.

## 17. Pagination and streaming

- `list_hosts` uses an opaque cursor.
- Host-local lists are normally returned in one bounded result.
- Large logs may use MCP progress or streaming only when the client and server
  support it, but the total result remains bounded.
- Cursors must be signed/opaque and short-lived in hosted mode.
- A cursor may not grant access beyond current policy.

## 18. Time semantics

- Timestamps are RFC 3339 UTC with fractional seconds when available.
- Durations are seconds or ISO-8601 durations as documented per field.
- `collected_at` is the adapter collection time, not the MCP receive time.
- Clock skew may be reported as a warning if connector and host differ beyond a
  configured threshold.
- Relative phrases from the user are resolved by the MCP layer into explicit
  time bounds before dispatch.

## 19. Numeric semantics

- Byte values are integers and represent bytes.
- Percentages are numbers from 0 through 100.
- CPU percentages state the sample window.
- Counters that may exceed JavaScript safe integer range should be encoded as
  decimal strings in a future schema or kept below that range; v1 fields must be
  reviewed individually.
- Human formatting belongs in the client presentation, not the adapter.

## 20. Public plugin response minimization

Hosted/public responses should not include by default:

- internal request or trace IDs;
- workspace, tenant, connector, or database IDs;
- Tailscale node IDs;
- certificate serials/fingerprints;
- exact IP addresses unless requested;
- user email or local account identity unless needed for the tool;
- debug payloads;
- environment variables;
- authentication metadata.

These values can remain in protected audit metadata where justified.

## 21. Conformance fixtures

Every adapter must pass shared fixtures for:

- online/offline host states;
- each normalized service state;
- zero, partial, and full process details;
- volume truncation and inaccessible volumes;
- log severity mapping;
- invalid Unicode and terminal escapes;
- secret redaction;
- exact allowlist enforcement;
- deadlines and cancellation;
- output limits;
- capability unavailable;
- stable schema serialization.

Golden outputs should be platform-neutral. Platform adapters additionally test
native-to-normalized mappings.

## 22. Compatibility with the current implementation

Migration requirements:

- Keep all seven current MCP tool names.
- Continue accepting the current `host` selector.
- Add `host_id` without immediately removing `host`.
- Preserve current default bounds or make them stricter.
- Keep agentless Linux Tailscale SSH behavior.
- Introduce the common envelope behind a documented feature/version boundary if
  current clients depend on the old shape.
- Rescan the ChatGPT development app after schemas change.

## 23. Future write contracts

Write operations are explicitly excluded from v1, but the contract shape is
reserved.

### 23.1 Prepare

```json
{
  "host_id": "host_01J...",
  "action": "service.restart",
  "arguments": {
    "service": "Tailscale"
  }
}
```

Returns:

```json
{
  "action_id": "action_01J...",
  "action_hash": "sha256:...",
  "expires_at": "2026-08-08T18:05:00Z",
  "summary": "Restart service Tailscale on windows-desktop",
  "effects": ["The service may be unavailable during restart."],
  "rollback": "The service manager will attempt to return it to running state.",
  "approval_required": true
}
```

### 23.2 Execute

Execution accepts only the immutable action ID/hash and approval reference. It
must not accept modified action arguments.

Write tools must have accurate non-read-only/destructive annotations and a
separate authorization scope.

## 24. Reference mapping sources

- Windows Service Control Manager status:
  <https://learn.microsoft.com/windows/win32/api/winsvc/nf-winsvc-queryservicestatusex>
- Windows Event Log query model:
  <https://learn.microsoft.com/windows/win32/wes/querying-for-events>
- Tailscale SSH behavior and platform limitations:
  <https://tailscale.com/docs/features/tailscale-ssh>
- OpenAI MCP tool review and annotations:
  <https://developers.openai.com/plugins/deploy/app-review>
