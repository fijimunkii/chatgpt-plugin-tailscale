package remote

import (
	"strings"
	"testing"
)

func TestServicePolicyIsExact(t *testing.T) {
	policy, err := NewServicePolicy([]string{"nginx.service", "tailscaled.service"})
	if err != nil {
		t.Fatalf("NewServicePolicy: %v", err)
	}
	if _, err := ServiceStatusCommand(policy, "nginx.service"); err != nil {
		t.Fatalf("allowlisted service rejected: %v", err)
	}
	if _, err := ServiceStatusCommand(policy, "nginx"); err == nil {
		t.Fatal("non-allowlisted alias accepted")
	}
}

func TestServiceNameRejectsShellSyntax(t *testing.T) {
	policy, _ := NewServicePolicy([]string{"nginx.service"})
	for _, value := range []string{"nginx.service;id", "$(id)", "-x", "foo bar.service", "foo/service"} {
		if _, err := ServiceStatusCommand(policy, value); err == nil {
			t.Fatalf("unsafe service name %q accepted", value)
		}
	}
}

func TestServiceLogsCapsLines(t *testing.T) {
	policy, _ := NewServicePolicy([]string{"nginx.service"})
	if _, err := ServiceLogsCommand(policy, "nginx.service", 201, 200); err == nil {
		t.Fatal("expected max-lines error")
	}
	command, err := ServiceLogsCommand(policy, "nginx.service", 50, 200)
	if err != nil {
		t.Fatalf("ServiceLogsCommand: %v", err)
	}
	if !strings.Contains(command.Shell(), "--lines=50") {
		t.Fatalf("unexpected command: %s", command.Shell())
	}
}

func TestProcessFilterIsLiteralAndLimited(t *testing.T) {
	output := "1 0 root S 1.0 0.1 10 init /sbin/init\n2 1 app S 0.5 0.2 9 worker [prod]\n3 1 app S 0.4 0.2 8 worker test\n"
	got := FilterProcessOutput(output, "[prod]", 10)
	if !strings.Contains(got, "[prod]") || strings.Contains(got, "test") {
		t.Fatalf("literal filter failed: %q", got)
	}
	got = FilterProcessOutput(output, "", 2)
	if lines := strings.Count(got, "\n") + 1; lines != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", lines, got)
	}
}

func TestShellQuote(t *testing.T) {
	got := ShellQuote("a'b")
	if got != `'a'"'"'b'` {
		t.Fatalf("ShellQuote = %q", got)
	}
}

func TestSystemSummaryHasNoCallerInput(t *testing.T) {
	command := SystemSummaryCommand()
	for _, expected := range []string{"/proc/uptime", "/proc/meminfo", "boot_id"} {
		if !strings.Contains(command.Shell(), expected) {
			t.Fatalf("command missing %q", expected)
		}
	}
}

func TestCommandIsOpaqueAndNamed(t *testing.T) {
	command := DiskUsageCommand()
	if !command.Valid() || command.Operation() != "disk_usage" || command.Shell() == "" {
		t.Fatalf("unexpected command: %+v", command)
	}
	if (Command{}).Valid() {
		t.Fatal("zero command must be invalid")
	}
}

func TestServicePolicyRejectsLeadingDashEvenIfConfigured(t *testing.T) {
	if _, err := NewServicePolicy([]string{"-x"}); err == nil {
		t.Fatal("leading-dash unit accepted")
	}
}

func TestSystemSummaryUsesAwkEscapesCorrectly(t *testing.T) {
	shell := SystemSummaryCommand().Shell()
	if strings.Contains(shell, `\\n`) || strings.Contains(shell, `\\t`) {
		t.Fatalf("script contains double-escaped awk sequences: %q", shell)
	}
}

func TestServiceLogsRejectsNegativeLines(t *testing.T) {
	policy, _ := NewServicePolicy([]string{"nginx.service"})
	if _, err := ServiceLogsCommand(policy, "nginx.service", -1, 200); err == nil {
		t.Fatal("negative line count accepted")
	}
}

func TestProcessCommandHasAbsoluteRemoteCap(t *testing.T) {
	command := ProcessListCommand(1_000_000)
	if !strings.Contains(command.Shell(), "head -n 2000") {
		t.Fatalf("remote process cap missing: %q", command.Shell())
	}
}
