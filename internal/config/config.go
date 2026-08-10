package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/remote"
)

const (
	defaultListenAddr            = "127.0.0.1:8080"
	defaultTailscaleHostname     = "chatgpt-ssh-gateway"
	defaultTailscaleStateDir     = "./data/tsnet"
	defaultAdvertiseTag          = "tag:chatgpt-ssh"
	defaultAllowedTag            = "tag:chatgpt-managed"
	defaultSSHUser               = "chatgpt"
	defaultDialTimeout           = 10 * time.Second
	defaultCommandTimeout        = 15 * time.Second
	defaultOperatorLeaseDuration = time.Hour
	defaultTailscaleUpTimeout    = 90 * time.Second
	defaultShutdownTimeout       = 10 * time.Second
	defaultMaxOutputBytes        = 256 << 10
	defaultMaxRequestBodyBytes   = 1 << 20
	defaultMaxLogLines           = 200
	defaultMaxProcesses          = 100
)

// LookupEnv matches os.LookupEnv and makes configuration parsing testable.
type LookupEnv func(string) (string, bool)

// Config is the complete runtime configuration for the gateway.
type Config struct {
	ListenAddr                 string
	BearerToken                string
	AllowUnauthenticated       bool
	DisableLocalhostProtection bool
	MaxRequestBodyBytes        int64

	TailscaleHostname      string
	TailscaleStateDir      string
	TailscaleAuthKey       string
	TailscaleClientSecret  string
	TailscaleAdvertiseTags []string
	TailscaleControlURL    string
	TailscaleEphemeral     bool
	TailscaleUpTimeout     time.Duration

	AllowedTags      []string
	AllowedHosts     []string
	AllowSharedNodes bool

	SSHUser         string
	AllowedServices []string
	DialTimeout     time.Duration
	CommandTimeout  time.Duration
	MaxOutputBytes  int
	MaxLogLines     int
	MaxProcesses    int

	OperatorMode          bool
	OperatorLeaseDuration time.Duration

	ShutdownTimeout time.Duration
	LogLevel        slog.Level
}

// Load reads configuration from the process environment.
func Load() (Config, error) {
	return LoadFrom(os.LookupEnv)
}

// LoadFrom reads configuration through lookup.
func LoadFrom(lookup LookupEnv) (Config, error) {
	cfg := Config{
		ListenAddr:             value(lookup, "MCP_LISTEN_ADDR", defaultListenAddr),
		BearerToken:            value(lookup, "MCP_BEARER_TOKEN", ""),
		TailscaleHostname:      value(lookup, "TS_HOSTNAME", defaultTailscaleHostname),
		TailscaleStateDir:      value(lookup, "TS_STATE_DIR", defaultTailscaleStateDir),
		TailscaleAuthKey:       value(lookup, "TS_AUTHKEY", ""),
		TailscaleClientSecret:  value(lookup, "TS_CLIENT_SECRET", ""),
		TailscaleControlURL:    value(lookup, "TS_CONTROL_URL", ""),
		SSHUser:                value(lookup, "SSH_USER", defaultSSHUser),
		TailscaleAdvertiseTags: csv(value(lookup, "TS_ADVERTISE_TAGS", defaultAdvertiseTag)),
		AllowedTags:            csv(value(lookup, "TS_ALLOWED_TAGS", defaultAllowedTag)),
		AllowedHosts:           csv(value(lookup, "TS_ALLOWED_HOSTS", "")),
		AllowedServices:        csv(value(lookup, "SSH_ALLOWED_SERVICES", "")),
		DialTimeout:            defaultDialTimeout,
		CommandTimeout:         defaultCommandTimeout,
		OperatorLeaseDuration:  defaultOperatorLeaseDuration,
		TailscaleUpTimeout:     defaultTailscaleUpTimeout,
		ShutdownTimeout:        defaultShutdownTimeout,
		MaxOutputBytes:         defaultMaxOutputBytes,
		MaxRequestBodyBytes:    defaultMaxRequestBodyBytes,
		MaxLogLines:            defaultMaxLogLines,
		MaxProcesses:           defaultMaxProcesses,
		LogLevel:               slog.LevelInfo,
	}

	var err error
	if cfg.AllowUnauthenticated, err = boolValue(lookup, "MCP_ALLOW_UNAUTHENTICATED", false); err != nil {
		return Config{}, err
	}
	if cfg.DisableLocalhostProtection, err = boolValue(lookup, "MCP_DISABLE_LOCALHOST_PROTECTION", false); err != nil {
		return Config{}, err
	}
	if cfg.TailscaleEphemeral, err = boolValue(lookup, "TS_EPHEMERAL", false); err != nil {
		return Config{}, err
	}
	if cfg.AllowSharedNodes, err = boolValue(lookup, "TS_ALLOW_SHARED_NODES", false); err != nil {
		return Config{}, err
	}
	if cfg.OperatorMode, err = boolValue(lookup, "OPERATOR_MODE", false); err != nil {
		return Config{}, err
	}
	if cfg.DialTimeout, err = durationValue(lookup, "SSH_DIAL_TIMEOUT", defaultDialTimeout, time.Second, time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.CommandTimeout, err = durationValue(lookup, "SSH_COMMAND_TIMEOUT", defaultCommandTimeout, time.Second, 2*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.OperatorLeaseDuration, err = durationValue(lookup, "OPERATOR_LEASE_DURATION", defaultOperatorLeaseDuration, 5*time.Minute, 24*time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.TailscaleUpTimeout, err = durationValue(lookup, "TS_UP_TIMEOUT", defaultTailscaleUpTimeout, 5*time.Second, 5*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = durationValue(lookup, "SERVER_SHUTDOWN_TIMEOUT", defaultShutdownTimeout, time.Second, time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.MaxOutputBytes, err = intValue(lookup, "SSH_MAX_OUTPUT_BYTES", defaultMaxOutputBytes, 4<<10, 4<<20); err != nil {
		return Config{}, err
	}
	if cfg.MaxRequestBodyBytes, err = int64Value(lookup, "MCP_MAX_REQUEST_BODY_BYTES", defaultMaxRequestBodyBytes, 4<<10, 8<<20); err != nil {
		return Config{}, err
	}
	if cfg.MaxLogLines, err = intValue(lookup, "SSH_MAX_LOG_LINES", defaultMaxLogLines, 1, 1000); err != nil {
		return Config{}, err
	}
	if cfg.MaxProcesses, err = intValue(lookup, "SSH_MAX_PROCESSES", defaultMaxProcesses, 1, 500); err != nil {
		return Config{}, err
	}
	if cfg.LogLevel, err = logLevel(value(lookup, "LOG_LEVEL", "info")); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks security-sensitive invariants.
func (c Config) Validate() error {
	if strings.TrimSpace(c.ListenAddr) == "" {
		return errors.New("MCP_LISTEN_ADDR must not be empty")
	}
	if _, _, err := net.SplitHostPort(c.ListenAddr); err != nil {
		return fmt.Errorf("MCP_LISTEN_ADDR: %w", err)
	}
	if c.BearerToken == "" && !c.AllowUnauthenticated && !isLoopbackListen(c.ListenAddr) {
		return errors.New("MCP_BEARER_TOKEN is required for a non-loopback listener unless MCP_ALLOW_UNAUTHENTICATED=true")
	}
	if len(c.BearerToken) > 0 && len(c.BearerToken) < 24 {
		return errors.New("MCP_BEARER_TOKEN must be at least 24 characters")
	}
	if strings.TrimSpace(c.TailscaleHostname) == "" {
		return errors.New("TS_HOSTNAME must not be empty")
	}
	if strings.TrimSpace(c.TailscaleStateDir) == "" {
		return errors.New("TS_STATE_DIR must not be empty")
	}
	if c.TailscaleAuthKey != "" && c.TailscaleClientSecret != "" {
		return errors.New("set only one of TS_AUTHKEY or TS_CLIENT_SECRET")
	}
	if len(c.TailscaleAdvertiseTags) == 0 {
		return errors.New("TS_ADVERTISE_TAGS must contain at least one tag")
	}
	for _, tag := range c.TailscaleAdvertiseTags {
		if !validTag(tag) {
			return fmt.Errorf("TS_ADVERTISE_TAGS value %q is not a valid tag", tag)
		}
	}
	for _, tag := range c.AllowedTags {
		if !validTag(tag) {
			return fmt.Errorf("TS_ALLOWED_TAGS value %q is not a valid tag", tag)
		}
	}
	if len(c.AllowedTags) == 0 && len(c.AllowedHosts) == 0 {
		return errors.New("at least one TS_ALLOWED_TAGS or TS_ALLOWED_HOSTS selector is required")
	}
	if !validSSHUser(c.SSHUser) {
		return fmt.Errorf("SSH_USER %q is not a safe POSIX username", c.SSHUser)
	}
	for _, service := range c.AllowedServices {
		if !remote.ValidServiceName(service) {
			return fmt.Errorf("SSH_ALLOWED_SERVICES value %q is invalid", service)
		}
	}
	return nil
}

// LoopbackListener reports whether the HTTP listener is loopback-only.
func (c Config) LoopbackListener() bool {
	return isLoopbackListen(c.ListenAddr)
}

func value(lookup LookupEnv, key, fallback string) string {
	if v, ok := lookup(key); ok {
		return strings.TrimSpace(v)
	}
	return fallback
}

func csv(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func boolValue(lookup LookupEnv, key string, fallback bool) (bool, error) {
	v, ok := lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}

func durationValue(lookup LookupEnv, key string, fallback, min, max time.Duration) (time.Duration, error) {
	v, ok := lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	if parsed < min || parsed > max {
		return 0, fmt.Errorf("%s must be between %s and %s", key, min, max)
	}
	return parsed, nil
}

func intValue(lookup LookupEnv, key string, fallback, min, max int) (int, error) {
	v, ok := lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	if parsed < min || parsed > max {
		return 0, fmt.Errorf("%s must be between %d and %d", key, min, max)
	}
	return parsed, nil
}

func int64Value(lookup LookupEnv, key string, fallback, min, max int64) (int64, error) {
	v, ok := lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	if parsed < min || parsed > max {
		return 0, fmt.Errorf("%s must be between %d and %d", key, min, max)
	}
	return parsed, nil
}

func logLevel(v string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("LOG_LEVEL %q must be debug, info, warn, or error", v)
	}
}

func isLoopbackListen(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback()
}

func validSSHUser(v string) bool {
	if len(v) == 0 || len(v) > 32 {
		return false
	}
	for i, r := range v {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || r == '_') {
				return false
			}
			continue
		}
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func validTag(v string) bool {
	if len(v) < len("tag:x") || len(v) > 64 || !strings.HasPrefix(v, "tag:") {
		return false
	}
	for _, r := range strings.TrimPrefix(v, "tag:") {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '-', '_', '.':
			continue
		default:
			return false
		}
	}
	return true
}
