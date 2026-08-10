package remote

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const systemSummaryScript = `set -eu
printf 'hostname\t'
hostname
printf 'kernel\t'
uname -srmo 2>/dev/null || uname -a
printf 'uptime_seconds\t'
awk '{printf "%d\n", $1}' /proc/uptime
printf 'load_average\t'
cat /proc/loadavg
awk '
  /^MemTotal:/ {print "memory_total_kib\t" $2}
  /^MemAvailable:/ {print "memory_available_kib\t" $2}
  /^SwapTotal:/ {print "swap_total_kib\t" $2}
  /^SwapFree:/ {print "swap_free_kib\t" $2}
' /proc/meminfo
printf 'boot_id\t'
cat /proc/sys/kernel/random/boot_id
`

const (
	diskUsageShell           = "LC_ALL=C df -P -B1 -x tmpfs -x devtmpfs"
	absoluteMaxLogLines      = 1000
	absoluteMaxProcessLines  = 2000
	maxOperatorCommandBytes  = 64 << 10
	maxOperatorWorkingDirLen = 4096
)

// Command is an opaque remote command. Most constructors in this package build
// fixed read-only operations. OperatorCommand is the deliberate exception: it
// accepts caller-supplied shell text only after the MCP layer has verified that
// locally enabled Operator mode is still inside its lease.
type Command struct {
	operation string
	shell     string
}

func newCommand(operation, shell string) Command {
	return Command{operation: operation, shell: shell}
}

// Operation returns the stable audit name for the command.
func (c Command) Operation() string { return c.operation }

// Shell returns the remote shell command. Only the SSH runner should use this
// value.
func (c Command) Shell() string { return c.shell }

// Valid reports whether c was constructed as a usable command.
func (c Command) Valid() bool { return c.operation != "" && c.shell != "" }

// ServicePolicy is an exact allowlist for systemd unit inspection.
type ServicePolicy struct {
	allowed map[string]struct{}
	names   []string
}

func NewServicePolicy(names []string) (ServicePolicy, error) {
	policy := ServicePolicy{allowed: make(map[string]struct{})}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !ValidServiceName(name) {
			return ServicePolicy{}, fmt.Errorf("invalid service name %q", name)
		}
		if _, exists := policy.allowed[name]; exists {
			continue
		}
		policy.allowed[name] = struct{}{}
		policy.names = append(policy.names, name)
	}
	sort.Strings(policy.names)
	return policy, nil
}

func (p ServicePolicy) Require(name string) error {
	if !ValidServiceName(name) {
		return fmt.Errorf("service %q has an invalid name", name)
	}
	if len(p.allowed) == 0 {
		return errors.New("service inspection is disabled because SSH_ALLOWED_SERVICES is empty")
	}
	if _, ok := p.allowed[name]; !ok {
		return fmt.Errorf("service %q is not allowlisted; configured services: %s", name, strings.Join(p.names, ", "))
	}
	return nil
}

func (p ServicePolicy) Names() []string {
	return append([]string(nil), p.names...)
}

func SystemSummaryCommand() Command {
	return newCommand("system_summary", "LC_ALL=C sh -c "+ShellQuote(systemSummaryScript))
}

func DiskUsageCommand() Command {
	return newCommand("disk_usage", diskUsageShell)
}

func ServiceStatusCommand(policy ServicePolicy, service string) (Command, error) {
	if err := policy.Require(service); err != nil {
		return Command{}, err
	}
	properties := strings.Join([]string{
		"Id",
		"Description",
		"LoadState",
		"ActiveState",
		"SubState",
		"UnitFileState",
		"MainPID",
		"ExecMainStatus",
		"ActiveEnterTimestamp",
		"InactiveEnterTimestamp",
	}, ",")
	return newCommand(
		"service_status",
		"LC_ALL=C systemctl show --no-pager --property="+properties+" -- "+ShellQuote(service),
	), nil
}

func ServiceLogsCommand(policy ServicePolicy, service string, lines, maxLines int) (Command, error) {
	if err := policy.Require(service); err != nil {
		return Command{}, err
	}
	if maxLines < 1 || maxLines > absoluteMaxLogLines {
		return Command{}, fmt.Errorf("maximum log line count must be between 1 and %d", absoluteMaxLogLines)
	}
	if lines < 0 {
		return Command{}, fmt.Errorf("lines must be between 1 and %d", maxLines)
	}
	if lines == 0 {
		lines = min(100, maxLines)
	}
	if lines > maxLines {
		return Command{}, fmt.Errorf("lines must be between 1 and %d", maxLines)
	}
	return newCommand(
		"service_logs",
		"LC_ALL=C journalctl --no-pager --output=short-iso --lines="+strconv.Itoa(lines)+" --unit="+ShellQuote(service),
	), nil
}

func ProcessListCommand(remoteLimit int) Command {
	if remoteLimit < 1 {
		remoteLimit = 1
	}
	if remoteLimit > absoluteMaxProcessLines {
		remoteLimit = absoluteMaxProcessLines
	}
	// The remote cap limits bandwidth before the gateway applies the caller's
	// optional literal filter and final result limit locally.
	return newCommand(
		"process_list",
		"LC_ALL=C ps -eo pid=,ppid=,user=,stat=,pcpu=,pmem=,etimes=,comm=,args= --sort=-pcpu | head -n "+strconv.Itoa(remoteLimit),
	)
}

// OperatorCommand deliberately creates an arbitrary POSIX shell operation for
// locally enabled Operator mode. The command is interpreted by /bin/sh on the
// selected host as the configured SSH user. It does not add sudo, a PTY, or an
// interactive stdin channel. cwd is shell-quoted and cannot inject shell text.
func OperatorCommand(command, cwd string) (Command, error) {
	if strings.TrimSpace(command) == "" {
		return Command{}, errors.New("operator command must not be empty")
	}
	if len(command) > maxOperatorCommandBytes {
		return Command{}, fmt.Errorf("operator command exceeds %d bytes", maxOperatorCommandBytes)
	}
	if strings.ContainsRune(command, '\x00') {
		return Command{}, errors.New("operator command must not contain NUL bytes")
	}
	cwd = strings.TrimSpace(cwd)
	if len(cwd) > maxOperatorWorkingDirLen {
		return Command{}, fmt.Errorf("operator working directory exceeds %d bytes", maxOperatorWorkingDirLen)
	}
	if strings.ContainsRune(cwd, '\x00') {
		return Command{}, errors.New("operator working directory must not contain NUL bytes")
	}

	shell := "/bin/sh -lc " + ShellQuote(command)
	if cwd != "" {
		shell = "cd " + ShellQuote(cwd) + " && exec /bin/sh -lc " + ShellQuote(command)
	}
	return newCommand("operator_exec", shell), nil
}

// FilterProcessOutput applies a literal, case-insensitive filter locally. The
// query is never included in a remote command.
func FilterProcessOutput(output, query string, limit int) string {
	if limit <= 0 {
		return ""
	}
	query = strings.ToLower(strings.TrimSpace(query))
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	selected := make([]string, 0, min(limit, len(lines)))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(line), query) {
			continue
		}
		selected = append(selected, line)
		if len(selected) == limit {
			break
		}
	}
	return strings.Join(selected, "\n")
}

// ValidServiceName accepts only a conservative subset of systemd unit names.
func ValidServiceName(v string) bool {
	if len(v) == 0 || len(v) > 128 || strings.HasPrefix(v, "-") {
		return false
	}
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '.', '_', '@', '-', ':':
			continue
		default:
			return false
		}
	}
	return true
}

// ShellQuote safely represents one literal POSIX shell argument.
func ShellQuote(v string) string {
	if v == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(v, "'", "'\"'\"'") + "'"
}
