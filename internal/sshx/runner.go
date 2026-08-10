package sshx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/remote"
	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/tailnet"
	"golang.org/x/crypto/ssh"
)

// Dialer is implemented by tsnet.Server.
type Dialer interface {
	Dial(context.Context, string, string) (net.Conn, error)
}

// Result is a bounded capture of one fixed remote inspection command.
type Result struct {
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
	ExitCode    int    `json:"exit_code"`
	Success     bool   `json:"success"`
	Truncated   bool   `json:"truncated"`
	OutputBytes int    `json:"output_bytes"`
	DurationMS  int64  `json:"duration_ms"`
}

// Runner executes commands through Tailscale SSH without private SSH keys.
type Runner struct {
	dialer         Dialer
	user           string
	dialTimeout    time.Duration
	commandTimeout time.Duration
	maxOutputBytes int
	logger         *slog.Logger
}

func NewRunner(dialer Dialer, user string, dialTimeout, commandTimeout time.Duration, maxOutputBytes int, logger *slog.Logger) (*Runner, error) {
	if dialer == nil {
		return nil, errors.New("SSH dialer is required")
	}
	if user == "" {
		return nil, errors.New("SSH user is required")
	}
	if dialTimeout <= 0 || commandTimeout <= 0 {
		return nil, errors.New("SSH timeouts must be positive")
	}
	if maxOutputBytes <= 0 {
		return nil, errors.New("SSH output limit must be positive")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Runner{
		dialer:         dialer,
		user:           user,
		dialTimeout:    dialTimeout,
		commandTimeout: commandTimeout,
		maxOutputBytes: maxOutputBytes,
		logger:         logger,
	}, nil
}

// Run executes a prebuilt command against a fully resolved and allowed target.
// A remote non-zero exit status is represented in Result and is not a transport
// error. The command text and command output are never written to logs.
func (r *Runner) Run(ctx context.Context, target tailnet.Target, operationID string, command remote.Command) (Result, error) {
	started := time.Now()
	if operationID == "" {
		return Result{}, errors.New("operation ID is required")
	}
	if !target.ReadyForInspection {
		return Result{}, fmt.Errorf("host %q is unavailable: %s", targetName(target), target.UnavailableReason)
	}
	if !target.DialIP.IsValid() {
		return Result{}, fmt.Errorf("host %q has no valid Tailscale IP", targetName(target))
	}
	if !command.Valid() {
		return Result{}, errors.New("invalid remote command")
	}
	operation := command.Operation()
	shellCommand := command.Shell()

	hostKeyCallback, _, err := HostKeyVerifier(target.SSHHostKeys)
	if err != nil {
		return Result{}, fmt.Errorf("host %q: %w", targetName(target), err)
	}

	runCtx, cancel := context.WithTimeout(ctx, r.commandTimeout)
	defer cancel()
	dialCtx, dialCancel := context.WithTimeout(runCtx, r.dialTimeout)
	address := net.JoinHostPort(target.DialIP.String(), "22")
	conn, err := r.dialer.Dial(dialCtx, "tcp", address)
	dialCancel()
	if err != nil {
		r.logFailure(operationID, operation, target, started, "dial", err)
		return Result{}, fmt.Errorf("dial Tailscale SSH host %q: %w", targetName(target), err)
	}
	defer conn.Close()
	if deadline, ok := runCtx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	clientConn, channels, requests, err := ssh.NewClientConn(conn, address, &ssh.ClientConfig{
		User:            r.user,
		Auth:            nil, // Tailscale SSH authenticates the node and accepts SSH auth method "none".
		HostKeyCallback: hostKeyCallback,
	})
	if err != nil {
		r.logFailure(operationID, operation, target, started, "handshake", err)
		return Result{}, fmt.Errorf("establish Tailscale SSH session to %q: %w", targetName(target), err)
	}
	client := ssh.NewClient(clientConn, channels, requests)
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		r.logFailure(operationID, operation, target, started, "new_session", err)
		return Result{}, fmt.Errorf("open SSH session to %q: %w", targetName(target), err)
	}
	defer session.Close()

	limiter := newOutputLimiter(r.maxOutputBytes)
	stdout := &limitedBuffer{limiter: limiter}
	stderr := &limitedBuffer{limiter: limiter}
	session.Stdout = stdout
	session.Stderr = stderr

	if err := session.Start(shellCommand); err != nil {
		r.logFailure(operationID, operation, target, started, "start", err)
		return Result{}, fmt.Errorf("start remote inspection on %q: %w", targetName(target), err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- session.Wait()
	}()

	var waitErr error
	select {
	case waitErr = <-waitCh:
	case <-runCtx.Done():
		_ = session.Signal(ssh.SIGKILL)
		_ = session.Close()
		_ = client.Close()
		select {
		case <-waitCh:
		case <-time.After(time.Second):
		}
		result := buildResult(stdout, stderr, -1, started)
		r.logFailure(operationID, operation, target, started, "timeout", runCtx.Err())
		return result, fmt.Errorf("remote inspection on %q: %w", targetName(target), runCtx.Err())
	}

	exitCode := 0
	if waitErr != nil {
		var exitErr *ssh.ExitError
		if errors.As(waitErr, &exitErr) {
			exitCode = exitErr.ExitStatus()
		} else {
			result := buildResult(stdout, stderr, -1, started)
			r.logFailure(operationID, operation, target, started, "wait", waitErr)
			return result, fmt.Errorf("wait for remote inspection on %q: %w", targetName(target), waitErr)
		}
	}
	result := buildResult(stdout, stderr, exitCode, started)
	r.logger.Info("ssh inspection completed",
		"operation_id", operationID,
		"operation", operation,
		"host_id", target.ID,
		"host", targetName(target),
		"ssh_user", r.user,
		"exit_code", result.ExitCode,
		"duration_ms", result.DurationMS,
		"truncated", result.Truncated,
		"output_bytes", result.OutputBytes,
	)
	return result, nil
}

func (r *Runner) logFailure(operationID, operation string, target tailnet.Target, started time.Time, stage string, err error) {
	r.logger.Warn("ssh inspection failed",
		"operation_id", operationID,
		"operation", operation,
		"host_id", target.ID,
		"host", targetName(target),
		"ssh_user", r.user,
		"stage", stage,
		"duration_ms", time.Since(started).Milliseconds(),
		"error", err,
	)
}

func targetName(target tailnet.Target) string {
	if target.DNSName != "" {
		return target.DNSName
	}
	if target.HostName != "" {
		return target.HostName
	}
	return target.ID
}

func buildResult(stdout, stderr *limitedBuffer, exitCode int, started time.Time) Result {
	stdoutText := stdout.String()
	stderrText := stderr.String()
	return Result{
		Stdout:      stdoutText,
		Stderr:      stderrText,
		ExitCode:    exitCode,
		Success:     exitCode == 0,
		Truncated:   stdout.limiter.Truncated(),
		OutputBytes: len(stdoutText) + len(stderrText),
		DurationMS:  time.Since(started).Milliseconds(),
	}
}

type outputLimiter struct {
	mu        sync.Mutex
	remaining int
	truncated bool
}

func newOutputLimiter(max int) *outputLimiter {
	return &outputLimiter{remaining: max}
}

func (l *outputLimiter) take(requested int) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if requested <= l.remaining {
		l.remaining -= requested
		return requested
	}
	accepted := l.remaining
	l.remaining = 0
	l.truncated = true
	return accepted
}

func (l *outputLimiter) Truncated() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.truncated
}

type limitedBuffer struct {
	mu      sync.Mutex
	limiter *outputLimiter
	buf     bytes.Buffer
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	accepted := b.limiter.take(len(p))
	if accepted > 0 {
		b.mu.Lock()
		_, _ = b.buf.Write(p[:accepted])
		b.mu.Unlock()
	}
	// Report the original length so SSH's copy loops continue draining the
	// channel even after the capture limit is reached.
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
