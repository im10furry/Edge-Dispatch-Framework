package config

import (
	"testing"
	"time"
)

func TestLoadControlPlaneDefaults(t *testing.T) {
	t.Setenv("CP_TOKEN_SECRET", "test-secret-not-default")
	t.Setenv("CP_ADMIN_JWT_SECRET", "test-jwt-not-default")

	cfg := LoadControlPlane()

	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %s, want :8080", cfg.ListenAddr)
	}
	if cfg.MaxCandidates != 5 {
		t.Errorf("MaxCandidates = %d, want 5", cfg.MaxCandidates)
	}
	if cfg.DefaultTTLMs != 30000 {
		t.Errorf("DefaultTTLMs = %d, want 30000", cfg.DefaultTTLMs)
	}
	if !cfg.DegradeToOrigin {
		t.Error("DegradeToOrigin should default to true")
	}
	if cfg.HeartbeatTTL != 30*time.Second {
		t.Errorf("HeartbeatTTL = %v, want 30s", cfg.HeartbeatTTL)
	}
	if cfg.ProbeInterval != 10*time.Second {
		t.Errorf("ProbeInterval = %v, want 10s", cfg.ProbeInterval)
	}
	if cfg.ProbeTimeout != 5*time.Second {
		t.Errorf("ProbeTimeout = %v, want 5s", cfg.ProbeTimeout)
	}
	if cfg.NodeCacheTTL != 30*time.Second {
		t.Errorf("NodeCacheTTL = %v, want 30s", cfg.NodeCacheTTL)
	}
}

func TestLoadControlPlaneOverrides(t *testing.T) {
	t.Setenv("CP_TOKEN_SECRET", "test-secret")
	t.Setenv("CP_LISTEN_ADDR", ":9090")
	t.Setenv("CP_MAX_CANDIDATES", "10")
	t.Setenv("CP_DEFAULT_TTL_MS", "60000")
	t.Setenv("CP_DEGRADE_TO_ORIGIN", "false")
	t.Setenv("CP_HEARTBEAT_TTL", "60s")
	t.Setenv("CP_PROBE_INTERVAL", "30s")
	t.Setenv("CP_NODE_CACHE_TTL", "60s")

	cfg := LoadControlPlane()

	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %s, want :9090", cfg.ListenAddr)
	}
	if cfg.MaxCandidates != 10 {
		t.Errorf("MaxCandidates = %d, want 10", cfg.MaxCandidates)
	}
	if cfg.DefaultTTLMs != 60000 {
		t.Errorf("DefaultTTLMs = %d, want 60000", cfg.DefaultTTLMs)
	}
	if cfg.DegradeToOrigin {
		t.Error("DegradeToOrigin should be false")
	}
	if cfg.HeartbeatTTL != 60*time.Second {
		t.Errorf("HeartbeatTTL = %v, want 60s", cfg.HeartbeatTTL)
	}
	if cfg.ProbeInterval != 30*time.Second {
		t.Errorf("ProbeInterval = %v, want 30s", cfg.ProbeInterval)
	}
	if cfg.NodeCacheTTL != 60*time.Second {
		t.Errorf("NodeCacheTTL = %v, want 60s", cfg.NodeCacheTTL)
	}
}

func TestLoadEdgeAgentDefaults(t *testing.T) {
	t.Setenv("EA_TOKEN_SECRET", "test-secret")

	cfg := LoadEdgeAgent()

	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %s, want :9090", cfg.ListenAddr)
	}
	if cfg.OriginURL != "http://localhost:7070" {
		t.Errorf("OriginURL = %s, want http://localhost:7070", cfg.OriginURL)
	}
	if cfg.ControlPlaneURL != "http://localhost:8080" {
		t.Errorf("ControlPlaneURL = %s, want http://localhost:8080", cfg.ControlPlaneURL)
	}
	if cfg.HeartbeatInterval != 10*time.Second {
		t.Errorf("HeartbeatInterval = %v, want 10s", cfg.HeartbeatInterval)
	}
	if cfg.CacheMaxGB != 10 {
		t.Errorf("CacheMaxGB = %d, want 10", cfg.CacheMaxGB)
	}
	if cfg.CacheDir != "/tmp/edf-cache" {
		t.Errorf("CacheDir = %s, want /tmp/edf-cache", cfg.CacheDir)
	}
}

func TestLoadEdgeAgentOverrides(t *testing.T) {
	t.Setenv("EA_TOKEN_SECRET", "test-secret")
	t.Setenv("EA_LISTEN_ADDR", ":9091")
	t.Setenv("EA_ORIGIN_URL", "http://custom-origin:8080")
	t.Setenv("EA_HEARTBEAT_INTERVAL", "15s")
	t.Setenv("EA_CACHE_MAX_GB", "50")
	t.Setenv("EA_CACHE_DIR", "/tmp/edge-cache")
	t.Setenv("EA_NAT_MODE", "true")

	cfg := LoadEdgeAgent()

	if cfg.ListenAddr != ":9091" {
		t.Errorf("ListenAddr = %s, want :9091", cfg.ListenAddr)
	}
	if cfg.OriginURL != "http://custom-origin:8080" {
		t.Errorf("OriginURL = %s, want http://custom-origin:8080", cfg.OriginURL)
	}
	if cfg.HeartbeatInterval != 15*time.Second {
		t.Errorf("HeartbeatInterval = %v, want 15s", cfg.HeartbeatInterval)
	}
	if cfg.CacheMaxGB != 50 {
		t.Errorf("CacheMaxGB = %d, want 50", cfg.CacheMaxGB)
	}
	if cfg.CacheDir != "/tmp/edge-cache" {
		t.Errorf("CacheDir = %s, want /tmp/edge-cache", cfg.CacheDir)
	}
	if !cfg.NATMode {
		t.Error("NATMode should be true")
	}
}

func TestLoadGatewayDefaults(t *testing.T) {
	t.Setenv("GW_AUTH_TOKEN", "test-auth-token")
	t.Setenv("GW_CP_TOKEN", "test-cp-token")

	cfg := LoadGateway()

	if cfg.ListenAddr != ":8880" {
		t.Errorf("ListenAddr = %s, want :8880", cfg.ListenAddr)
	}
	if cfg.TunnelAddr != ":7700" {
		t.Errorf("TunnelAddr = %s, want :7700", cfg.TunnelAddr)
	}
	if cfg.ControlPlaneURL != "http://localhost:8080" {
		t.Errorf("ControlPlaneURL = %s, want http://localhost:8080", cfg.ControlPlaneURL)
	}
	if cfg.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout = %v, want 30s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 60*time.Second {
		t.Errorf("WriteTimeout = %v, want 60s", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 120*time.Second {
		t.Errorf("IdleTimeout = %v, want 2m0s", cfg.IdleTimeout)
	}
}

func TestLoadGatewayOverrides(t *testing.T) {
	t.Setenv("GW_AUTH_TOKEN", "test-token")
	t.Setenv("GW_CP_TOKEN", "test-cp-token")
	t.Setenv("GW_LISTEN_ADDR", ":9999")
	t.Setenv("GW_TUNNEL_ADDR", ":9998")
	t.Setenv("GW_READ_TIMEOUT", "10s")
	t.Setenv("GW_WRITE_TIMEOUT", "15s")
	t.Setenv("GW_IDLE_TIMEOUT", "30s")
	t.Setenv("GW_QUIC_ENABLED", "true")

	cfg := LoadGateway()

	if cfg.ListenAddr != ":9999" {
		t.Errorf("ListenAddr = %s, want :9999", cfg.ListenAddr)
	}
	if cfg.TunnelAddr != ":9998" {
		t.Errorf("TunnelAddr = %s, want :9998", cfg.TunnelAddr)
	}
	if cfg.ReadTimeout != 10*time.Second {
		t.Errorf("ReadTimeout = %v, want 10s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 15*time.Second {
		t.Errorf("WriteTimeout = %v, want 15s", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 30*time.Second {
		t.Errorf("IdleTimeout = %v, want 30s", cfg.IdleTimeout)
	}
	if !cfg.Quic.Enabled {
		t.Error("Quic.Enabled should be true")
	}
}

func TestLoadOriginDefaults(t *testing.T) {
	cfg := LoadOrigin()

	if cfg.ListenAddr != ":7070" {
		t.Errorf("ListenAddr = %s, want :7070", cfg.ListenAddr)
	}
	if cfg.DataDir != "/tmp/edf-origin" {
		t.Errorf("DataDir = %s, want /tmp/edf-origin", cfg.DataDir)
	}
}

func TestLoadDNSAdapterDefaults(t *testing.T) {
	t.Setenv("DNS_TOKEN_SECRET", "test-dns-secret")

	cfg := LoadDNSAdapter()

	if cfg.ListenAddr != ":5353" {
		t.Errorf("ListenAddr = %s, want :5353", cfg.ListenAddr)
	}
	if cfg.ControlPlaneURL != "http://localhost:8080" {
		t.Errorf("ControlPlaneURL = %s, want http://localhost:8080", cfg.ControlPlaneURL)
	}
	if cfg.Domain != "edge.local" {
		t.Errorf("Domain = %s, want edge.local", cfg.Domain)
	}
	if cfg.TTLSeconds != 30 {
		t.Errorf("TTLSeconds = %d, want 30", cfg.TTLSeconds)
	}
	if cfg.RefreshInterval != 10*time.Second {
		t.Errorf("RefreshInterval = %v, want 10s", cfg.RefreshInterval)
	}
}

func TestLoadContentIndexDefaults(t *testing.T) {
	cfg := LoadContentIndex()

	if cfg.BloomCapacity != 10000 {
		t.Errorf("BloomCapacity = %d, want 10000", cfg.BloomCapacity)
	}
	if cfg.BloomFPRate != 0.01 {
		t.Errorf("BloomFPRate = %f, want 0.01", cfg.BloomFPRate)
	}
	if cfg.HotKeyTTL != 5*time.Minute {
		t.Errorf("HotKeyTTL = %v, want 5m", cfg.HotKeyTTL)
	}
}

func TestDefaultStreamingConfig(t *testing.T) {
	cfg := DefaultStreamingConfig()

	if !cfg.Enabled {
		t.Error("Streaming should be enabled by default")
	}
	if cfg.PrefetchCount != 3 {
		t.Errorf("PrefetchCount = %d, want 3", cfg.PrefetchCount)
	}
	if cfg.WindowSize != 60 {
		t.Errorf("WindowSize = %d, want 60", cfg.WindowSize)
	}
}

func TestIntEnvInvalid(t *testing.T) {
	t.Setenv("CP_MAX_CANDIDATES", "not-a-number")
	t.Setenv("CP_TOKEN_SECRET", "test-secret")

	cfg := LoadControlPlane()
	if cfg.MaxCandidates != 5 {
		t.Errorf("MaxCandidates = %d, want default 5 on invalid input", cfg.MaxCandidates)
	}
}

func TestFloatEnvInvalid(t *testing.T) {
	t.Setenv("CI_BLOOM_FP_RATE", "not-a-float")

	cfg := LoadContentIndex()
	if cfg.BloomFPRate != 0.01 {
		t.Errorf("BloomFPRate = %f, want default 0.01 on invalid input", cfg.BloomFPRate)
	}
}

func TestDurationEnvInvalid(t *testing.T) {
	t.Setenv("EA_HEARTBEAT_INTERVAL", "invalid-duration")
	t.Setenv("EA_TOKEN_SECRET", "test-secret")

	cfg := LoadEdgeAgent()
	if cfg.HeartbeatInterval != 10*time.Second {
		t.Errorf("HeartbeatInterval = %v, want default 10s on invalid input", cfg.HeartbeatInterval)
	}
}

func TestBoolEnvInvalid(t *testing.T) {
	t.Setenv("EA_NAT_MODE", "invalid")
	t.Setenv("EA_TOKEN_SECRET", "test-secret")

	cfg := LoadEdgeAgent()
	if cfg.NATMode {
		t.Error("NATMode should default to false on invalid input")
	}
}
