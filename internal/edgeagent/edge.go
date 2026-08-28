package edgeagent

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/im10furry/edge-dispatch-framework/internal/autotls"
	"github.com/im10furry/edge-dispatch-framework/internal/config"
	"github.com/im10furry/edge-dispatch-framework/internal/edgeagent/localconfig"
	"github.com/im10furry/edge-dispatch-framework/internal/streaming"
	"github.com/im10furry/edge-dispatch-framework/internal/tunnel"
)

// Edge coordinates all edge agent components.
type Edge struct {
	cfg           *config.EdgeAgentConfig
	cache         *Cache
	fetcher       *Fetcher
	server        *Server
	reporter      *Reporter
	tunnelClient  *tunnel.Client // nil if not in NAT mode
	prefetchMgr   *streaming.PrefetchManager
	smartPrefetch *SmartPrefetchManager
	localConfig   *localconfig.LocalConfigServer
}

// New creates a new Edge agent, initializing all sub-components.
func New(cfg *config.EdgeAgentConfig) (*Edge, error) {
	cache, err := NewCache(cfg.CacheDir, cfg.CacheMaxGB)
	if err != nil {
		return nil, fmt.Errorf("create cache: %w", err)
	}

	fetcher := NewFetcher(cfg)
	server := NewServer(cache, fetcher, cfg)
	reporter := NewReporter(cfg, server, cache)

	edge := &Edge{
		cfg:      cfg,
		cache:    cache,
		fetcher:  fetcher,
		server:   server,
		reporter: reporter,
	}

	// Initialize streaming components (v0.4)
	if cfg.Streaming != nil && cfg.Streaming.Enabled {
		window := streaming.NewSlidingWindow(cache, cfg.Streaming)
		prefetchMgr := streaming.NewPrefetchManager(cfg.Streaming, window, cfg.OriginURL, cfg.NodeToken)
		streamH := streaming.NewHandler(window, prefetchMgr, cfg.Streaming)
		server.WithStreaming(streamH)
		edge.prefetchMgr = prefetchMgr
		slog.Info("streaming mode enabled",
			"window_size", cfg.Streaming.WindowSize,
			"prefetch_count", cfg.Streaming.PrefetchCount,
		)
	}

	// Auto-generate TLS certificate for IP-based endpoints (v0.7+)
	if cfg.TLSAutoCert {
		if err := os.MkdirAll(cfg.TLSCertDir, 0o700); err != nil {
			slog.Warn("failed to create TLS cert dir", "dir", cfg.TLSCertDir, "err", err)
		} else {
			ips := collectLocalIPs()
			certFile := cfg.TLSCertDir + "/cert.pem"
			keyFile := cfg.TLSCertDir + "/key.pem"
			_, err := autotls.LoadOrGenerate(ips, nil, certFile, keyFile)
			if err != nil {
				slog.Warn("failed to generate TLS cert", "err", err)
			} else {
				cfg.TLSCertFile = certFile
				cfg.TLSKeyFile = keyFile
				cfg.TLSEnabled = true
				slog.Info("TLS auto-cert generated", "ips", fmtIPs(ips), "cert", certFile)
			}
		}
	}

	// Initialize tunnel client if in NAT mode
	if cfg.NATMode && cfg.TunnelServerAddr != "" {
		tunnelCfg := tunnel.ClientConfig{
			ServerAddr:    cfg.TunnelServerAddr,
			NodeID:        "", // Will be set after registration
			NodeToken:     cfg.TunnelAuthToken,
			ServiceAddr:   cfg.ListenAddr,
			KeepAlive:     30 * time.Second,
			ReconnectWait: 5 * time.Second,
		}
		edge.tunnelClient = tunnel.NewClient(tunnelCfg, slog.Default())
		slog.Info("NAT mode enabled, tunnel client configured",
			"tunnel_server", cfg.TunnelServerAddr,
		)
	}

	// Initialize smart prefetch (v0.6+)
	if cfg.PrefetchEnabled {
		workers := cfg.PrefetchWorkers
		if workers <= 0 {
			workers = 2
		}
		bwLimit := cfg.PrefetchBandwidthLimit
		if bwLimit <= 0 {
			bwLimit = 20
		}
		edge.smartPrefetch = NewSmartPrefetchManager(cache, fetcher, workers, bwLimit)
		if cfg.PrefetchNightModeStart > 0 {
			edge.smartPrefetch.SetNightMode(
				cfg.PrefetchNightModeStart,
				cfg.PrefetchNightModeEnd,
				cfg.PrefetchBandwidthLimit,
			)
		}
		slog.Info("smart prefetch enabled",
			"workers", workers,
			"bw_limit", bwLimit,
			"night_start", cfg.PrefetchNightModeStart,
			"night_end", cfg.PrefetchNightModeEnd,
		)
	}

	// Initialize local config web UI (v0.6+)
	edge.localConfig = localconfig.NewLocalConfigServer(cfg, server)

	return edge, nil
}

// Start runs the edge agent: registers if needed, then starts server and reporter.
func (e *Edge) Start(ctx context.Context) error {
	// Register with control plane to get a NodeID
	slog.Info("registering with control plane")
	if err := e.reporter.Register(ctx); err != nil {
		return fmt.Errorf("register: %w", err)
	}

	// Update tunnel client with node ID after registration
	if e.tunnelClient != nil {
		e.tunnelClient = tunnel.NewClient(tunnel.ClientConfig{
			ServerAddr:    e.cfg.TunnelServerAddr,
			NodeID:        e.reporter.NodeID(),
			NodeToken:     e.cfg.TunnelAuthToken,
			ServiceAddr:   e.cfg.ListenAddr,
			KeepAlive:     30 * time.Second,
			ReconnectWait: 5 * time.Second,
		}, slog.Default())
	}

	// Start tunnel client if in NAT mode
	if e.tunnelClient != nil {
		go func() {
			slog.Info("starting tunnel client")
			if err := e.tunnelClient.Run(); err != nil {
				slog.Error("tunnel client error", "err", err)
			}
		}()
	}

	// Start the reporter (heartbeats)
	if err := e.reporter.Start(ctx); err != nil {
		return fmt.Errorf("start reporter: %w", err)
	}

	// Start prefetch manager (v0.4)
	if e.prefetchMgr != nil {
		e.prefetchMgr.Start(ctx)
	}

	// Start smart prefetch manager (v0.6+)
	if e.smartPrefetch != nil {
		e.smartPrefetch.Start()
	}

	// Start local config web UI (v0.6+)
	if err := e.localConfig.Start(); err != nil {
		slog.Warn("local config UI failed to start", "err", err)
	}

	// Start the HTTP server (blocks until ctx canceled)
	if err := e.server.Start(ctx); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	slog.Info("edge agent started",
		"nat_mode", e.cfg.NATMode,
		"tunnel_enabled", e.tunnelClient != nil,
	)
	return nil
}

// Shutdown gracefully stops the edge agent.
func (e *Edge) Shutdown(ctx context.Context) error {
	// Stop tunnel client first
	if e.tunnelClient != nil {
		e.tunnelClient.Stop()
	}

	e.reporter.Stop()

	// Stop prefetch manager (v0.4)
	if e.prefetchMgr != nil {
		e.prefetchMgr.Stop()
	}

	// Stop smart prefetch manager (v0.6+)
	if e.smartPrefetch != nil {
		e.smartPrefetch.Stop()
	}

	// Shutdown local config UI
	if e.localConfig != nil {
		if err := e.localConfig.Shutdown(ctx); err != nil {
			slog.Warn("local config shutdown error", "err", err)
		}
	}

	if err := e.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}
	return nil
}

// IsNATMode returns true if this node is operating in NAT mode.
func (e *Edge) IsNATMode() bool {
	return e.cfg.NATMode
}

// TunnelID returns the tunnel ID if connected, empty string otherwise.
func (e *Edge) TunnelID() string {
	if e.tunnelClient != nil {
		return e.tunnelClient.TunnelID()
	}
	return ""
}

func collectLocalIPs() []net.IP {
	var ips []net.IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip4 := ipnet.IP.To4(); ip4 != nil && !ip4.IsLoopback() {
				ips = append(ips, ip4)
			}
		}
	}
	return ips
}

func fmtIPs(ips []net.IP) []string {
	s := make([]string, len(ips))
	for i, ip := range ips {
		s[i] = ip.String()
	}
	return s
}
