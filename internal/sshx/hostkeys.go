package sshx

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"

	"golang.org/x/crypto/ssh"
)

// HostKeyVerifier builds a strict callback from public host keys distributed by
// the Tailscale control plane. It intentionally has no trust-on-first-use mode.
func HostKeyVerifier(encodedKeys []string) (ssh.HostKeyCallback, []string, error) {
	allowed := make([]ssh.PublicKey, 0, len(encodedKeys))
	fingerprints := make([]string, 0, len(encodedKeys))
	seen := make(map[string]struct{})
	for _, encoded := range encodedKeys {
		encoded = strings.TrimSpace(encoded)
		if encoded == "" {
			continue
		}
		key, _, _, rest, err := ssh.ParseAuthorizedKey([]byte(encoded))
		if err != nil {
			return nil, nil, fmt.Errorf("parse Tailscale SSH host key: %w", err)
		}
		if len(bytes.TrimSpace(rest)) != 0 {
			return nil, nil, errors.New("parse Tailscale SSH host key: unexpected trailing data")
		}
		fingerprint := ssh.FingerprintSHA256(key)
		if _, ok := seen[fingerprint]; ok {
			continue
		}
		seen[fingerprint] = struct{}{}
		allowed = append(allowed, key)
		fingerprints = append(fingerprints, fingerprint)
	}
	if len(allowed) == 0 {
		return nil, nil, errors.New("Tailscale did not provide any SSH host keys for the peer")
	}

	callback := func(hostname string, remote net.Addr, presented ssh.PublicKey) error {
		for _, expected := range allowed {
			if bytes.Equal(expected.Marshal(), presented.Marshal()) {
				return nil
			}
		}
		remoteAddress := "unknown"
		if remote != nil {
			remoteAddress = remote.String()
		}
		return fmt.Errorf(
			"SSH host key mismatch for %s (%s): received %s",
			hostname,
			remoteAddress,
			ssh.FingerprintSHA256(presented),
		)
	}
	return callback, fingerprints, nil
}
