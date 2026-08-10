package config

import (
	"strings"
	"testing"
	"time"
)

func env(values map[string]string) LookupEnv {
	return func(key string) (string, bool) {
		v, ok := values[key]
		return v, ok
	}
}

func TestDefaultsAreLoopbackAndFailClosed(t *testing.T) {
	cfg, err := LoadFrom(env(nil))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:8080" || !cfg.LoopbackListener() {
		t.Fatalf("unexpected default listener: %q", cfg.ListenAddr)
	}
	if got, want := strings.Join(cfg.AllowedTags, ","), "tag:chatgpt-managed"; got != want {
		t.Fatalf("allowed tags = %q, want %q", got, want)
	}
	if len(cfg.AllowedServices) != 0 {
		t.Fatalf("service allowlist should default empty: %v", cfg.AllowedServices)
	}
}

func TestNonLoopbackRequiresAuthentication(t *testing.T) {
	_, err := LoadFrom(env(map[string]string{"MCP_LISTEN_ADDR": "0.0.0.0:8080"}))
	if err == nil || !strings.Contains(err.Error(), "MCP_BEARER_TOKEN") {
		t.Fatalf("expected bearer-token error, got %v", err)
	}
}

func TestNonLoopbackAllowsStrongToken(t *testing.T) {
	cfg, err := LoadFrom(env(map[string]string{
		"MCP_LISTEN_ADDR":  "0.0.0.0:8080",
		"MCP_BEARER_TOKEN": "0123456789abcdefghijklmnop",
	}))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.BearerToken == "" {
		t.Fatal("expected bearer token")
	}
}

func TestSelectorsCannotBothBeEmpty(t *testing.T) {
	_, err := LoadFrom(env(map[string]string{
		"TS_ALLOWED_TAGS":  "",
		"TS_ALLOWED_HOSTS": "",
	}))
	if err == nil || !strings.Contains(err.Error(), "selector") {
		t.Fatalf("expected selector error, got %v", err)
	}
}

func TestParsesBoundsAndLists(t *testing.T) {
	cfg, err := LoadFrom(env(map[string]string{
		"SSH_COMMAND_TIMEOUT":  "30s",
		"SSH_MAX_LOG_LINES":    "500",
		"TS_ALLOWED_TAGS":      "tag:prod, tag:prod,tag:stage",
		"SSH_ALLOWED_SERVICES": "nginx.service, tailscaled.service",
	}))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.CommandTimeout != 30*time.Second {
		t.Fatalf("timeout = %s", cfg.CommandTimeout)
	}
	if cfg.MaxLogLines != 500 {
		t.Fatalf("max log lines = %d", cfg.MaxLogLines)
	}
	if got := strings.Join(cfg.AllowedTags, ","); got != "tag:prod,tag:stage" {
		t.Fatalf("tags = %q", got)
	}
}

func TestRejectsMalformedTags(t *testing.T) {
	for _, tag := range []string{"prod", "tag:", "tag:prod server", "tag:prod/blue"} {
		_, err := LoadFrom(env(map[string]string{"TS_ALLOWED_TAGS": tag}))
		if err == nil {
			t.Fatalf("malformed tag %q accepted", tag)
		}
	}
}

func TestRejectsUnsafeConfiguredService(t *testing.T) {
	_, err := LoadFrom(env(map[string]string{"SSH_ALLOWED_SERVICES": "-x"}))
	if err == nil || !strings.Contains(err.Error(), "SSH_ALLOWED_SERVICES") {
		t.Fatalf("expected service validation error, got %v", err)
	}
}

func TestRejectsMultipleTailscaleEnrollmentCredentials(t *testing.T) {
	_, err := LoadFrom(env(map[string]string{
		"TS_AUTHKEY":       "tskey-auth-example",
		"TS_CLIENT_SECRET": "tskey-client-example",
	}))
	if err == nil || !strings.Contains(err.Error(), "only one") {
		t.Fatalf("expected mutually exclusive credentials error, got %v", err)
	}
}
