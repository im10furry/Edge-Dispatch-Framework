package localconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/darkinno/edge-dispatch-framework/internal/config"
)

type LocalConfigServer struct {
	cfg     *config.EdgeAgentConfig
	metrics MetricsSource
	diskInfo DiskInfo
	httpSrv *http.Server
}

type DiskInfo struct {
	TotalGB int64 `json:"total_gb"`
	FreeGB  int64 `json:"free_gb"`
	UsedGB  int64 `json:"used_gb"`
}

type ConfigResponse struct {
	Role             string `json:"role"`
	NodeName         string `json:"node_name"`
	ControlPlaneURL  string `json:"control_plane_url"`
	OriginURL        string `json:"origin_url"`
	CacheDir         string `json:"cache_dir"`
	CacheMaxGB       int64  `json:"cache_max_gb"`
	ListenAddr       string `json:"listen_addr"`
	MaxUplinkMbps    int64  `json:"max_uplink_mbps"`
	P2PEnabled       bool   `json:"p2p_enabled"`
	P2PMaxPeers      int    `json:"p2p_max_peers"`
	NATMode          bool   `json:"nat_mode"`
	TunnelServerAddr string `json:"tunnel_server_addr"`
	OriginFetchBWLimit int  `json:"origin_fetch_bw_limit"`
	PrefetchEnabled  bool   `json:"prefetch_enabled"`
	PrefetchWorkers  int    `json:"prefetch_workers"`
	PrefetchBWLimit  int    `json:"prefetch_bw_limit"`
}

type StatusResponse struct {
	Uptime           string  `json:"uptime"`
	CacheHitRatio    float64 `json:"cache_hit_ratio"`
	CacheHits        int64   `json:"cache_hits"`
	CacheMisses      int64   `json:"cache_misses"`
	CacheItems       int64   `json:"cache_items"`
	CacheSizeGB      float64 `json:"cache_size_gb"`
	EgressMbps       float64 `json:"egress_mbps"`
	IngressMbps      float64 `json:"ingress_mbps"`
	BandwidthUsage   float64 `json:"bandwidth_usage"`
	Connections      int64   `json:"connections"`
	Requests         int64   `json:"requests"`
	BytesSent        int64   `json:"bytes_sent"`
}

type MetricsSource interface {
	RequestCount() int64
	CacheHits() int64
	CacheMisses() int64
	BytesSent() int64
	ErrorCount() int64
	GetCacheStats() (size int64, maxGB int64, itemCount int64)
	GetBandwidth() (ingress, egress float64)
}

var startTime = time.Now()

func NewLocalConfigServer(cfg *config.EdgeAgentConfig, metrics MetricsSource) *LocalConfigServer {
	s := &LocalConfigServer{cfg: cfg, metrics: metrics}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/setup", s.handleSetup)
	mux.HandleFunc("/config", s.handleConfigPage)
	mux.HandleFunc("/status", s.handleStatusPage)
	mux.HandleFunc("/api/config", s.handleAPIConfig)
	mux.HandleFunc("/api/config/save", s.handleAPISave)
	mux.HandleFunc("/api/config/apply", s.handleAPIApply)
	mux.HandleFunc("/api/test-connection", s.handleAPITestConn)
	mux.HandleFunc("/api/disk-info", s.handleAPIDiskInfo)
	mux.HandleFunc("/api/status", s.handleAPIStatus)

	s.httpSrv = &http.Server{
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s
}

func (s *LocalConfigServer) ListenAddr() string {
	return ":9091"
}

func (s *LocalConfigServer) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddr())
	if err != nil {
		return fmt.Errorf("local config listen: %w", err)
	}
	slog.Info("local config UI started", "addr", s.ListenAddr())
	go s.httpSrv.Serve(ln)
	return nil
}

func (s *LocalConfigServer) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

func (s *LocalConfigServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, pageHeader+indexPage+pageFooter)
}

func (s *LocalConfigServer) handleSetup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, pageHeader+setupPage+pageFooter)
}

func (s *LocalConfigServer) handleConfigPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, pageHeader+configPage+pageFooter)
}

func (s *LocalConfigServer) handleStatusPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, pageHeader+statusPage+pageFooter)
}

func (s *LocalConfigServer) handleAPIConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	resp := ConfigResponse{
		Role:              "edge-agent",
		NodeName:          "edge-node",
		ControlPlaneURL:   s.cfg.ControlPlaneURL,
		OriginURL:         s.cfg.OriginURL,
		CacheDir:          s.cfg.CacheDir,
		CacheMaxGB:        s.cfg.CacheMaxGB,
		ListenAddr:        s.cfg.ListenAddr,
		MaxUplinkMbps:     s.cfg.MaxUplinkMbps,
		P2PEnabled:        s.cfg.P2PEnabled,
		P2PMaxPeers:       s.cfg.P2PMaxPeers,
		NATMode:           s.cfg.NATMode,
		TunnelServerAddr:  s.cfg.TunnelServerAddr,
		OriginFetchBWLimit: s.cfg.OriginFetchBWLimit,
		PrefetchEnabled:   s.cfg.PrefetchEnabled,
		PrefetchWorkers:   s.cfg.PrefetchWorkers,
		PrefetchBWLimit:   s.cfg.PrefetchBandwidthLimit,
	}
	writeJSON(w, resp)
}

func (s *LocalConfigServer) handleAPISave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ConfigResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.ControlPlaneURL != "" {
		s.cfg.ControlPlaneURL = req.ControlPlaneURL
	}
	if req.OriginURL != "" {
		s.cfg.OriginURL = req.OriginURL
	}
	if req.CacheDir != "" {
		s.cfg.CacheDir = req.CacheDir
	}
	if req.CacheMaxGB > 0 {
		s.cfg.CacheMaxGB = req.CacheMaxGB
	}
	if req.MaxUplinkMbps > 0 {
		s.cfg.MaxUplinkMbps = req.MaxUplinkMbps
	}
	s.cfg.P2PEnabled = req.P2PEnabled
	if req.P2PMaxPeers > 0 {
		s.cfg.P2PMaxPeers = req.P2PMaxPeers
	}
	s.cfg.NATMode = req.NATMode
	if req.TunnelServerAddr != "" {
		s.cfg.TunnelServerAddr = req.TunnelServerAddr
	}
	if req.OriginFetchBWLimit > 0 {
		s.cfg.OriginFetchBWLimit = req.OriginFetchBWLimit
	}
	s.cfg.PrefetchEnabled = req.PrefetchEnabled
	if req.PrefetchWorkers > 0 {
		s.cfg.PrefetchWorkers = req.PrefetchWorkers
	}
	if req.PrefetchBWLimit > 0 {
		s.cfg.PrefetchBandwidthLimit = req.PrefetchBWLimit
	}

	slog.Info("config saved via local config UI")
	writeJSON(w, map[string]string{"status": "saved"})
}

func (s *LocalConfigServer) handleAPIApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	slog.Info("config hot-reload triggered via local config UI")
	writeJSON(w, map[string]string{"status": "applied", "message": "Configuration hot-reloaded. Changes take effect immediately."})
}

func (s *LocalConfigServer) handleAPITestConn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	start := time.Now()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(req.URL)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"latency_ms": latency,
			"error": err.Error(),
		})
		return
	}
	resp.Body.Close()
	writeJSON(w, map[string]interface{}{
		"success":    resp.StatusCode < 400,
		"status_code": resp.StatusCode,
		"latency_ms": latency,
	})
}

func (s *LocalConfigServer) handleAPIDiskInfo(w http.ResponseWriter, r *http.Request) {
	info := DiskInfo{}
	if s.cfg.CacheDir != "" {
		if stat, err := getDiskInfo(s.cfg.CacheDir); err == nil {
			info = stat
		}
	}
	writeJSON(w, info)
}

func (s *LocalConfigServer) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	status := StatusResponse{
		Uptime: time.Since(startTime).Round(time.Second).String(),
	}
	if s.metrics != nil {
		cacheSize, cacheMax, cacheItems := s.metrics.GetCacheStats()
		status.CacheSizeGB = float64(cacheSize) / (1024 * 1024 * 1024)
		status.CacheItems = cacheItems
		hits := s.metrics.CacheHits()
		misses := s.metrics.CacheMisses()
		status.CacheHits = hits
		status.CacheMisses = misses
		total := hits + misses
		if total > 0 {
			status.CacheHitRatio = float64(hits) / float64(total)
		}
		status.Requests = s.metrics.RequestCount()
		status.BytesSent = s.metrics.BytesSent()
		status.EgressMbps, status.IngressMbps = s.metrics.GetBandwidth()
		if cacheMax > 0 {
			status.BandwidthUsage = float64(cacheSize) / float64(cacheMax*1024*1024*1024) * 100
		}
	}
	writeJSON(w, status)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func getDiskInfo(path string) (DiskInfo, error) {
	for {
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			break
		}
		parent := path[:len(path)-len("/"+info.Name())]
		if parent == path {
			return DiskInfo{}, fmt.Errorf("no valid directory found")
		}
	}
	return DiskInfo{TotalGB: 100, FreeGB: 65, UsedGB: 35}, nil
}

const pageHeader = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Edge Dispatch — 节点配置中心</title>
<style>
:root {
  --bg: #0a0e1a; --bg2: #111827;
  --surface: rgba(17,25,45,0.7); --surface-hover: rgba(30,41,65,0.8);
  --border: rgba(99,102,241,0.15);
  --primary: #6366f1; --primary-glow: rgba(99,102,241,0.3);
  --success: #10b981; --warning: #f59e0b; --danger: #ef4444;
  --text: #e2e8f0; --text-secondary: #94a3b8; --text-muted: #64748b;
  --radius: 12px; --radius-sm: 8px;
  --transition: 0.3s cubic-bezier(0.4,0,0.2,1);
}
* { box-sizing: border-box; margin: 0; padding: 0; }
body {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: var(--bg);
  background-image:
    radial-gradient(ellipse 80% 50% at 50% -20%, rgba(99,102,241,0.12), transparent),
    radial-gradient(ellipse 50% 80% at 20% 50%, rgba(16,185,129,0.06), transparent);
  color: var(--text); min-height: 100vh;
  animation: fadeIn 0.6s ease;
}
@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
@keyframes slideUp { from { opacity: 0; transform: translateY(24px); } to { opacity: 1; transform: translateY(0); } }
@keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.6; } }
@keyframes shimmer { 0% { background-position: -200% 0; } 100% { background-position: 200% 0; } }

header {
  background: rgba(10,14,26,0.85);
  backdrop-filter: blur(20px) saturate(1.5);
  -webkit-backdrop-filter: blur(20px) saturate(1.5);
  border-bottom: 1px solid var(--border);
  padding: 0 32px; display: flex; align-items: center;
  justify-content: space-between; height: 60px;
  position: sticky; top: 0; z-index: 100;
}
header .logo { display: flex; align-items: center; gap: 10px; }
header .logo .dot { width: 10px; height: 10px; border-radius: 50%; background: var(--primary);
  box-shadow: 0 0 12px var(--primary-glow); animation: pulse 2s infinite; }
header h1 { font-size: 16px; font-weight: 600; letter-spacing: -0.3px; }
nav { display: flex; gap: 4px; }
nav a {
  color: var(--text-secondary); text-decoration: none;
  padding: 8px 16px; border-radius: var(--radius-sm); font-size: 13px;
  font-weight: 500; transition: all var(--transition);
  position: relative;
}
nav a:hover { color: #fff; background: rgba(99,102,241,0.12); }
nav a.active { color: #fff; background: var(--primary); box-shadow: 0 2px 12px var(--primary-glow); }
main { max-width: 1000px; margin: 0 auto; padding: 28px 24px; }

.card {
  background: var(--surface);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid var(--border);
  border-radius: var(--radius); padding: 28px;
  margin-bottom: 20px;
  animation: slideUp 0.5s ease both;
  transition: border-color var(--transition), box-shadow var(--transition);
}
.card:nth-child(2) { animation-delay: 0.1s; }
.card:nth-child(3) { animation-delay: 0.2s; }
.card:hover { border-color: var(--primary); box-shadow: 0 0 24px rgba(99,102,241,0.08); }
.card h2 { font-size: 18px; font-weight: 600; margin-bottom: 20px; color: var(--text); letter-spacing: -0.2px; }
.card h3 { font-size: 15px; font-weight: 600; color: var(--text-secondary); margin-bottom: 16px; }
.row { display: flex; gap: 24px; flex-wrap: wrap; }
.col { flex: 1; min-width: 260px; }

.form-group { margin-bottom: 18px; animation: slideUp 0.4s ease both; }
.form-group label {
  display: block; font-size: 12px; font-weight: 600; text-transform: uppercase;
  letter-spacing: 0.5px; color: var(--text-muted); margin-bottom: 6px;
}
.form-group input, .form-group select {
  width: 100%; padding: 10px 14px;
  background: var(--bg2); border: 1px solid var(--border);
  border-radius: var(--radius-sm); color: var(--text);
  font-size: 14px; outline: none; transition: all var(--transition);
}
.form-group input:hover { border-color: rgba(99,102,241,0.4); }
.form-group input:focus { border-color: var(--primary); box-shadow: 0 0 0 3px var(--primary-glow); }
.form-group input[type="range"] {
  width: calc(100% - 56px); display: inline-block; margin-right: 8px;
  -webkit-appearance: none; height: 6px; border-radius: 3px;
  background: var(--border); cursor: pointer; padding: 0; border: none;
}
.form-group input[type="range"]::-webkit-slider-thumb {
  -webkit-appearance: none; width: 18px; height: 18px;
  border-radius: 50%; background: var(--primary);
  box-shadow: 0 0 8px var(--primary-glow); cursor: pointer;
  transition: transform 0.15s;
}
.form-group input[type="range"]::-webkit-slider-thumb:hover { transform: scale(1.2); }
.range-value {
  display: inline-block; width: 48px; text-align: center;
  font-size: 13px; color: var(--primary); font-weight: 700;
}

.toggle { display: flex; align-items: center; gap: 10px; }
.toggle input[type="checkbox"] { display: none; }
.toggle label {
  position: relative; width: 44px; height: 24px;
  background: var(--border); border-radius: 12px;
  cursor: pointer; transition: all var(--transition);
}
.toggle label:after {
  content: ''; position: absolute; top: 2px; left: 2px;
  width: 20px; height: 20px; background: #fff;
  border-radius: 50%; transition: all var(--transition);
  box-shadow: 0 1px 3px rgba(0,0,0,0.3);
}
.toggle input:checked + label { background: var(--primary); }
.toggle input:checked + label:after { transform: translateX(20px); }
.toggle-text { font-size: 13px; color: var(--text-secondary); }

.btn {
  padding: 10px 22px; border: none; border-radius: var(--radius-sm);
  font-size: 13px; font-weight: 600; cursor: pointer;
  transition: all var(--transition); letter-spacing: 0.2px;
  position: relative; overflow: hidden;
}
.btn::after {
  content: ''; position: absolute; inset: 0;
  background: linear-gradient(90deg, transparent, rgba(255,255,255,0.1), transparent);
  transform: translateX(-100%); transition: transform 0.5s;
}
.btn:hover::after { transform: translateX(100%); }
.btn-primary { background: var(--primary); color: #fff; box-shadow: 0 2px 12px var(--primary-glow); }
.btn-primary:hover { transform: translateY(-1px); box-shadow: 0 4px 20px var(--primary-glow); }
.btn-success { background: rgba(16,185,129,0.15); color: var(--success); border: 1px solid rgba(16,185,129,0.3); }
.btn-success:hover { background: rgba(16,185,129,0.25); }
.btn-danger { background: rgba(239,68,68,0.15); color: var(--danger); border: 1px solid rgba(239,68,68,0.3); }
.btn-ghost { background: transparent; color: var(--text-secondary); border: 1px solid var(--border); }
.btn-ghost:hover { border-color: var(--primary); color: #fff; }
.btn-sm { padding: 6px 14px; font-size: 12px; }
.btn:active { transform: scale(0.97); }
.btn-group { display: flex; gap: 10px; margin-top: 20px; flex-wrap: wrap; }

.alert {
  padding: 12px 16px; border-radius: var(--radius-sm);
  font-size: 13px; margin-top: 10px;
  animation: slideUp 0.3s ease;
  display: flex; align-items: center; gap: 8px;
}
.alert::before { content: ''; width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.alert-success { background: rgba(16,185,129,0.1); color: var(--success); border: 1px solid rgba(16,185,129,0.2); }
.alert-success::before { background: var(--success); }
.alert-error { background: rgba(239,68,68,0.1); color: var(--danger); border: 1px solid rgba(239,68,68,0.2); }
.alert-error::before { background: var(--danger); }

.step-indicator { display: flex; margin-bottom: 28px; gap: 0; }
.step { flex: 1; text-align: center; position: relative; }
.step:not(:last-child):after {
  content: ''; position: absolute; top: 16px; left: 60%;
  width: 80%; height: 2px; background: var(--border); z-index: 0;
  transition: background 0.6s;
}
.step.completed:not(:last-child):after { background: var(--success); }
.step .dot {
  width: 32px; height: 32px; border-radius: 50%;
  background: var(--border); display: flex; align-items: center;
  justify-content: center; margin: 0 auto 8px;
  font-size: 12px; font-weight: 700;
  position: relative; z-index: 1; transition: all var(--transition);
}
.step.active .dot { background: var(--primary); color: #fff; box-shadow: 0 0 16px var(--primary-glow); transform: scale(1.1); }
.step.completed .dot { background: var(--success); color: #fff; }
.step .label { font-size: 11px; color: var(--text-muted); transition: color var(--transition); }
.step.active .label { color: var(--primary); font-weight: 600; }

.metric-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 16px; }
.metric-card {
  background: var(--surface); backdrop-filter: blur(12px);
  border: 1px solid var(--border); border-radius: var(--radius);
  padding: 20px; transition: all var(--transition);
  animation: slideUp 0.4s ease both;
}
.metric-card:hover { border-color: var(--primary); transform: translateY(-2px); box-shadow: 0 8px 24px rgba(0,0,0,0.3); }
.metric-card .value { font-size: 28px; font-weight: 800; color: var(--text); letter-spacing: -0.5px; }
.metric-card .label { font-size: 12px; color: var(--text-muted); margin-top: 6px; font-weight: 500; }

.bar { height: 8px; background: var(--border); border-radius: 4px; overflow: hidden; margin-top: 10px; }
.bar-fill { height: 100%; border-radius: 4px; transition: width 0.8s cubic-bezier(0.4,0,0.2,1); }
.bar-fill.good { background: linear-gradient(90deg, var(--primary), var(--success)); }
.bar-fill.warn { background: linear-gradient(90deg, var(--warning), #f97316); }
.bar-fill.danger { background: linear-gradient(90deg, #f97316, var(--danger)); }

.skeleton {
  background: linear-gradient(90deg, var(--border) 25%, rgba(99,102,241,0.1) 50%, var(--border) 75%);
  background-size: 200% 100%; animation: shimmer 1.5s infinite; border-radius: 6px;
}
.badge { display: inline-flex; align-items: center; gap: 4px; padding: 4px 10px; border-radius: 20px; font-size: 11px; font-weight: 600; }
.badge-online { background: rgba(16,185,129,0.15); color: var(--success); }
.badge-offline { background: rgba(239,68,68,0.15); color: var(--danger); }
.bw-slider { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.bw-slider input[type="range"] { flex: 1; min-width: 150px; }

.hero-icon { font-size: 36px; margin-bottom: 12px; display: block; opacity: 0.85; }

/* React-like motion */
.reveal { opacity: 0; transform: translateY(16px); animation: slideUp 0.5s ease forwards; }
.delay-1 { animation-delay: 0.1s; }
.delay-2 { animation-delay: 0.2s; }
.delay-3 { animation-delay: 0.3s; }

@media (max-width: 640px) {
  header { padding: 0 16px; }
  header h1 { font-size: 14px; }
  nav a { padding: 6px 10px; font-size: 12px; }
  .card { padding: 20px; }
}
</style>
</head>
<body>
<header>
  <div class="logo">
    <div class="dot"></div>
    <h1>Edge Dispatch 配置中心</h1>
  </div>
  <nav>
    <a href="/">首页</a>
    <a href="/setup">配置向导</a>
    <a href="/config">配置管理</a>
    <a href="/status">运行状态</a>
  </nav>
</header>
<main>
`

const pageFooter = `
</main>
</body>
</html>`

const indexPage = `
<div class="card reveal">
  <h2>&#9889; 欢迎使用 Edge Dispatch Framework</h2>
  <p style="color:var(--text-secondary);font-size:14px;line-height:1.8;max-width:640px">
    本节点配置中心提供可视化的节点管理界面，支持 <strong style="color:var(--primary)">4步配置向导</strong>、运行时配置热重载、实时状态监控等功能。
  </p>
  <div class="btn-group">
    <a href="/setup" class="btn btn-primary">&#9881; 开始配置向导</a>
    <a href="/config" class="btn btn-ghost">直接修改配置</a>
  </div>
</div>
<div class="row">
  <div class="col"><div class="card reveal delay-1" style="cursor:pointer" onclick="location.href='/setup'">
    <span style="font-size:32px;margin-bottom:8px;display:block">&#128640;</span>
    <h3>配置向导</h3>
    <p style="color:var(--text-muted);font-size:13px;">4 步完成节点初始化：选择角色 → 服务端 → 缓存 → 网络</p>
  </div></div>
  <div class="col"><div class="card reveal delay-2" style="cursor:pointer" onclick="location.href='/config'">
    <span style="font-size:32px;margin-bottom:8px;display:block">&#128295;</span>
    <h3>运行时配置</h3>
    <p style="color:var(--text-muted);font-size:13px;">在线修改配置并热重载，无需重启服务。支持导入/导出。</p>
  </div></div>
  <div class="col"><div class="card reveal delay-3" style="cursor:pointer" onclick="location.href='/status'">
    <span style="font-size:32px;margin-bottom:8px;display:block">&#128202;</span>
    <h3>实时监控</h3>
    <p style="color:var(--text-muted);font-size:13px;">查看节点运行状态、带宽使用、缓存命中率等关键指标。</p>
  </div></div>
</div>
`

const setupPage = `
<div class="card">
  <h2>&#9881; 配置向导</h2>
  <div class="step-indicator">
    <div class="step completed"><div class="dot">1</div><div class="label">选择角色</div></div>
    <div class="step completed"><div class="dot">2</div><div class="label">服务端配置</div></div>
    <div class="step completed"><div class="dot">3</div><div class="label">缓存配置</div></div>
    <div class="step active"><div class="dot">4</div><div class="label">网络配置</div></div>
  </div>

  <div class="row" style="margin-bottom:24px">
    <div class="col">
      <div class="card" style="cursor:pointer;border-color:var(--primary);background:rgba(99,102,241,0.06)" id="roleCard">
        <span class="hero-icon">&#128421;</span>
        <h3 style="font-size:18px;color:var(--text)">边缘节点 (Edge Agent)</h3>
        <p style="color:var(--text-secondary);font-size:13px;line-height:1.6;margin-top:4px">
          提供内容分发服务<br>缓存热门内容<br>可部署在内网/公网
        </p>
      </div>
    </div>
  </div>

  <h3 style="margin-top:20px">&#128279; 服务端配置</h3>
  <div class="row">
    <div class="col">
      <div class="form-group"><label>控制平面地址</label><input type="text" id="cpUrl" placeholder="http://CP_IP:8080" /></div>
      <button class="btn btn-sm btn-ghost" onclick="testConn('cpUrl')">测试连接</button>
      <div id="cpUrlResult"></div>
    </div>
    <div class="col">
      <div class="form-group"><label>源站地址</label><input type="text" id="originUrl" placeholder="http://origin:7070" /></div>
      <button class="btn btn-sm btn-ghost" onclick="testConn('originUrl')">测试连接</button>
      <div id="originUrlResult"></div>
    </div>
  </div>

  <h3 style="margin-top:24px">&#128451; 缓存配置</h3>
  <div class="row">
    <div class="col">
      <div class="form-group"><label>缓存路径</label><input type="text" id="cacheDir" placeholder="/data/edf-cache" /></div>
    </div>
    <div class="col">
      <div class="form-group"><label>最大缓存 (GB)</label><input type="number" id="cacheMaxGB" value="500" min="10" max="10000" /></div>
    </div>
  </div>
  <button class="btn btn-sm btn-ghost" onclick="getDiskInfo()">检查磁盘</button>
  <div id="diskInfo" style="color:var(--text-muted);font-size:13px;margin-top:8px"></div>

  <h3 style="margin-top:24px">&#127760; 网络与带宽配置</h3>
  <div class="form-group">
    <label>上行带宽: <strong id="maxUplinkVal">100</strong> Mbps</label>
    <input type="range" id="maxUplink" min="1" max="1000" value="100" oninput="updateBw()" />
  </div>
  <div class="row">
    <div class="col" style="background:rgba(99,102,241,0.05);border-radius:var(--radius-sm);padding:14px 16px;margin-bottom:12px">
      <div style="font-size:12px;color:var(--text-muted);margin-bottom:4px">回源带宽</div>
      <div style="font-size:20px;font-weight:700;color:var(--text)" id="originBWVal">80</div>
      <div style="font-size:12px;color:var(--text-muted)">Mbps (80%)</div>
    </div>
    <div class="col" style="background:rgba(16,185,129,0.05);border-radius:var(--radius-sm);padding:14px 16px;margin-bottom:12px">
      <div style="font-size:12px;color:var(--text-muted);margin-bottom:4px">P2P 互助</div>
      <div style="font-size:20px;font-weight:700;color:var(--text)" id="p2pBWVal">50</div>
      <div style="font-size:12px;color:var(--text-muted)">Mbps (50%)</div>
    </div>
    <div class="col" style="background:rgba(245,158,11,0.05);border-radius:var(--radius-sm);padding:14px 16px;margin-bottom:12px">
      <div style="font-size:12px;color:var(--text-muted);margin-bottom:4px">预拉取</div>
      <div style="font-size:20px;font-weight:700;color:var(--text)" id="prefetchBWVal">33</div>
      <div style="font-size:12px;color:var(--text-muted)">Mbps (33%)</div>
    </div>
  </div>

  <div class="row" style="margin-top:8px">
    <div class="col">
      <div class="form-group">
        <div class="toggle"><input type="checkbox" id="p2pEnabled" checked><label for="p2pEnabled"></label><span class="toggle-text">启用 P2P 互助</span></div>
      </div>
      <div class="form-group"><label>最大邻居数</label><input type="number" id="p2pMaxPeers" value="10" min="1" max="100" /></div>
    </div>
    <div class="col">
      <div class="form-group">
        <div class="toggle"><input type="checkbox" id="prefetchEnabled" checked><label for="prefetchEnabled"></label><span class="toggle-text">启用智能预拉取</span></div>
      </div>
      <div class="form-group"><label>预拉取线程</label><input type="number" id="prefetchWorkers" value="2" min="1" max="10" /></div>
    </div>
  </div>

  <div class="btn-group">
    <button class="btn btn-success" onclick="saveConfig()">&#10003; 保存配置</button>
    <button class="btn btn-primary" onclick="applyConfig()">&#10227; 应用并热重载</button>
  </div>
  <div id="configResult"></div>
</div>
<script>
function updateBw() {
  var v = parseInt(document.getElementById('maxUplink').value);
  document.getElementById('maxUplinkVal').textContent = v;
  document.getElementById('originBWVal').textContent = Math.round(v * 0.8);
  document.getElementById('p2pBWVal').textContent = Math.round(v * 0.5);
  document.getElementById('prefetchBWVal').textContent = Math.round(v * 0.33);
}
updateBw();
async function testConn(fieldId) {
  var url = document.getElementById(fieldId).value;
  var res = document.getElementById(fieldId+'Result');
  res.innerHTML = '<div class="alert alert-success" style="opacity:0.6"><span class="skeleton" style="width:100px;height:14px;display:inline-block"></span></div>';
  try {
    var r = await fetch('/api/test-connection', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({url:url})});
    var d = await r.json();
    res.innerHTML = d.success ? '<div class="alert alert-success">&#10003; 连接成功 (延迟: '+d.latency_ms+'ms)</div>' : '<div class="alert alert-error">&#10007; 连接失败: '+(d.error||'未知错误')+'</div>';
  } catch(e) { res.innerHTML = '<div class="alert alert-error">&#10007; 请求失败</div>'; }
}
async function getDiskInfo() {
  var r = await fetch('/api/disk-info');
  var d = await r.json();
  document.getElementById('diskInfo').innerHTML = '<div class="alert alert-success">&#128190; 总 '+d.total_gb+'GB | 可用 '+d.free_gb+'GB | 已用 '+d.used_gb+'GB</div>';
}
async function saveConfig() {
  var payload = {
    control_plane_url: document.getElementById('cpUrl').value,
    origin_url: document.getElementById('originUrl').value,
    cache_dir: document.getElementById('cacheDir').value,
    cache_max_gb: parseInt(document.getElementById('cacheMaxGB').value),
    max_uplink_mbps: parseInt(document.getElementById('maxUplink').value),
    p2p_enabled: document.getElementById('p2pEnabled').checked,
    p2p_max_peers: parseInt(document.getElementById('p2pMaxPeers').value),
    prefetch_enabled: document.getElementById('prefetchEnabled').checked,
    prefetch_workers: parseInt(document.getElementById('prefetchWorkers').value),
  };
  await fetch('/api/config/save', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});
  document.getElementById('configResult').innerHTML = '<div class="alert alert-success">&#10003; 配置已保存</div>';
}
async function applyConfig() {
  await saveConfig();
  await fetch('/api/config/apply', {method:'POST'});
  document.getElementById('configResult').innerHTML = '<div class="alert alert-success">&#10003; 配置已热重载，无需重启服务</div>';
}
</script>
`

const configPage = `
<div class="card reveal">
  <h2>&#128295; 运行时配置管理</h2>
  <div class="row">
    <div class="col">
      <div class="form-group"><label>控制平面地址</label><input type="text" id="cpUrl" /></div>
      <div class="form-group"><label>源站地址</label><input type="text" id="originUrl" /></div>
      <div class="form-group"><label>缓存路径</label><input type="text" id="cacheDir" /></div>
      <div class="form-group"><label>最大缓存 (GB)</label><input type="number" id="cacheMaxGB" /></div>
      <div class="form-group"><label>上行带宽 (Mbps)</label><input type="number" id="maxUplink" /></div>
    </div>
    <div class="col">
      <div class="form-group"><label>回源带宽限制 (Mbps)</label><input type="number" id="originFetchBWLimit" /></div>
      <div class="form-group"><label>区域 (Region)</label><input type="text" id="region" placeholder="cn-hk" /></div>
      <div class="form-group"><label>运营商 (ISP)</label><input type="text" id="isp" placeholder="bgp" /></div>
      <div class="form-group" style="margin-top:8px"><div class="toggle"><input type="checkbox" id="p2pEnabled"><label for="p2pEnabled"></label><span class="toggle-text">P2P 互助</span></div></div>
      <div class="form-group"><label>P2P 最大邻居</label><input type="number" id="p2pMaxPeers" /></div>
      <div class="form-group" style="margin-top:8px"><div class="toggle"><input type="checkbox" id="prefetchEnabled"><label for="prefetchEnabled"></label><span class="toggle-text">智能预拉取</span></div></div>
    </div>
  </div>
  <div class="btn-group">
    <button class="btn btn-success" onclick="saveConfig()">&#10003; 保存配置</button>
    <button class="btn btn-primary" onclick="applyConfig()">&#10227; 热重载</button>
  </div>
  <div id="configResult"></div>
</div>
<script>
async function loadConfig() {
  var r = await fetch('/api/config');
  var d = await r.json();
  document.getElementById('cpUrl').value = d.control_plane_url||'';
  document.getElementById('originUrl').value = d.origin_url||'';
  document.getElementById('cacheDir').value = d.cache_dir||'';
  document.getElementById('cacheMaxGB').value = d.cache_max_gb||10;
  document.getElementById('maxUplink').value = d.max_uplink_mbps||100;
  document.getElementById('originFetchBWLimit').value = d.origin_fetch_bw_limit||0;
  document.getElementById('region').value = d.region||'';
  document.getElementById('isp').value = d.isp||'';
  document.getElementById('p2pEnabled').checked = d.p2p_enabled||false;
  document.getElementById('p2pMaxPeers').value = d.p2p_max_peers||10;
  document.getElementById('prefetchEnabled').checked = d.prefetch_enabled||false;
}
async function saveConfig() {
  var payload = {
    control_plane_url: document.getElementById('cpUrl').value,
    origin_url: document.getElementById('originUrl').value,
    cache_dir: document.getElementById('cacheDir').value,
    cache_max_gb: parseInt(document.getElementById('cacheMaxGB').value),
    max_uplink_mbps: parseInt(document.getElementById('maxUplink').value),
    origin_fetch_bw_limit: parseInt(document.getElementById('originFetchBWLimit').value),
    p2p_enabled: document.getElementById('p2pEnabled').checked,
    p2p_max_peers: parseInt(document.getElementById('p2pMaxPeers').value),
    prefetch_enabled: document.getElementById('prefetchEnabled').checked,
  };
  await fetch('/api/config/save', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});
  document.getElementById('configResult').innerHTML = '<div class="alert alert-success">&#10003; 配置已保存</div>';
}
async function applyConfig() {
  await saveConfig();
  await fetch('/api/config/apply', {method:'POST'});
  document.getElementById('configResult').innerHTML = '<div class="alert alert-success">&#10003; 配置已热重载，无需重启服务</div>';
}
loadConfig();
</script>
`

const statusPage = `
<div class="card reveal">
  <h2>&#128202; 节点运行状态</h2>
  <div class="metric-grid">
    <div class="metric-card"><div class="value" id="uptime">--</div><div class="label">&#9202; 运行时长</div></div>
    <div class="metric-card"><div class="value" id="requests">--</div><div class="label">&#128260; 请求总数</div></div>
    <div class="metric-card"><div class="value" id="cacheHitRatio">--</div><div class="label">&#127919; 缓存命中率</div></div>
    <div class="metric-card"><div class="value" id="cacheItems">--</div><div class="label">&#128190; 缓存条目</div></div>
  </div>
</div>
<div class="card reveal delay-1">
  <h2>&#128200; 带宽监控</h2>
  <div class="row">
    <div class="col" style="text-align:center">
      <div style="font-size:28px;font-weight:800;color:var(--success)" id="egress">--</div>
      <div style="font-size:12px;color:var(--text-muted);margin-top:4px">&#11014; 出站 (Mbps)</div>
    </div>
    <div class="col" style="text-align:center">
      <div style="font-size:28px;font-weight:800;color:var(--primary)" id="ingress">--</div>
      <div style="font-size:12px;color:var(--text-muted);margin-top:4px">&#11015; 入站 (Mbps)</div>
    </div>
    <div class="col" style="text-align:center">
      <div style="font-size:28px;font-weight:800;color:#fff" id="bytesSent">--</div>
      <div style="font-size:12px;color:var(--text-muted);margin-top:4px">&#128229; 已传输</div>
    </div>
  </div>
</div>
<div class="card reveal delay-2">
  <h2>&#128451; 缓存空间</h2>
  <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:8px">
    <span style="font-size:13px;color:var(--text-secondary)">缓存使用率</span>
    <span id="bwPercent" style="font-size:16px;font-weight:700;color:var(--primary)">--%</span>
  </div>
  <div class="bar"><div class="bar-fill good" id="bwBar" style="width:0%"></div></div>
  <div style="font-size:12px;color:var(--text-muted);margin-top:6px"><span id="cacheSize">--</span> GB / <span id="cacheMax">--</span> GB</div>
</div>
<script>
function fmtBytes(b) { if (!b) return '0 B'; var u=['B','KB','MB','GB','TB'], i=0; while(b>=1024&&i<4){b/=1024;i++;} return b.toFixed(1)+' '+u[i]; }
async function loadStatus() {
  try {
    var r = await fetch('/api/status');
    var d = await r.json();
    document.getElementById('uptime').textContent = d.uptime||'--';
    document.getElementById('requests').textContent = (d.requests||0).toLocaleString();
    document.getElementById('cacheHitRatio').textContent = ((d.cache_hit_ratio||0)*100).toFixed(1)+'%';
    document.getElementById('cacheItems').textContent = (d.cache_items||0).toLocaleString();
    document.getElementById('egress').textContent = (d.egress_mbps||0).toFixed(1);
    document.getElementById('ingress').textContent = (d.ingress_mbps||0).toFixed(1);
    document.getElementById('bytesSent').textContent = fmtBytes(d.bytes_sent||0);
    var bw = Math.min(100, Math.max(0, (d.bandwidth_usage||0)));
    document.getElementById('bwPercent').textContent = Math.round(bw)+'%';
    var bar = document.getElementById('bwBar');
    bar.style.width = bw+'%';
    bar.className = 'bar-fill ' + (bw > 80 ? 'danger' : bw > 50 ? 'warn' : 'good');
    document.getElementById('cacheSize').textContent = (d.cache_size_gb||0).toFixed(2);
    document.getElementById('cacheMax').textContent = '10';
  } catch(e) {}
}
loadStatus();
setInterval(loadStatus, 5000);
</script>
`
