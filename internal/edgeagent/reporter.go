package edgeagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/im10furry/edge-dispatch-framework/internal/config"
	"github.com/im10furry/edge-dispatch-framework/internal/contentindex"
	"github.com/im10furry/edge-dispatch-framework/internal/models"
)

type Reporter struct {
	cfg     *config.EdgeAgentConfig
	server  *Server
	cache   *Cache
	client  *http.Client
	nodeID  string
	stopCh  chan struct{}
	stopped bool
	mu      sync.RWMutex
}

func NewReporter(cfg *config.EdgeAgentConfig, server *Server, cache *Cache) *Reporter {
	return &Reporter{
		cfg:    cfg,
		server: server,
		cache:  cache,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		stopCh: make(chan struct{}),
	}
}

func (r *Reporter) SetNodeID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodeID = id
}

func (r *Reporter) NodeID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.nodeID
}

func (r *Reporter) Start(ctx context.Context) error {
	go r.reportLoop(ctx)
	return nil
}

func (r *Reporter) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.stopped {
		r.stopped = true
		close(r.stopCh)
	}
}

func (r *Reporter) reportLoop(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.HeartbeatInterval)
	defer ticker.Stop()

	slog.Info("reporter started", "interval", r.cfg.HeartbeatInterval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("reporter context canceled")
			return
		case <-r.stopCh:
			slog.Info("reporter stopped")
			return
		case <-ticker.C:
			if err := r.ReportOnce(ctx); err != nil {
				slog.Error("heartbeat failed", "err", err)
			}
		}
	}
}

func (r *Reporter) ReportOnce(ctx context.Context) error {
	r.mu.Lock()
	nid := r.nodeID
	r.mu.Unlock()

	if nid == "" {
		return fmt.Errorf("node not yet registered")
	}

	cs := r.cache.Stats()
	hitRatio := float64(0)
	if cs.Hits+cs.Misses > 0 {
		hitRatio = float64(cs.Hits) / float64(cs.Hits+cs.Misses)
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	delta := r.server.GetMetricsDelta()
	ingressMbps, bwEgressMbps := r.server.BWMeter().Snapshot()
	intervalSec := r.cfg.HeartbeatInterval.Seconds()

	var egressMbps float64
	var errRate float64
	egressMbps = bwEgressMbps
	if egressMbps <= 0 && intervalSec > 0 {
		egressMbps = float64(delta.BytesSent*8) / (intervalSec * 1e6)
	}
	if delta.Requests > 0 {
		errRate = float64(delta.Errors) / float64(delta.Requests)
	}
	cpuLoad := float64(runtime.NumGoroutine()) / float64(runtime.NumCPU())

	hb := models.HeartbeatRequest{
		NodeID: nid,
		TS:     time.Now().Unix(),
		Runtime: models.NodeRuntime{
			CPU:        cpuLoad,
			MemMB:      int64(memStats.Alloc / (1024 * 1024)),
			DiskFreeGB: cs.MaxGB - cs.Size/(1024*1024*1024),
			Conn:       r.server.RequestCount(),
		},
		Traffic: models.NodeTraffic{
			EgressMbps:  egressMbps,
			IngressMbps: ingressMbps,
			Err5xxRate:  errRate,
		},
		Cache: models.NodeCache{
			HitRatio: hitRatio,
		},
		ContentSummary: r.buildContentSummary(),
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	if err := json.NewEncoder(buf).Encode(hb); err != nil {
		return fmt.Errorf("marshal heartbeat: %w", err)
	}

	url := r.cfg.ControlPlaneURL + "/v1/nodes/heartbeat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return fmt.Errorf("create heartbeat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if r.cfg.NodeToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.cfg.NodeToken)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("heartbeat rejected: %d", resp.StatusCode)
	}

	slog.Debug("heartbeat sent", "node_id", nid, "hit_ratio", fmt.Sprintf("%.2f", hitRatio))
	return nil
}

func (r *Reporter) Register(ctx context.Context) error {
	caps := r.collectCapabilities()

	host := r.cfg.PublicHost
	if host == "" {
		host = r.detectPublicIP()
	}
	region := r.cfg.Region
	isp := r.cfg.ISP
	if region == "" || isp == "" || region == "auto" || isp == "auto" {
		geo := r.detectGeoIP(host)
		if region == "" || region == "auto" {
			region = geo.Region
		}
		if isp == "" || isp == "auto" {
			isp = geo.ISP
		}
		slog.Info("geoip detected", "region", region, "isp", isp, "country", geo.Country)
	}

	regReq := models.RegisterRequest{
		NodeName:     "edge-agent",
		Endpoints:    []models.Endpoint{{Scheme: "http", Host: host, Port: parsePort(r.cfg.ListenAddr)}},
		Region:       region,
		ISP:          isp,
		Capabilities: caps,
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	if err := json.NewEncoder(buf).Encode(regReq); err != nil {
		return fmt.Errorf("marshal register: %w", err)
	}

	url := r.cfg.ControlPlaneURL + "/v1/nodes/register"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return fmt.Errorf("create register request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if r.cfg.NodeToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.cfg.NodeToken)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("register rejected: %d", resp.StatusCode)
	}

	var regResp models.RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("decode register response: %w", err)
	}

	r.SetNodeID(regResp.NodeID)
	slog.Info("node registered", "node_id", regResp.NodeID)
	return nil
}

func (r *Reporter) collectCapabilities() models.Capabilities {
	maxBW := r.cfg.MaxUplinkMbps
	if maxBW <= 0 {
		maxBW = 100
	}
	return models.Capabilities{
		InboundReachable: true,
		CacheDiskGB:      r.cfg.CacheMaxGB,
		MaxUplinkMbps:    maxBW,
		SupportsHTTPS:    false,
		SupportsP2P:      r.cfg.P2PEnabled,
		ShieldMode:       r.cfg.ShieldMode,
	}
}

func (r *Reporter) buildContentSummary() *models.ContentSummary {
	allKeys := r.cache.AllKeys()
	if len(allKeys) == 0 {
		return nil
	}

	hotKeys := r.cache.HotKeys(20)

	bf := contentindex.NewBloomFilter(max(len(allKeys), 100), 0.05)
	for _, k := range allKeys {
		bf.AddString(k)
	}

	return &models.ContentSummary{
		NodeID:      r.nodeID,
		HotKeys:     hotKeys,
		BloomFilter: bf.Bytes(),
		TotalKeys:   int64(len(allKeys)),
		UpdatedAt:   time.Now().Unix(),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func parsePort(addr string) int {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 9090
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 9090
	}
	return port
}

func (r *Reporter) detectPublicIP() string {
	services := []string{
		"https://api.ipify.org",
		"https://icanhazip.com",
		"https://ifconfig.me/ip",
	}
	client := &http.Client{Timeout: 5 * time.Second}
	for _, svc := range services {
		resp, err := client.Get(svc)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
		resp.Body.Close()
		if err != nil {
			continue
		}
		ip := strings.TrimSpace(string(body))
		if net.ParseIP(ip) != nil {
			slog.Info("detected public IP", "ip", ip, "service", svc)
			return ip
		}
	}
	slog.Warn("could not detect public IP, using localhost")
	return "localhost"
}

type GeoIPResult struct {
	Country string `json:"country"`
	Region  string `json:"regionName"`
	City    string `json:"city"`
	ISP     string `json:"isp"`
}

func (r *Reporter) detectGeoIP(ip string) GeoIPResult {
	if ip == "" || ip == "localhost" || net.ParseIP(ip) == nil {
		return GeoIPResult{Region: "unknown", ISP: "unknown"}
	}
	if net.ParseIP(ip).IsPrivate() {
		return GeoIPResult{Region: "local", ISP: "local"}
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://ip-api.com/json/" + ip + "?fields=country,regionName,city,isp")
	if err != nil {
		slog.Warn("geoip lookup failed", "err", err)
		return GeoIPResult{Region: "unknown", ISP: "unknown"}
	}
	defer resp.Body.Close()

	var geo GeoIPResult
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		slog.Warn("geoip decode failed", "err", err)
		return GeoIPResult{Region: "unknown", ISP: "unknown"}
	}

	region := geo.Region
	if region == "" {
		region = geo.City
	}
	if region == "" {
		region = geo.Country
	}
	return GeoIPResult{
		Country: geo.Country,
		Region:  region,
		ISP:     geo.ISP,
	}
}

type BandwidthMeter struct {
	bytesIn     atomic.Int64
	bytesOut    atomic.Int64
	lastIn      int64
	lastOut     int64
	lastCheck   time.Time
	mu          sync.Mutex
	egressMbps  float64
	ingressMbps float64
}

func NewBandwidthMeter() *BandwidthMeter {
	return &BandwidthMeter{lastCheck: time.Now()}
}

func (b *BandwidthMeter) AddIn(bytes int64)  { b.bytesIn.Add(bytes) }
func (b *BandwidthMeter) AddOut(bytes int64) { b.bytesOut.Add(bytes) }

func (b *BandwidthMeter) Snapshot() (ingressMbps, egressMbps float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastCheck).Seconds()
	currentIn := b.bytesIn.Load()
	currentOut := b.bytesOut.Load()

	if elapsed > 0 {
		b.ingressMbps = float64(currentIn-b.lastIn) * 8 / elapsed / 1e6
		b.egressMbps = float64(currentOut-b.lastOut) * 8 / elapsed / 1e6
	}

	b.lastIn = currentIn
	b.lastOut = currentOut
	b.lastCheck = now

	if b.ingressMbps < 0 {
		b.ingressMbps = 0
	}
	if b.egressMbps < 0 {
		b.egressMbps = 0
	}
	return b.ingressMbps, b.egressMbps
}

func (b *BandwidthMeter) Current() (ingressMbps, egressMbps float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.ingressMbps, b.egressMbps
}
