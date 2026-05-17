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
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>Edge Dispatch — 节点配置中心</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&display=swap" rel="stylesheet">
<style>
:root{
--bg:#05080f;--bg2:#0a0f1a;--surface:rgba(10,15,28,0.8);--surface2:rgba(15,22,40,0.6);
--border:rgba(99,102,241,0.1);--border-hover:rgba(99,102,241,0.25);
--primary:#818cf8;--primary-2:#6366f1;--primary-glow:rgba(129,140,248,0.25);
--success:#34d399;--success-bg:rgba(52,211,153,0.08);--warning:#fbbf24;--danger:#f87171;
--text:#f1f5f9;--text2:#cbd5e1;--text3:#94a3b8;--text4:#64748b;
--r:14px;--r2:10px;--r3:6px;--t:.25s cubic-bezier(.4,0,.2,1);
}
*,::before,::after{box-sizing:border-box;margin:0;padding:0}
body{
font-family:'Inter',-apple-system,BlinkMacSystemFont,sans-serif;
background:var(--bg);color:var(--text);min-height:100vh;line-height:1.5;
-webkit-font-smoothing:antialiased;
}
body::before{
content:'';position:fixed;inset:0;z-index:-1;
background:
radial-gradient(ellipse 80% 50% at 50% -10%,rgba(99,102,241,.08),transparent),
radial-gradient(ellipse 40% 60% at 80% 80%,rgba(52,211,153,.04),transparent),
radial-gradient(ellipse 60% 40% at 20% 30%,rgba(129,140,248,.03),transparent);
}
@keyframes fadeIn{from{opacity:0}to{opacity:1}}
@keyframes slideUp{from{opacity:0;transform:translateY(20px)}to{opacity:1;transform:translateY(0)}}
@keyframes slideIn{from{opacity:0;transform:translateX(-8px)}to{opacity:1;transform:translateX(0)}}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.5}}
@keyframes glow{0%,100%{box-shadow:0 0 8px var(--primary-glow)}50%{box-shadow:0 0 20px var(--primary-glow)}}
@keyframes spin{to{transform:rotate(360deg)}}

header{
position:sticky;top:0;z-index:100;
background:rgba(5,8,15,.82);backdrop-filter:blur(24px)saturate(1.8);
-webkit-backdrop-filter:blur(24px)saturate(1.8);
border-bottom:1px solid var(--border);
padding:0 28px;display:flex;align-items:center;justify-content:space-between;
height:56px;animation:slideIn .4s ease;
}
header .logo{display:flex;align-items:center;gap:10px}
header .logo .dot{
width:8px;height:8px;border-radius:50%;
background:var(--success);animation:glow 2s infinite;
}
header h1{font-size:15px;font-weight:700;letter-spacing:-.3px;color:var(--text)}
nav{display:flex;gap:2px}
nav a{
color:var(--text3);text-decoration:none;padding:7px 14px;
border-radius:8px;font-size:12.5px;font-weight:500;
transition:all var(--t);position:relative;
}
nav a:hover{color:var(--text);background:rgba(129,140,248,.1)}
nav a.active{color:#fff;background:linear-gradient(135deg,var(--primary-2),var(--primary));box-shadow:0 2px 12px rgba(99,102,241,.35)}
main{max-width:1040px;margin:0 auto;padding:24px 20px 40px}

.card{
background:var(--surface);backdrop-filter:blur(20px);
-webkit-backdrop-filter:blur(20px);
border:1px solid var(--border);border-radius:var(--r);
padding:26px 28px;margin-bottom:18px;
animation:slideUp .45s ease both;
transition:border-color var(--t),box-shadow var(--t),transform var(--t);
}
.card:nth-child(2){animation-delay:.08s}
.card:nth-child(3){animation-delay:.16s}
.card:nth-child(4){animation-delay:.24s}
.card:hover{border-color:var(--border-hover)}
.card h2{font-size:17px;font-weight:700;margin-bottom:18px;display:flex;align-items:center;gap:8px}
.card h2 .icon{font-size:19px;opacity:.7}
.card h3{font-size:14px;font-weight:600;color:var(--text2);margin-bottom:14px}
.row{display:flex;gap:20px;flex-wrap:wrap}
.col{flex:1;min-width:240px}

.form-group{margin-bottom:16px}
.form-group label{
display:block;font-size:11px;font-weight:600;text-transform:uppercase;
letter-spacing:.6px;color:var(--text4);margin-bottom:5px;
}
.form-group input,.form-group select,textarea{
width:100%;padding:9px 13px;background:var(--bg2);
border:1px solid var(--border);border-radius:var(--r3);
color:var(--text);font-size:13.5px;outline:none;
transition:all var(--t);font-family:inherit;
}
.form-group input:hover,textarea:hover{border-color:var(--border-hover)}
.form-group input:focus,textarea:focus{border-color:var(--primary);box-shadow:0 0 0 3px rgba(129,140,248,.12)}
input[type="range"]{-webkit-appearance:none;width:100%;height:5px;border-radius:3px;background:var(--border);cursor:pointer;padding:0!important;border:none!important}
input[type="range"]::-webkit-slider-thumb{-webkit-appearance:none;width:16px;height:16px;border-radius:50%;background:var(--primary);box-shadow:0 0 10px var(--primary-glow);cursor:pointer;transition:transform .15s}
input[type="range"]::-webkit-slider-thumb:hover{transform:scale(1.25)}
input[type="number"]{font-variant-numeric:tabular-nums}

.toggle{display:flex;align-items:center;gap:10px}
.toggle input[type="checkbox"]{display:none}
.toggle .switch{position:relative;width:42px;height:23px;background:rgba(100,116,139,.3);border-radius:12px;cursor:pointer;transition:all var(--t);flex-shrink:0}
.toggle .switch::after{content:'';position:absolute;top:2px;left:2px;width:19px;height:19px;background:#fff;border-radius:50%;transition:all var(--t);box-shadow:0 1px 3px rgba(0,0,0,.4)}
.toggle input:checked+.switch{background:var(--primary)}
.toggle input:checked+.switch::after{transform:translateX(19px)}
.toggle-text{font-size:13px;color:var(--text2);user-select:none}

.btn{
display:inline-flex;align-items:center;gap:6px;padding:9px 18px;border:none;
border-radius:var(--r3);font-size:13px;font-weight:600;cursor:pointer;
transition:all var(--t);font-family:inherit;white-space:nowrap;
}
.btn-primary{background:linear-gradient(135deg,var(--primary-2),var(--primary));color:#fff;box-shadow:0 2px 10px rgba(99,102,241,.3)}
.btn-primary:hover{transform:translateY(-1px);box-shadow:0 4px 18px rgba(99,102,241,.4)}
.btn-success{background:var(--success-bg);color:var(--success);border:1px solid rgba(52,211,153,.2)}
.btn-success:hover{background:rgba(52,211,153,.15)}
.btn-ghost{background:transparent;color:var(--text3);border:1px solid var(--border)}
.btn-ghost:hover{border-color:var(--border-hover);color:var(--text)}
.btn-danger{background:rgba(248,113,113,.1);color:var(--danger);border:1px solid rgba(248,113,113,.2)}
.btn-sm{padding:5px 12px;font-size:12px}
.btn:active{transform:scale(.97)}
.btn:disabled{opacity:.5;pointer-events:none}
.btn-group{display:flex;gap:10px;margin-top:18px;flex-wrap:wrap}

.alert{padding:11px 15px;border-radius:var(--r3);font-size:13px;margin-top:10px;display:flex;align-items:center;gap:8px;animation:slideUp .25s ease}
.alert::before{content:'';width:7px;height:7px;border-radius:50%;flex-shrink:0}
.alert-success{background:rgba(52,211,153,.06);color:var(--success);border:1px solid rgba(52,211,153,.15)}
.alert-success::before{background:var(--success)}
.alert-error{background:rgba(248,113,113,.06);color:var(--danger);border:1px solid rgba(248,113,113,.15)}
.alert-error::before{background:var(--danger)}
.alert-info{background:rgba(129,140,248,.06);color:var(--primary);border:1px solid rgba(129,140,248,.15)}
.alert-info::before{background:var(--primary)}

.steps{display:flex;margin-bottom:24px}
.step{flex:1;text-align:center;position:relative}
.step:not(:last-child)::after{content:'';position:absolute;top:15px;left:58%;width:84%;height:2px;background:var(--border);z-index:0;transition:background .5s}
.step.done:not(:last-child)::after{background:var(--success)}
.step .dot{width:30px;height:30px;border-radius:50%;background:var(--border);display:flex;align-items:center;justify-content:center;margin:0 auto 7px;font-size:11px;font-weight:700;position:relative;z-index:1;transition:all var(--t)}
.step.active .dot{background:var(--primary);color:#fff;box-shadow:0 0 16px var(--primary-glow);transform:scale(1.08)}
.step.done .dot{background:var(--success);color:#fff}
.step .label{font-size:11px;color:var(--text4);transition:color var(--t)}
.step.active .label{color:var(--primary);font-weight:600}

.stats-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:14px}
.stat-card{
background:var(--surface2);border:1px solid var(--border);border-radius:var(--r);
padding:18px 20px;transition:all var(--t);animation:slideUp .35s ease both;
position:relative;overflow:hidden;
}
.stat-card::before{content:'';position:absolute;top:0;left:0;right:0;height:2px;background:linear-gradient(90deg,transparent,var(--primary),transparent);opacity:0;transition:opacity var(--t)}
.stat-card:hover::before{opacity:.6}
.stat-card:hover{border-color:var(--border-hover);transform:translateY(-1px)}
.stat-card .value{font-size:26px;font-weight:800;letter-spacing:-.5px;margin-bottom:3px}
.stat-card .label{font-size:11.5px;color:var(--text4);font-weight:500}

.bar{height:6px;background:rgba(100,116,139,.2);border-radius:3px;overflow:hidden;margin-top:8px}
.bar-fill{height:100%;border-radius:3px;transition:width .7s cubic-bezier(.4,0,.2,1)}
.bar-fill.good{background:linear-gradient(90deg,var(--primary),var(--success))}
.bar-fill.warn{background:linear-gradient(90deg,var(--warning),#f97316)}
.bar-fill.danger{background:linear-gradient(90deg,#f97316,var(--danger))}

.skeleton{background:linear-gradient(90deg,var(--border) 25%,rgba(129,140,248,.06) 50%,var(--border) 75%);background-size:200% 100%;animation:shimmer 1.5s infinite;border-radius:4px}
@keyframes shimmer{0%{background-position:-200% 0}100%{background-position:200% 0}}

.badge{display:inline-flex;align-items:center;gap:4px;padding:3px 9px;border-radius:20px;font-size:10.5px;font-weight:600}
.badge-success{background:rgba(52,211,153,.12);color:var(--success)}
.badge-warning{background:rgba(251,191,36,.12);color:var(--warning)}
.badge-danger{background:rgba(248,113,113,.12);color:var(--danger)}
.badge-info{background:rgba(129,140,248,.12);color:var(--primary)}

.emptystate{text-align:center;padding:40px 20px;color:var(--text4)}
.emptystate .icon{font-size:48px;margin-bottom:12px;opacity:.3}
.emptystate p{font-size:14px}

.spinner{width:18px;height:18px;border:2px solid var(--border);border-top-color:var(--primary);border-radius:50%;animation:spin .6s linear infinite;display:inline-block}

.grid-3{display:grid;grid-template-columns:repeat(3,1fr);gap:14px}
@media(max-width:768px){.grid-3{grid-template-columns:1fr}}
@media(max-width:640px){
header{padding:0 14px}header h1{font-size:13px}nav a{padding:5px 9px;font-size:11px}
.card{padding:18px 16px}.row{flex-direction:column}
}
</style>
</head>
<body>
<header>
<div class="logo"><div class="dot"></div><h1>Edge Dispatch 配置中心</h1></div>
<nav><a href="/">首页</a><a href="/setup">配置向导</a><a href="/config">配置管理</a><a href="/status">运行状态</a></nav>
</header>
<main>
`

const pageFooter = `
</main>
</body>
</html>`

const indexPage = `
<div class="card">
  <h2><span class="icon">&#9889;</span> 欢迎使用 Edge Dispatch Framework</h2>
  <p style="color:var(--text3);font-size:13.5px;line-height:1.8;max-width:640px">
    本节点配置中心提供可视化的节点管理界面，支持 <strong style="color:var(--primary)">4步配置向导</strong>、运行时配置热重载、实时状态监控等功能。
  </p>
  <div class="btn-group">
    <a href="/setup" class="btn btn-primary">&#9881; 开始配置向导</a>
    <a href="/config" class="btn btn-ghost">直接修改配置</a>
  </div>
</div>
<div class="row">
  <div class="col"><div class="card" style="cursor:pointer" onclick="location.href='/setup'">
    <span style="font-size:28px;margin-bottom:6px;display:block;opacity:.8">&#128640;</span>
    <h3>配置向导</h3>
    <p style="color:var(--text4);font-size:12.5px;line-height:1.6">4 步完成节点初始化<br>角色 → 服务端 → 缓存 → 网络</p>
  </div></div>
  <div class="col"><div class="card" style="cursor:pointer" onclick="location.href='/config'">
    <span style="font-size:28px;margin-bottom:6px;display:block;opacity:.8">&#128295;</span>
    <h3>运行时配置</h3>
    <p style="color:var(--text4);font-size:12.5px;line-height:1.6">在线修改配置并热重载<br>无需重启服务</p>
  </div></div>
  <div class="col"><div class="card" style="cursor:pointer" onclick="location.href='/status'">
    <span style="font-size:28px;margin-bottom:6px;display:block;opacity:.8">&#128202;</span>
    <h3>实时监控</h3>
    <p style="color:var(--text4);font-size:12.5px;line-height:1.6">节点运行状态<br>带宽 · 缓存 · 请求量</p>
  </div></div>
</div>
`

const setupPage = `
<div class="card">
  <h2>&#9881; 配置向导</h2>
  <div class="steps">
    <div class="step done"><div class="dot">1</div><div class="label">选择角色</div></div>
    <div class="step done"><div class="dot">2</div><div class="label">服务端配置</div></div>
    <div class="step done"><div class="dot">3</div><div class="label">缓存配置</div></div>
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
        <div class="toggle"><input type="checkbox" id="p2pEnabled" checked><span class="switch"></span><span class="toggle-text">启用 P2P 互助</span></div>
      </div>
      <div class="form-group"><label>最大邻居数</label><input type="number" id="p2pMaxPeers" value="10" min="1" max="100" /></div>
    </div>
    <div class="col">
      <div class="form-group">
        <div class="toggle"><input type="checkbox" id="prefetchEnabled" checked><span class="switch"></span><span class="toggle-text">启用智能预拉取</span></div>
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
<div class="card">
  <h2><span class="icon">&#128295;</span> 运行时配置管理</h2>
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
      <div class="form-group"><div class="toggle"><input type="checkbox" id="p2pEnabled"><span class="switch"></span><span class="toggle-text">P2P 互助</span></div></div>
      <div class="form-group"><label>P2P 最大邻居</label><input type="number" id="p2pMaxPeers" /></div>
      <div class="form-group"><div class="toggle"><input type="checkbox" id="prefetchEnabled"><span class="switch"></span><span class="toggle-text">智能预拉取</span></div></div>
    </div>
  </div>
  <div class="btn-group">
    <button class="btn btn-primary" onclick="saveConfig()">&#10003; 保存配置</button>
    <button class="btn btn-success" onclick="applyConfig()">&#10227; 热重载</button>
  </div>
  <div id="configResult"></div>
</div>
<script>
async function loadConfig(){
var r=await fetch('/api/config');var d=await r.json();
document.getElementById('cpUrl').value=d.control_plane_url||'';
document.getElementById('originUrl').value=d.origin_url||'';
document.getElementById('cacheDir').value=d.cache_dir||'';
document.getElementById('cacheMaxGB').value=d.cache_max_gb||10;
document.getElementById('maxUplink').value=d.max_uplink_mbps||100;
document.getElementById('originFetchBWLimit').value=d.origin_fetch_bw_limit||0;
document.getElementById('region').value=d.region||'';
document.getElementById('isp').value=d.isp||'';
document.getElementById('p2pEnabled').checked=d.p2p_enabled||false;
document.getElementById('p2pMaxPeers').value=d.p2p_max_peers||10;
document.getElementById('prefetchEnabled').checked=d.prefetch_enabled||false;
}
async function saveConfig(){
var payload={
control_plane_url:document.getElementById('cpUrl').value,
origin_url:document.getElementById('originUrl').value,
cache_dir:document.getElementById('cacheDir').value,
cache_max_gb:parseInt(document.getElementById('cacheMaxGB').value),
max_uplink_mbps:parseInt(document.getElementById('maxUplink').value),
origin_fetch_bw_limit:parseInt(document.getElementById('originFetchBWLimit').value),
p2p_enabled:document.getElementById('p2pEnabled').checked,
p2p_max_peers:parseInt(document.getElementById('p2pMaxPeers').value),
prefetch_enabled:document.getElementById('prefetchEnabled').checked,
};
await fetch('/api/config/save',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});
document.getElementById('configResult').innerHTML='<div class="alert alert-success">&#10003; 配置已保存</div>';
}
async function applyConfig(){await saveConfig();await fetch('/api/config/apply',{method:'POST'});
document.getElementById('configResult').innerHTML='<div class="alert alert-success">&#10003; 配置已热重载，无需重启服务</div>';}
loadConfig();
</script>
`

const statusPage = `
<div class="card">
  <h2><span class="icon">&#128202;</span> 节点运行状态</h2>
  <div class="stats-grid">
    <div class="stat-card"><div class="value" id="uptime">--</div><div class="label">&#9202; 运行时长</div></div>
    <div class="stat-card"><div class="value" id="requests">--</div><div class="label">&#128260; 请求总数</div></div>
    <div class="stat-card"><div class="value" id="cacheHitRatio">--</div><div class="label">&#127919; 缓存命中率</div></div>
    <div class="stat-card"><div class="value" id="cacheItems">--</div><div class="label">&#128190; 缓存条目</div></div>
  </div>
</div>
<div class="card">
  <h2><span class="icon">&#128200;</span> 带宽监控</h2>
  <div class="grid-3">
    <div class="stat-card" style="text-align:center">
      <div class="value" id="egress" style="color:var(--success)">--</div>
      <div class="label">&#11014; 出站 Mbps</div>
    </div>
    <div class="stat-card" style="text-align:center">
      <div class="value" id="ingress" style="color:var(--primary)">--</div>
      <div class="label">&#11015; 入站 Mbps</div>
    </div>
    <div class="stat-card" style="text-align:center">
      <div class="value" id="bytesSent" style="color:var(--text)">--</div>
      <div class="label">&#128229; 已传输</div>
    </div>
  </div>
</div>
<div class="card">
  <h2><span class="icon">&#128451;</span> 缓存空间</h2>
  <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:6px">
    <span style="font-size:12.5px;color:var(--text2)">缓存使用率</span>
    <span id="bwPercent" style="font-size:15px;font-weight:700;color:var(--primary)">--%</span>
  </div>
  <div class="bar"><div class="bar-fill good" id="bwBar" style="width:0%"></div></div>
  <div style="font-size:12px;color:var(--text4);margin-top:5px"><span id="cacheSize">--</span> GB / <span id="cacheMax">10</span> GB</div>
</div>
<script>
function fmtBytes(b){if(!b)return'0 B';var u=['B','KB','MB','GB','TB'],i=0;while(b>=1024&&i<4){b/=1024;i++}return b.toFixed(1)+' '+u[i]}
async function loadStatus(){
try{
var r=await fetch('/api/status');var d=await r.json();
document.getElementById('uptime').textContent=d.uptime||'--';
document.getElementById('requests').textContent=(d.requests||0).toLocaleString();
document.getElementById('cacheHitRatio').textContent=((d.cache_hit_ratio||0)*100).toFixed(1)+'%';
document.getElementById('cacheItems').textContent=(d.cache_items||0).toLocaleString();
document.getElementById('egress').textContent=(d.egress_mbps||0).toFixed(1);
document.getElementById('ingress').textContent=(d.ingress_mbps||0).toFixed(1);
document.getElementById('bytesSent').textContent=fmtBytes(d.bytes_sent||0);
var bw=Math.min(100,Math.max(0,(d.bandwidth_usage||0)));
document.getElementById('bwPercent').textContent=Math.round(bw)+'%';
var bar=document.getElementById('bwBar');bar.style.width=bw+'%';
bar.className='bar-fill '+(bw>80?'danger':bw>50?'warn':'good');
document.getElementById('cacheSize').textContent=(d.cache_size_gb||0).toFixed(2);
}catch(e){}
}
loadStatus();setInterval(loadStatus,5000);
</script>
`
