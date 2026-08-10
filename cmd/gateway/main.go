package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/config"
	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/mcpapp"
	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/remote"
	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/sshx"
	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/tailnet"
	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/version"
	"tailscale.com/tsnet"
)

func main() {
	if err := run(); err != nil {
		slog.Error("gateway stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	logger.Info("starting gateway",
		"version", version.Version,
		"commit", version.Commit,
		"build_date", version.BuildDate,
		"listen_addr", cfg.ListenAddr,
		"tailscale_hostname", cfg.TailscaleHostname,
	)
	if cfg.BearerToken == "" {
		if cfg.LoopbackListener() {
			logger.Info("MCP authentication delegated to local transport", "listener", "loopback-only")
		} else {
			logger.Warn("MCP endpoint is unauthenticated on a non-loopback listener")
		}
	}
	if cfg.DisableLocalhostProtection {
		logger.Warn("MCP SDK localhost Host-header protection is disabled")
	}
	if len(cfg.AllowedServices) == 0 {
		logger.Warn("service_status and service_logs are disabled because SSH_ALLOWED_SERVICES is empty")
	}

	operatorConfig := mcpapp.OperatorConfig{}
	if cfg.OperatorMode {
		operatorConfig = mcpapp.OperatorConfig{
			Enabled:   true,
			ExpiresAt: time.Now().Add(cfg.OperatorLeaseDuration),
		}
		logger.Warn("OPERATOR MODE ENABLED: arbitrary shell execution is available on admitted hosts as the configured SSH user",
			"ssh_user", cfg.SSHUser,
			"lease_duration", cfg.OperatorLeaseDuration.String(),
			"lease_expires_at", operatorConfig.ExpiresAt.UTC().Format(time.RFC3339),
			"root_added", false,
		)
	}

	if err := os.MkdirAll(cfg.TailscaleStateDir, 0o700); err != nil {
		return fmt.Errorf("create Tailscale state directory: %w", err)
	}
	if err := os.Chmod(cfg.TailscaleStateDir, 0o700); err != nil {
		return fmt.Errorf("secure Tailscale state directory: %w", err)
	}

	tsServer := &tsnet.Server{
		Dir:           cfg.TailscaleStateDir,
		Hostname:      cfg.TailscaleHostname,
		AuthKey:       cfg.TailscaleAuthKey,
		ClientSecret:  cfg.TailscaleClientSecret,
		AdvertiseTags: cfg.TailscaleAdvertiseTags,
		Ephemeral:     cfg.TailscaleEphemeral,
		ControlURL:    cfg.TailscaleControlURL,
		UserLogf: func(format string, args ...any) {
			logger.Info("tsnet", "message", fmt.Sprintf(format, args...))
		},
	}
	defer func() {
		if err := tsServer.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Warn("close tsnet", "error", err)
		}
	}()

	upCtx, upCancel := context.WithTimeout(context.Background(), cfg.TailscaleUpTimeout)
	status, err := tsServer.Up(upCtx)
	upCancel()
	if err != nil {
		return fmt.Errorf("join tailnet: %w", err)
	}
	if status == nil {
		return errors.New("join tailnet: Tailscale returned no status")
	}
	if status.CurrentTailnet != nil {
		logger.Info("joined tailnet",
			"tailnet", status.CurrentTailnet.Name,
			"tailscale_ips", status.TailscaleIPs,
		)
	} else {
		logger.Info("joined tailnet", "tailscale_ips", status.TailscaleIPs)
	}

	localClient, err := tsServer.LocalClient()
	if err != nil {
		return fmt.Errorf("create Tailscale LocalAPI client: %w", err)
	}
	catalog, err := tailnet.NewCatalog(tailnet.NewTSNetSource(localClient), tailnet.Policy{
		AllowedTags:      cfg.AllowedTags,
		AllowedHosts:     cfg.AllowedHosts,
		AllowSharedNodes: cfg.AllowSharedNodes,
	})
	if err != nil {
		return fmt.Errorf("create tailnet catalog: %w", err)
	}
	servicePolicy, err := remote.NewServicePolicy(cfg.AllowedServices)
	if err != nil {
		return fmt.Errorf("create service policy: %w", err)
	}
	runner, err := sshx.NewRunner(
		tsServer,
		cfg.SSHUser,
		cfg.DialTimeout,
		cfg.CommandTimeout,
		cfg.MaxOutputBytes,
		logger,
	)
	if err != nil {
		return fmt.Errorf("create SSH runner: %w", err)
	}
	app, err := mcpapp.New(catalog, runner, servicePolicy, cfg.MaxLogLines, cfg.MaxProcesses, operatorConfig)
	if err != nil {
		return fmt.Errorf("create MCP app: %w", err)
	}
	mcpServer := app.Server(version.Version)
	handler := mcpapp.HTTPHandler(mcpServer, mcpapp.HTTPConfig{
		BearerToken:                cfg.BearerToken,
		DisableLocalhostProtection: cfg.DisableLocalhostProtection,
		MaxRequestBodyBytes:        cfg.MaxRequestBodyBytes,
		Logger:                     logger,
	})

	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.ListenAddr, err)
	}
	defer listener.Close()

	httpServer := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("MCP gateway ready", "endpoint", "http://"+listener.Addr().String()+"/mcp")
		serveErr <- httpServer.Serve(listener)
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-signalCtx.Done():
		logger.Info("shutdown requested")
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	err = <-serveErr
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}
