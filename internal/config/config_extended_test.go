package config

import (
	"testing"
	"time"
)

func TestLoadDNSAdapterOverrides(t *testing.T) {
	t.Setenv("DNS_TOKEN_SECRET", "test-dns-secret")
	t.Setenv("DNS_LISTEN_ADDR", ":5354")
	t.Setenv("DNS_DOMAIN", "cdn.example.com")
	t.Setenv("DNS_TTL_SECONDS", "60")
	t.Setenv("DNS_REFRESH_INTERVAL", "30s")
	t.Setenv("DNS_CONTENT_AWARE_SCORE", "50.0")

	cfg := LoadDNSAdapter()

	if cfg.ListenAddr != ":5354" {
		t.Errorf("ListenAddr = %s, want :5354", cfg.ListenAddr)
	}
	if cfg.Domain != "cdn.example.com" {
		t.Errorf("Domain = %s, want cdn.example.com", cfg.Domain)
	}
	if cfg.TTLSeconds != 60 {
		t.Errorf("TTLSeconds = %d, want 60", cfg.TTLSeconds)
	}
	if cfg.RefreshInterval != 30*time.Second {
		t.Errorf("RefreshInterval = %v, want 30s", cfg.RefreshInterval)
	}
	if cfg.ContentAwareScore != 50.0 {
		t.Errorf("ContentAwareScore = %f, want 50.0", cfg.ContentAwareScore)
	}
}

func TestEdgeAgentQuicConfig(t *testing.T) {
	t.Setenv("EA_TOKEN_SECRET", "test-secret")
	t.Setenv("EA_QUIC_ENABLED", "true")
	t.Setenv("EA_QUIC_LISTEN_ADDR", ":9444")

	cfg := LoadEdgeAgent()

	if !cfg.Quic.Enabled {
		t.Error("Quic.Enabled should be true")
	}
	if cfg.Quic.ListenAddr != ":9444" {
		t.Errorf("Quic.ListenAddr = %s, want :9444", cfg.Quic.ListenAddr)
	}
}

func TestEdgeAgentSmallBandwidthConfig(t *testing.T) {
	t.Setenv("EA_TOKEN_SECRET", "test-secret")
	t.Setenv("EA_MAX_UPLINK_MBPS", "100")
	t.Setenv("EA_P2P_ENABLED", "true")
	t.Setenv("EA_P2P_DISCOVERY_INTERVAL", "120")
	t.Setenv("EA_P2P_MAX_PEERS", "20")
	t.Setenv("EA_PREFETCH_ENABLED", "true")
	t.Setenv("EA_PREFETCH_WORKERS", "4")
	t.Setenv("EA_PREFETCH_BANDWIDTH_LIMIT", "50")
	t.Setenv("EA_ORIGIN_FETCH_BW_LIMIT", "200")
	t.Setenv("EA_TLS_ENABLED", "true")

	cfg := LoadEdgeAgent()

	if cfg.MaxUplinkMbps != 100 {
		t.Errorf("MaxUplinkMbps = %d, want 100", cfg.MaxUplinkMbps)
	}
	if !cfg.P2PEnabled {
		t.Error("P2PEnabled should be true")
	}
	if cfg.P2PDiscoveryIntervalSec != 120 {
		t.Errorf("P2PDiscoveryIntervalSec = %d, want 120", cfg.P2PDiscoveryIntervalSec)
	}
	if cfg.P2PMaxPeers != 20 {
		t.Errorf("P2PMaxPeers = %d, want 20", cfg.P2PMaxPeers)
	}
	if !cfg.PrefetchEnabled {
		t.Error("PrefetchEnabled should be true")
	}
	if cfg.PrefetchWorkers != 4 {
		t.Errorf("PrefetchWorkers = %d, want 4", cfg.PrefetchWorkers)
	}
	if cfg.PrefetchBandwidthLimit != 50 {
		t.Errorf("PrefetchBandwidthLimit = %d, want 50", cfg.PrefetchBandwidthLimit)
	}
	if cfg.OriginFetchBWLimit != 200 {
		t.Errorf("OriginFetchBWLimit = %d, want 200", cfg.OriginFetchBWLimit)
	}
	if !cfg.TLSEnabled {
		t.Error("TLSEnabled should be true")
	}
}

func TestEdgeAgentPrefetchNightMode(t *testing.T) {
	t.Setenv("EA_TOKEN_SECRET", "test-secret")
	t.Setenv("EA_PREFETCH_NIGHT_MODE_START", "23")
	t.Setenv("EA_PREFETCH_NIGHT_MODE_END", "5")

	cfg := LoadEdgeAgent()

	if cfg.PrefetchNightModeStart != 23 {
		t.Errorf("PrefetchNightModeStart = %d, want 23", cfg.PrefetchNightModeStart)
	}
	if cfg.PrefetchNightModeEnd != 5 {
		t.Errorf("PrefetchNightModeEnd = %d, want 5", cfg.PrefetchNightModeEnd)
	}
}

func TestEdgeAgentWSGRPCProxy(t *testing.T) {
	t.Setenv("EA_TOKEN_SECRET", "test-secret")
	t.Setenv("EA_WS_PROXY_ENABLED", "false")
	t.Setenv("EA_GRPC_PROXY_ENABLED", "false")

	cfg := LoadEdgeAgent()

	if cfg.WSProxyEnabled {
		t.Error("WSProxyEnabled should be false")
	}
	if cfg.GRPCProxyEnabled {
		t.Error("GRPCProxyEnabled should be false")
	}
}

func TestEdgeAgentTLSAutoCert(t *testing.T) {
	t.Setenv("EA_TOKEN_SECRET", "test-secret")
	t.Setenv("EA_TLS_AUTO_CERT", "true")
	t.Setenv("EA_TLS_CERT_DIR", "/custom/tls")

	cfg := LoadEdgeAgent()

	if !cfg.TLSAutoCert {
		t.Error("TLSAutoCert should be true")
	}
	if cfg.TLSCertDir != "/custom/tls" {
		t.Errorf("TLSCertDir = %s, want /custom/tls", cfg.TLSCertDir)
	}
}

func TestEdgeAgentDefaultsFalse(t *testing.T) {
	t.Setenv("EA_TOKEN_SECRET", "test-secret")

	cfg := LoadEdgeAgent()
	// These should all default to false
	if cfg.NATMode {
		t.Error("NATMode should default to false")
	}
	if cfg.TLSEnabled {
		t.Error("TLSEnabled should default to false")
	}
	if cfg.TLSAutoCert {
		t.Error("TLSAutoCert should default to false")
	}
}

func TestControlPlaneSmallBandwidthConfig(t *testing.T) {
	t.Setenv("CP_TOKEN_SECRET", "test-secret")
	t.Setenv("CP_SB_OPT_ENABLED", "false")
	t.Setenv("CP_SB_THRESHOLD", "100")
	t.Setenv("CP_SB_P2P_ENABLED", "false")
	t.Setenv("CP_SB_PREFETCH_ENABLED", "false")
	t.Setenv("CP_SB_PREFETCH_WORKERS", "5")
	t.Setenv("CP_SB_PREFETCH_BW_LIMIT", "30")

	cfg := LoadControlPlane()

	if cfg.SmallBandwidthOptimization.Enabled {
		t.Error("SB Enabled should be false")
	}
	if cfg.SmallBandwidthOptimization.SmallBandwidthThreshold != 100 {
		t.Errorf("SmallBandwidthThreshold = %d, want 100", cfg.SmallBandwidthOptimization.SmallBandwidthThreshold)
	}
	if cfg.SmallBandwidthOptimization.P2PEnabled {
		t.Error("P2PEnabled should be false")
	}
	if cfg.SmallBandwidthOptimization.PrefetchEnabled {
		t.Error("PrefetchEnabled should be false")
	}
	if cfg.SmallBandwidthOptimization.PrefetchWorkers != 5 {
		t.Errorf("PrefetchWorkers = %d, want 5", cfg.SmallBandwidthOptimization.PrefetchWorkers)
	}
	if cfg.SmallBandwidthOptimization.PrefetchBandwidthLimit != 30 {
		t.Errorf("PrefetchBandwidthLimit = %d, want 30", cfg.SmallBandwidthOptimization.PrefetchBandwidthLimit)
	}
}

func TestAdminAPIConfigDefaults(t *testing.T) {
	t.Setenv("CP_TOKEN_SECRET", "test-secret")

	cfg := LoadControlPlane()

	if cfg.Admin.Enabled {
		t.Error("Admin.Enabled should default to false")
	}
	if cfg.Admin.JWTExpirySeconds != 3600 {
		t.Errorf("JWTExpirySeconds = %d, want 3600", cfg.Admin.JWTExpirySeconds)
	}
	if cfg.Admin.SessionStoreType != "memory" {
		t.Errorf("SessionStoreType = %s, want memory", cfg.Admin.SessionStoreType)
	}
	if !cfg.Admin.EnableLocalAuth {
		t.Error("EnableLocalAuth should default to true")
	}
}

func TestAdminAPIConfigOverrides(t *testing.T) {
	t.Setenv("CP_TOKEN_SECRET", "test-secret")
	t.Setenv("CP_ADMIN_ENABLED", "true")
	t.Setenv("CP_ADMIN_ACCESS_KEY", "test-access")
	t.Setenv("CP_ADMIN_SECRET_KEY", "test-secret-key")
	t.Setenv("CP_ADMIN_JWT_SECRET", "test-jwt-secret")
	t.Setenv("CP_ADMIN_JWT_EXPIRY", "7200")
	t.Setenv("CP_ADMIN_SESSION_STORE", "redis")
	t.Setenv("CP_ENABLE_MULTITENANCY", "true")
	t.Setenv("CP_ADMIN_OIDC_ENABLED", "true")
	t.Setenv("CP_ADMIN_OIDC_PROVIDER_URL", "https://oidc.example.com")
	t.Setenv("CP_ADMIN_OIDC_CLIENT_ID", "client-id-123")
	t.Setenv("CP_ADMIN_OIDC_CLIENT_SECRET", "oidc-secret")
	t.Setenv("CP_ADMIN_LOCAL_AUTH", "false")
	t.Setenv("CP_ADMIN_GRAFANA_URL", "https://grafana.example.com")
	t.Setenv("CP_ADMIN_PROMETHEUS_URL", "https://prometheus.example.com")
	t.Setenv("CP_ADMIN_LOKI_URL", "https://loki.example.com")

	cfg := LoadControlPlane()

	if !cfg.Admin.Enabled {
		t.Error("Admin.Enabled should be true")
	}
	if cfg.Admin.AdminAccessKey != "test-access" {
		t.Errorf("AdminAccessKey = %s, want test-access", cfg.Admin.AdminAccessKey)
	}
	if cfg.Admin.AdminSecretKey != "test-secret-key" {
		t.Errorf("AdminSecretKey = %s, want test-secret-key", cfg.Admin.AdminSecretKey)
	}
	if cfg.Admin.JWTSecret != "test-jwt-secret" {
		t.Errorf("JWTSecret = %s, want test-jwt-secret", cfg.Admin.JWTSecret)
	}
	if cfg.Admin.JWTExpirySeconds != 7200 {
		t.Errorf("JWTExpirySeconds = %d, want 7200", cfg.Admin.JWTExpirySeconds)
	}
	if cfg.Admin.SessionStoreType != "redis" {
		t.Errorf("SessionStoreType = %s, want redis", cfg.Admin.SessionStoreType)
	}
	if !cfg.Admin.EnableMultiTenancy {
		t.Error("EnableMultiTenancy should be true")
	}
	if !cfg.Admin.EnableOIDC {
		t.Error("EnableOIDC should be true")
	}
	if cfg.Admin.OIDCProviderURL != "https://oidc.example.com" {
		t.Errorf("OIDCProviderURL = %s", cfg.Admin.OIDCProviderURL)
	}
	if cfg.Admin.OIDCClientID != "client-id-123" {
		t.Errorf("OIDCClientID = %s", cfg.Admin.OIDCClientID)
	}
	if cfg.Admin.OIDCClientSecret != "oidc-secret" {
		t.Errorf("OIDCClientSecret = %s", cfg.Admin.OIDCClientSecret)
	}
	if cfg.Admin.EnableLocalAuth {
		t.Error("EnableLocalAuth should be false")
	}
	if cfg.Admin.GrafanaURL != "https://grafana.example.com" {
		t.Errorf("GrafanaURL = %s", cfg.Admin.GrafanaURL)
	}
	if cfg.Admin.PrometheusURL != "https://prometheus.example.com" {
		t.Errorf("PrometheusURL = %s", cfg.Admin.PrometheusURL)
	}
	if cfg.Admin.LokiURL != "https://loki.example.com" {
		t.Errorf("LokiURL = %s", cfg.Admin.LokiURL)
	}
}

func TestContentIndexOverrides(t *testing.T) {
	t.Setenv("CI_BLOOM_CAPACITY", "50000")
	t.Setenv("CI_BLOOM_FP_RATE", "0.001")
	t.Setenv("CI_HOT_KEY_TTL", "10m")
	t.Setenv("CI_CONTENT_AWARE_WEIGHT", "20.0")
	t.Setenv("CI_HOT_CONTENT_WEIGHT", "50.0")

	cfg := LoadContentIndex()

	if cfg.BloomCapacity != 50000 {
		t.Errorf("BloomCapacity = %d, want 50000", cfg.BloomCapacity)
	}
	if cfg.BloomFPRate != 0.001 {
		t.Errorf("BloomFPRate = %f, want 0.001", cfg.BloomFPRate)
	}
	if cfg.HotKeyTTL != 10*time.Minute {
		t.Errorf("HotKeyTTL = %v, want 10m", cfg.HotKeyTTL)
	}
	if cfg.ContentAwareWeight != 20.0 {
		t.Errorf("ContentAwareWeight = %f, want 20.0", cfg.ContentAwareWeight)
	}
	if cfg.HotContentAwareWeight != 50.0 {
		t.Errorf("HotContentAwareWeight = %f, want 50.0", cfg.HotContentAwareWeight)
	}
}

func TestOriginOverrides(t *testing.T) {
	t.Setenv("ORIGIN_LISTEN_ADDR", ":8080")
	t.Setenv("ORIGIN_DATA_DIR", "/custom/origin")
	t.Setenv("ORIGIN_TLS_CERT_FILE", "/path/cert.pem")
	t.Setenv("ORIGIN_TLS_KEY_FILE", "/path/key.pem")

	cfg := LoadOrigin()

	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %s, want :8080", cfg.ListenAddr)
	}
	if cfg.DataDir != "/custom/origin" {
		t.Errorf("DataDir = %s, want /custom/origin", cfg.DataDir)
	}
	if cfg.TLSCertFile != "/path/cert.pem" {
		t.Errorf("TLSCertFile = %s", cfg.TLSCertFile)
	}
	if cfg.TLSKeyFile != "/path/key.pem" {
		t.Errorf("TLSKeyFile = %s", cfg.TLSKeyFile)
	}
}

func TestControlPlaneTLSConfig(t *testing.T) {
	t.Setenv("CP_TOKEN_SECRET", "test-secret")
	t.Setenv("CP_TLS_CERT_FILE", "/path/cp-cert.pem")
	t.Setenv("CP_TLS_KEY_FILE", "/path/cp-key.pem")

	cfg := LoadControlPlane()

	if cfg.TLSCertFile != "/path/cp-cert.pem" {
		t.Errorf("TLSCertFile = %s", cfg.TLSCertFile)
	}
	if cfg.TLSKeyFile != "/path/cp-key.pem" {
		t.Errorf("TLSKeyFile = %s", cfg.TLSKeyFile)
	}
}

func TestControlPlaneRedis(t *testing.T) {
	t.Setenv("CP_TOKEN_SECRET", "test-secret")
	t.Setenv("CP_REDIS_PASSWORD", "redispass")
	t.Setenv("CP_ORIGIN_URL", "https://origin.example.com")

	cfg := LoadControlPlane()

	if cfg.RedisPassword != "redispass" {
		t.Errorf("RedisPassword = %s, want redispass", cfg.RedisPassword)
	}
	if cfg.OriginURL != "https://origin.example.com" {
		t.Errorf("OriginURL = %s, want https://origin.example.com", cfg.OriginURL)
	}
}

func TestEdgeAgentEdgeCases(t *testing.T) {
	t.Setenv("EA_TOKEN_SECRET", "test-secret")
	t.Setenv("EA_MAX_CONNS", "5000")
	t.Setenv("EA_NAT_MODE", "true")
	t.Setenv("EA_TUNNEL_SERVER_ADDR", "tunnel.example.com:7700")
	t.Setenv("EA_TUNNEL_AUTH_TOKEN", "tunnel-auth-token")
	t.Setenv("EA_PUBLIC_HOST", "edge.example.com")
	t.Setenv("EA_REGION", "cn-sh")
	t.Setenv("EA_ISP", "ctcc")

	cfg := LoadEdgeAgent()

	if cfg.MaxConns != 5000 {
		t.Errorf("MaxConns = %d, want 5000", cfg.MaxConns)
	}
	if !cfg.NATMode {
		t.Error("NATMode should be true")
	}
	if cfg.TunnelServerAddr != "tunnel.example.com:7700" {
		t.Errorf("TunnelServerAddr = %s", cfg.TunnelServerAddr)
	}
	if cfg.TunnelAuthToken != "tunnel-auth-token" {
		t.Errorf("TunnelAuthToken = %s", cfg.TunnelAuthToken)
	}
	if cfg.PublicHost != "edge.example.com" {
		t.Errorf("PublicHost = %s", cfg.PublicHost)
	}
	if cfg.Region != "cn-sh" {
		t.Errorf("Region = %s", cfg.Region)
	}
	if cfg.ISP != "ctcc" {
		t.Errorf("ISP = %s", cfg.ISP)
	}
}

func TestGatewayQuicConfig(t *testing.T) {
	t.Setenv("GW_AUTH_TOKEN", "test-auth")
	t.Setenv("GW_CP_TOKEN", "test-cp")
	t.Setenv("GW_QUIC_ENABLED", "true")
	t.Setenv("GW_QUIC_LISTEN_ADDR", ":9445")

	cfg := LoadGateway()

	if !cfg.Quic.Enabled {
		t.Error("Quic.Enabled should be true")
	}
	if cfg.Quic.ListenAddr != ":9445" {
		t.Errorf("Quic.ListenAddr = %s, want :9445", cfg.Quic.ListenAddr)
	}
}

func TestControlPlaneContentIndex(t *testing.T) {
	t.Setenv("CP_TOKEN_SECRET", "test-secret")
	t.Setenv("CP_CI_BLOOM_CAPACITY", "20000")
	t.Setenv("CP_CI_HOT_KEY_TTL", "10m")

	cfg := LoadControlPlane()

	if cfg.ContentIndex.BloomCapacity != 20000 {
		t.Errorf("BloomCapacity = %d, want 20000", cfg.ContentIndex.BloomCapacity)
	}
	if cfg.ContentIndex.HotKeyTTL != 10*time.Minute {
		t.Errorf("HotKeyTTL = %v, want 10m", cfg.ContentIndex.HotKeyTTL)
	}
}
