package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/im10furry/edge-dispatch-framework/internal/auth"
	"github.com/im10furry/edge-dispatch-framework/internal/config"
	"github.com/im10furry/edge-dispatch-framework/internal/contentindex"
	"github.com/im10furry/edge-dispatch-framework/internal/controlplane"
	"github.com/im10furry/edge-dispatch-framework/internal/controlplane/adminui"
	"github.com/im10furry/edge-dispatch-framework/internal/store"
	"github.com/im10furry/edge-dispatch-framework/internal/tracing"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("edge-dispatch control-plane v%s (commit %s, built %s)\n", Version, Commit, BuildDate)
		return
	}

	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--version" {
			fmt.Printf("edge-dispatch control-plane v%s (commit %s, built %s)\n", Version, Commit, BuildDate)
			return
		}
	}
	debug.SetGCPercent(80)
	debug.SetMemoryLimit(1 << 30)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	logger.Info("starting control plane")

	cfg := config.LoadControlPlane()

	ctx := context.Background()

	// Initialize OpenTelemetry tracing (v0.9+)
	otelShutdown, err := tracing.Init(ctx, tracing.Config{
		Enabled:      cfg.OTELEnabled,
		OTLPEndpoint: cfg.OTELPEndpoint,
		ServiceName:  cfg.OTELServiceName,
		SampleRate:   cfg.OTELSampleRate,
	})
	if err != nil {
		logger.Error("failed to initialize tracing", "error", err)
		os.Exit(1)
	}
	defer otelShutdown(ctx)

	// Initialize PostgreSQL store
	logger.Info("connecting to postgres", "host", sanitizePGURL(cfg.PGURL))
	pgStore, err := store.NewPGStore(ctx, cfg.PGURL)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pgStore.Close()

	// Initialize Redis store
	logger.Info("connecting to redis", "addr", cfg.RedisAddr)
	redisStore, err := store.NewRedisStore(ctx, cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisStore.Close()

	// Create a shared signer for token operations
	signer := auth.NewSigner(cfg.TokenSecret)

	// Create control plane components
	registry := controlplane.NewRegistry(pgStore, redisStore)
	heartbeat := controlplane.NewHeartbeat(pgStore, redisStore, cfg)
	nodeCache := controlplane.NewNodeCache(pgStore, cfg.NodeCacheTTL)
	prober := controlplane.NewProber(pgStore, nodeCache, cfg)
	scheduler := controlplane.NewScheduler(nodeCache, signer, cfg)

	// Content index (v0.2+)
	ciStore, err := contentindex.NewStore(ctx, pgStore.Pool(), &cfg.ContentIndex)
	if err != nil {
		logger.Error("failed to create content index store", "error", err)
		os.Exit(1)
	}
	defer ciStore.Close()
	heartbeat.SetContentStore(ciStore)
	scheduler.SetContentIndex(ciStore.Index())
	scheduler.SetRedis(redisStore)

	// Load existing content index data into memory
	if err := ciStore.LoadAll(ctx); err != nil {
		logger.Warn("failed to load content index from db", "error", err)
	}

	// Create and start task executor for async cache operations (v0.7+)
	taskExecutor := controlplane.NewTaskExecutor(pgStore, scheduler)

	// Initialize Admin API (v0.5)
	var adminHandler http.Handler
	if cfg.Admin.Enabled {
		adminHandler, err = controlplane.NewAdminAPI(pgStore, redisStore, registry, scheduler, &cfg.Admin)
		if err != nil {
			logger.Error("failed to create admin API", "error", err)
			os.Exit(1)
		}
		if adminHandler != nil {
			logger.Info("admin API enabled", "jwt_expiry_seconds", cfg.Admin.JWTExpirySeconds)
		}
	}

	// Start background tasks
	ctxBg, cancelBg := context.WithCancel(context.Background())
	defer cancelBg()

	// Create API handler
	apiHandler := controlplane.NewAPI(registry, heartbeat, scheduler, cfg)

	// Set health checker for structured health endpoint (v0.9+)
	apiHandler.SetHealthChecker(&store.CombinedHealthChecker{PG: pgStore, Redis: redisStore})

	handler := http.Handler(apiHandler)

	// Start policy sync from DB (v0.9+)
	apiHandler.StartPolicySync(ctxBg, pgStore)

	// Tenant rate limiter (v0.8+)
	tenantRL := controlplane.NewTenantRateLimiter(redisStore.Client())
	handler = tenantRL.Middleware(handler)

	leader := controlplane.NewLeaderElection(redisStore.Client())
	go leader.Start(ctxBg)
	time.Sleep(100 * time.Millisecond)

	if leader.IsLeader() {
		heartbeat.Start(ctxBg)
		prober.Start(ctxBg)
		go taskExecutor.Start(ctxBg)
		slog.Info("running as leader — background tasks started")
	} else {
		slog.Info("running as follower — background tasks deferred to leader")
	}

	go func() {
		wasLeader := leader.IsLeader()
		for {
			time.Sleep(1 * time.Second)
			select {
			case <-ctxBg.Done():
				return
			default:
			}
			isNow := leader.IsLeader()
			if isNow && !wasLeader {
				slog.Info("promoted to leader — starting background tasks")
				heartbeat.Start(ctxBg)
				prober.Start(ctxBg)
				go taskExecutor.Start(ctxBg)
			}
			if !isNow && wasLeader {
				slog.Info("demoted to follower — background tasks will be run by new leader")
			}
			wasLeader = isNow
		}
	}()

	spaHandler := adminui.SPAHandler()

	combinedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/internal/admin/v1") && adminHandler != nil {
			http.StripPrefix("/internal/admin/v1", adminHandler).ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(path, "/admin/") || path == "/admin" {
			spaHandler.ServeHTTP(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           combinedHandler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	go func() {
		logger.Info("http server listening", "addr", cfg.ListenAddr)
		var err error
		if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			err = srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("received shutdown signal", "signal", sig.String())

	cancelBg()
	prober.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server forced shutdown", "error", err)
	}

	logger.Info("control plane stopped")
}

// sanitizePGURL returns a redacted PostgreSQL URL safe for logging.
func sanitizePGURL(raw string) string {
	if u, err := url.Parse(raw); err == nil {
		return u.Host
	}
	return raw
}
