package mcpapp

import (
	"context"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/remote"
	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/sshx"
	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/tailnet"
)

type fakeCatalog struct {
	target tailnet.Target
}

func (f fakeCatalog) List(context.Context) ([]tailnet.Host, error) {
	return []tailnet.Host{f.target.Host}, nil
}

func (f fakeCatalog) Resolve(context.Context, string) (tailnet.Target, error) {
	return f.target, nil
}

type recordingRunner struct {
	command remote.Command
	result  sshx.Result
}

func (r *recordingRunner) Run(_ context.Context, _ tailnet.Target, _ string, command remote.Command) (sshx.Result, error) {
	r.command = command
	return r.result, nil
}

func testTarget() tailnet.Target {
	return tailnet.Target{
		Host: tailnet.Host{
			ID:                 "n123",
			HostName:           "app-1",
			DNSName:            "app-1.example.ts.net",
			OS:                 "linux",
			Online:             true,
			ReadyForInspection: true,
		},
		DialIP:      netip.MustParseAddr("100.64.0.10"),
		SSHHostKeys: []string{"ssh-ed25519 AAAATEST"},
	}
}

func TestProcessFilterNeverEntersRemoteCommand(t *testing.T) {
	policy, err := remote.NewServicePolicy(nil)
	if err != nil {
		t.Fatalf("NewServicePolicy: %v", err)
	}
	runner := &recordingRunner{result: sshx.Result{
		Success:  true,
		ExitCode: 0,
		Stdout: strings.Join([]string{
			"1 0 root S 1.0 0.1 10 init /sbin/init",
			"2 1 app S 0.5 0.2 9 worker needle-value",
		}, "\n"),
	}}
	app, err := New(fakeCatalog{target: testTarget()}, runner, policy, 200, 100, OperatorConfig{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, result, err := app.processList(context.Background(), nil, processListInput{
		Host:  "app-1",
		Query: "needle-value",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("processList: %v", err)
	}
	if strings.Contains(runner.command.Shell(), "needle-value") {
		t.Fatalf("caller query entered remote command: %q", runner.command.Shell())
	}
	if !strings.Contains(result.Stdout, "needle-value") || strings.Contains(result.Stdout, "/sbin/init") {
		t.Fatalf("unexpected locally filtered output: %q", result.Stdout)
	}
	if result.Operation != "process_list" || !result.UntrustedData {
		t.Fatalf("unexpected result metadata: %+v", result)
	}
}

func TestReadOnlyToolAnnotationsDescribeExternalRead(t *testing.T) {
	tool := readOnlyTool("test", "Test", "test")
	if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint || !tool.Annotations.IdempotentHint {
		t.Fatalf("missing read-only annotations: %+v", tool.Annotations)
	}
	if tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint {
		t.Fatalf("destructive annotation is wrong: %+v", tool.Annotations)
	}
	if tool.Annotations.OpenWorldHint == nil || !*tool.Annotations.OpenWorldHint {
		t.Fatalf("external interaction must be marked open-world: %+v", tool.Annotations)
	}
}

func TestOperatorToolAnnotationsAreDestructive(t *testing.T) {
	tool := operatorTool()
	if tool.Annotations == nil || tool.Annotations.ReadOnlyHint || tool.Annotations.IdempotentHint {
		t.Fatalf("operator annotations should be writable/non-idempotent: %+v", tool.Annotations)
	}
	if tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
		t.Fatalf("operator tool must be marked destructive: %+v", tool.Annotations)
	}
}

func TestOperatorExecRequiresActiveLocalLease(t *testing.T) {
	policy, err := remote.NewServicePolicy(nil)
	if err != nil {
		t.Fatalf("NewServicePolicy: %v", err)
	}
	runner := &recordingRunner{result: sshx.Result{Success: true, ExitCode: 0, Stdout: "ok\n"}}
	app, err := New(fakeCatalog{target: testTarget()}, runner, policy, 200, 100, OperatorConfig{
		Enabled:   true,
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, result, err := app.operatorExec(context.Background(), nil, operatorExecInput{
		Host:    "app-1",
		Command: "printf ok",
		CWD:     "/tmp/work dir",
	})
	if err != nil {
		t.Fatalf("operatorExec: %v", err)
	}
	if runner.command.Operation() != "operator_exec" {
		t.Fatalf("operation = %q", runner.command.Operation())
	}
	if !strings.Contains(runner.command.Shell(), "printf ok") || !strings.Contains(runner.command.Shell(), "/tmp/work dir") {
		t.Fatalf("unexpected operator shell: %q", runner.command.Shell())
	}
	if !result.OperatorMode || result.PrivilegeBoundary == "" || result.LeaseExpiresAt == "" {
		t.Fatalf("unexpected operator result metadata: %+v", result)
	}
}

func TestExpiredOperatorLeaseIsRejected(t *testing.T) {
	cfg := OperatorConfig{Enabled: true, ExpiresAt: time.Now().Add(-time.Minute)}
	if cfg.validate(time.Now()) == nil {
		t.Fatal("expected expired operator config to be rejected")
	}
}
