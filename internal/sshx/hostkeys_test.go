package sshx

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"testing"

	"golang.org/x/crypto/ssh"
)

func publicKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	key, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("NewPublicKey: %v", err)
	}
	return key
}

func TestHostKeyVerifierAcceptsOnlyControlPlaneKey(t *testing.T) {
	allowed := publicKey(t)
	other := publicKey(t)
	callback, fingerprints, err := HostKeyVerifier([]string{string(ssh.MarshalAuthorizedKey(allowed))})
	if err != nil {
		t.Fatalf("HostKeyVerifier: %v", err)
	}
	if len(fingerprints) != 1 || fingerprints[0] != ssh.FingerprintSHA256(allowed) {
		t.Fatalf("unexpected fingerprints: %v", fingerprints)
	}
	remote := &net.TCPAddr{}
	if err := callback("host:22", remote, allowed); err != nil {
		t.Fatalf("allowed key rejected: %v", err)
	}
	if err := callback("host:22", remote, other); err == nil {
		t.Fatal("unexpected key accepted")
	}
}

func TestHostKeyVerifierRejectsEmptySet(t *testing.T) {
	if _, _, err := HostKeyVerifier(nil); err == nil {
		t.Fatal("empty host-key set accepted")
	}
}

func TestOutputLimiterBoundsCombinedStreams(t *testing.T) {
	limiter := newOutputLimiter(5)
	stdout := &limitedBuffer{limiter: limiter}
	stderr := &limitedBuffer{limiter: limiter}
	_, _ = stdout.Write([]byte("abcd"))
	_, _ = stderr.Write([]byte("efgh"))
	if got := len(stdout.String()) + len(stderr.String()); got != 5 {
		t.Fatalf("captured %d bytes, want 5", got)
	}
	if !limiter.Truncated() {
		t.Fatal("expected truncation")
	}
}
