package sshx

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/remote"
	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/tailnet"
)

type rejectingDialer struct {
	called bool
}

func (d *rejectingDialer) Dial(context.Context, string, string) (net.Conn, error) {
	d.called = true
	return nil, errors.New("unexpected dial")
}

func TestRunnerRejectsCommandNotConstructedByRemotePackage(t *testing.T) {
	dialer := new(rejectingDialer)
	runner, err := NewRunner(dialer, "chatgpt", time.Second, time.Second, 4096, nil)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	target := tailnet.Target{
		Host: tailnet.Host{
			ID:                 "n123",
			HostName:           "app-1",
			ReadyForInspection: true,
		},
		DialIP:      netip.MustParseAddr("100.64.0.10"),
		SSHHostKeys: []string{"ssh-ed25519 AAAATEST"},
	}
	_, err = runner.Run(context.Background(), target, "op-1", remote.Command{})
	if err == nil || err.Error() != "invalid remote command" {
		t.Fatalf("expected invalid command error, got %v", err)
	}
	if dialer.called {
		t.Fatal("runner dialed before rejecting an invalid command")
	}
}
