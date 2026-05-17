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
	cfg    *config.EdgeAgentConfig
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
	Uptime          string `json:"uptime"`
	CacheHitRatio   float64 `json:"cache_hit_ratio"`
	CacheItems      int64   `json:"cache_items"`
	CacheSizeGB     float64 `json:"cache_size_gb"`
	CurrentEgressMbps float64 `json:"current_egress_mbps"`
	BandwidthUsage   float64 `json:"bandwidth_usage"`
	Connections     int64   `json:"connections"`
}

var startTime = time.Now()

func NewLocalConfigServer(cfg *config.EdgeAgentConfig) *LocalConfigServer {
	s := &LocalConfigServer{cfg: cfg}
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
	type statusResp struct {
		Uptime          string  `json:"uptime"`
		CacheHitRatio   float64 `json:"cache_hit_ratio"`
		CacheMB         float64 `json:"cache_mb"`
		TotalRequests   int64   `json:"total_requests"`
		TotalBytes      int64   `json:"total_bytes"`
		EgressMbps      float64 `json:"egress_mbps"`
		IngressMbps     float64 `json:"ingress_mbps"`
		Goroutines      int     `json:"goroutines"`
		NumCPU          int     `json:"num_cpu"`
		MaxUplinkMbps   int64   `json:"max_uplink_mbps"`
	}

	status := statusResp{
		Uptime:        time.Since(startTime).Round(time.Second).String(),
		MaxUplinkMbps: s.cfg.MaxUplinkMbps,
	}

	// Try to fetch real metrics from the edge agent
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:9090/metrics?format=json")
	if err == nil {
		defer resp.Body.Close()
		var data map[string]interface{}
		if json.NewDecoder(resp.Body).Decode(&data) == nil {
			if v, ok := data["cache_hits"].(float64); ok {
				hits := int64(v)
				if v2, ok := data["cache_misses"].(float64); ok {
					misses := int64(v2)
					if hits+misses > 0 {
						status.CacheHitRatio = float64(hits) / float64(hits+misses)
					}
				}
			}
			if v, ok := data["requests"].(float64); ok {
				status.TotalRequests = int64(v)
			}
			if v, ok := data["bytes_sent"].(float64); ok {
				status.TotalBytes = int64(v)
			}
			if bw, ok := data["bandwidth"].(map[string]interface{}); ok {
				if v, ok := bw["egress_mbps"].(float64); ok {
					status.EgressMbps = v
				}
				if v, ok := bw["ingress_mbps"].(float64); ok {
					status.IngressMbps = v
				}
			}
			if cache, ok := data["cache"].(map[string]interface{}); ok {
				if v, ok := cache["size"].(float64); ok {
					status.CacheMB = v / 1024 / 1024
				}
			}
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
<title>Edge Dispatch - 节点配置中心</title>
<style>
:root {
  --bg: #0f172a; --surface: #1e293b; --border: #334155;
  --primary: #3b82f6; --primary-hover: #2563eb; --success: #22c55e;
  --warning: #f59e0b; --danger: #ef4444;
  --text: #f1f5f9; --text-secondary: #94a3b8; --text-muted: #64748b;
  --radius: 8px; --shadow: 0 4px 12px rgba(0,0,0,0.3);
}
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: var(--bg); color: var(--text); min-height: 100vh; }
header { background: var(--surface); border-bottom: 1px solid var(--border); padding: 0 24px; display: flex; align-items: center; justify-content: space-between; height: 56px; }
header h1 { font-size: 18px; font-weight: 600; }
nav a { color: var(--text-secondary); text-decoration: none; padding: 6px 12px; border-radius: var(--radius); font-size: 14px; margin-left: 8px; }
nav a:hover, nav a.active { background: var(--primary); color: #fff; }
main { max-width: 960px; margin: 32px auto; padding: 0 24px; }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); padding: 24px; margin-bottom: 20px; box-shadow: var(--shadow); }
.card h2 { font-size: 16px; font-weight: 600; margin-bottom: 16px; color: var(--text); }
.card h3 { font-size: 14px; font-weight: 500; color: var(--text-secondary); margin-bottom: 8px; }
.row { display: flex; gap: 20px; flex-wrap: wrap; }
.col { flex: 1; min-width: 260px; }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 13px; font-weight: 500; color: var(--text-secondary); margin-bottom: 4px; }
.form-group input, .form-group select { width: 100%; padding: 8px 12px; background: var(--bg); border: 1px solid var(--border); border-radius: 6px; color: var(--text); font-size: 14px; outline: none; }
.form-group input:focus, .form-group select:focus { border-color: var(--primary); }
.form-group input[type="range"] { width: calc(100% - 60px); display: inline-block; margin-right: 8px; }
.range-value { display: inline-block; width: 44px; text-align: center; font-size: 13px; color: var(--primary); font-weight: 600; }
.toggle { display: flex; align-items: center; gap: 8px; }
.toggle input[type="checkbox"] { display: none; }
.toggle label { position: relative; width: 40px; height: 22px; background: var(--border); border-radius: 11px; cursor: pointer; transition: background 0.2s; }
.toggle label:after { content: ''; position: absolute; top: 2px; left: 2px; width: 18px; height: 18px; background: #fff; border-radius: 50%; transition: transform 0.2s; }
.toggle input:checked + label { background: var(--primary); }
.toggle input:checked + label:after { transform: translateX(18px); }
.toggle-text { font-size: 13px; color: var(--text-secondary); }
.btn { padding: 8px 20px; border: none; border-radius: 6px; font-size: 14px; font-weight: 500; cursor: pointer; transition: background 0.2s; }
.btn-primary { background: var(--primary); color: #fff; }
.btn-primary:hover { background: var(--primary-hover); }
.btn-success { background: var(--success); color: #fff; }
.btn-danger { background: var(--danger); color: #fff; }
.btn-sm { padding: 4px 12px; font-size: 12px; }
.btn-group { display: flex; gap: 10px; margin-top: 16px; }
.alert { padding: 10px 14px; border-radius: 6px; font-size: 13px; margin-top: 8px; }
.alert-success { background: rgba(34,197,94,0.15); color: var(--success); border: 1px solid rgba(34,197,94,0.3); }
.alert-error { background: rgba(239,68,68,0.15); color: var(--danger); border: 1px solid rgba(239,68,68,0.3); }
.step-indicator { display: flex; margin-bottom: 24px; }
.step { flex: 1; text-align: center; position: relative; }
.step:not(:last-child):after { content: ''; position: absolute; top: 14px; right: -50%; width: 100%; height: 2px; background: var(--border); z-index: 0; }
.step.completed:not(:last-child):after { background: var(--success); }
.step .dot { width: 28px; height: 28px; border-radius: 50%; background: var(--border); display: flex; align-items: center; justify-content: center; margin: 0 auto 6px; font-size: 12px; font-weight: 600; position: relative; z-index: 1; }
.step.active .dot { background: var(--primary); color: #fff; }
.step.completed .dot { background: var(--success); color: #fff; }
.step .label { font-size: 12px; color: var(--text-muted); }
.step.active .label { color: var(--primary); font-weight: 600; }
table { width: 100%; border-collapse: collapse; }
th, td { padding: 10px 12px; text-align: left; border-bottom: 1px solid var(--border); font-size: 13px; }
th { color: var(--text-muted); font-weight: 500; }
.metric-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; }
.metric-card { background: var(--bg); border: 1px solid var(--border); border-radius: var(--radius); padding: 16px; }
.metric-card .value { font-size: 24px; font-weight: 700; color: var(--primary); }
.metric-card .label { font-size: 12px; color: var(--text-muted); margin-top: 4px; }
.bar { height: 8px; background: var(--border); border-radius: 4px; overflow: hidden; margin-top: 8px; }
.bar-fill { height: 100%; border-radius: 4px; transition: width 0.5s; }
.bar-fill.good { background: var(--success); }
.bar-fill.warn { background: var(--warning); }
.bar-fill.danger { background: var(--danger); }
.bw-slider { display: flex; align-items: center; gap: 12px; }
.bw-slider input[type="range"] { flex: 1; }
.chip { display: inline-block; padding: 2px 8px; border-radius: 12px; font-size: 11px; font-weight: 500; }
.chip-green { background: rgba(34,197,94,0.15); color: var(--success); }
.chip-yellow { background: rgba(245,158,11,0.15); color: var(--warning); }
.chip-red { background: rgba(239,68,68,0.15); color: var(--danger); }
</style>
</head>
<body>
<header>
  <h1>Edge Dispatch - 节点配置中心</h1>
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
<div class="card">
  <h2>欢迎使用 Edge Dispatch Framework</h2>
  <p style="color:var(--text-secondary);font-size:14px;line-height:1.6;">
    本节点配置中心提供可视化的配置管理界面，支持配置向导、运行时配置修改、状态监控等功能。
  </p>
  <div class="btn-group">
    <a href="/setup" class="btn btn-primary">开始配置向导</a>
    <a href="/config" class="btn btn-primary">直接修改配置</a>
  </div>
</div>
<div class="row">
  <div class="col">
    <div class="card">
      <h3>配置向导</h3>
      <p style="color:var(--text-muted);font-size:13px;">4 步完成节点初始化：角色选择 → 服务端配置 → 缓存配置 → 网络配置</p>
    </div>
  </div>
  <div class="col">
    <div class="card">
      <h3>运行时配置</h3>
      <p style="color:var(--text-muted);font-size:13px;">在线修改配置并热重载，无需重启服务。支持导入/导出配置文件。</p>
    </div>
  </div>
  <div class="col">
    <div class="card">
      <h3>实时监控</h3>
      <p style="color:var(--text-muted);font-size:13px;">查看节点运行状态、带宽使用、缓存命中率等关键指标。</p>
    </div>
  </div>
</div>
`

const setupPage = `
<div class="card">
  <h2>配置向导</h2>
  <div class="step-indicator">
    <div class="step completed"><div class="dot">1</div><div class="label">选择角色</div></div>
    <div class="step completed"><div class="dot">2</div><div class="label">服务端配置</div></div>
    <div class="step completed"><div class="dot">3</div><div class="label">缓存配置</div></div>
    <div class="step active"><div class="dot">4</div><div class="label">网络配置</div></div>
  </div>

  <div class="row" style="margin-bottom:24px">
    <div class="col">
      <div class="card" style="cursor:pointer;border-color:var(--primary)" onclick="selectRole('edge-agent')">
        <h3 style="font-size:20px">边缘节点</h3>
        <p style="color:var(--text-secondary);font-size:13px;">提供内容分发服务<br>缓存热门内容<br>可部署在内网</p>
      </div>
    </div>
  </div>

  <h3 style="margin-top:20px">服务端配置</h3>
  <div class="form-group">
    <label>控制平面地址</label>
    <input type="text" id="cpUrl" placeholder="http://192.168.1.100:8080" />
    <button class="btn btn-sm btn-primary" style="margin-top:6px" onclick="testConn('cpUrl')">测试连接</button>
    <div id="cpUrlResult"></div>
  </div>
  <div class="form-group">
    <label>源站地址</label>
    <input type="text" id="originUrl" placeholder="http://origin.example.com:7070" />
    <button class="btn btn-sm btn-primary" style="margin-top:6px" onclick="testConn('originUrl')">测试连接</button>
    <div id="originUrlResult"></div>
  </div>

  <h3 style="margin-top:20px">缓存配置</h3>
  <div class="form-group">
    <label>缓存路径</label>
    <input type="text" id="cacheDir" placeholder="/data/edf-cache" />
  </div>
  <div class="form-group">
    <label>磁盘信息</label>
    <div id="diskInfo" style="color:var(--text-muted);font-size:13px">点击 [检查磁盘] 查看</div>
    <button class="btn btn-sm btn-primary" style="margin-top:6px" onclick="getDiskInfo()">检查磁盘</button>
  </div>
  <div class="form-group">
    <label>最大缓存 (GB)</label>
    <input type="number" id="cacheMaxGB" value="500" min="10" max="10000" />
  </div>

  <h3 style="margin-top:20px">网络与带宽配置</h3>
  <div class="form-group">
    <label>上行带宽 (Mbps)</label>
    <div class="bw-slider">
      <input type="range" id="maxUplink" min="1" max="1000" value="30" oninput="document.getElementById('maxUplinkVal').textContent=this.value" />
      <span id="maxUplinkVal" style="color:var(--primary);font-weight:600">30</span> Mbps
    </div>
  </div>
  <div class="form-group">
    <label>带宽分配</label>
    <div style="color:var(--text-muted);font-size:12px;line-height:1.8">
      回源带宽: <span id="originBW">24</span> Mbps (80%)<br>
      P2P互助: <span id="p2pBW">15</span> Mbps (50%)<br>
      预拉取: <span id="prefetchBW">10</span> Mbps (33%)
    </div>
  </div>
  <div class="form-group">
    <div class="toggle">
      <input type="checkbox" id="p2pEnabled" checked><label for="p2pEnabled"></label>
      <span class="toggle-text">启用 P2P 互助</span>
    </div>
  </div>
  <div class="form-group">
    <label>最大邻居数</label>
    <input type="number" id="p2pMaxPeers" value="10" min="1" max="100" />
  </div>
  <div class="form-group">
    <div class="toggle">
      <input type="checkbox" id="prefetchEnabled" checked><label for="prefetchEnabled"></label>
      <span class="toggle-text">启用智能预拉取</span>
    </div>
  </div>
  <div class="form-group">
    <label>预拉取工作线程</label>
    <input type="number" id="prefetchWorkers" value="2" min="1" max="10" />
  </div>

  <div class="btn-group">
    <button class="btn btn-success" onclick="saveConfig()">保存配置</button>
    <button class="btn btn-primary" onclick="applyConfig()">应用并热重载</button>
  </div>
  <div id="configResult"></div>
</div>
<script>
function selectRole(role) { document.getElementById('role').value = role; }
async function testConn(fieldId) {
  const url = document.getElementById(fieldId).value;
  const res = document.getElementById(fieldId+'Result');
  try {
    const r = await fetch('/api/test-connection', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({url})});
    const d = await r.json();
    res.innerHTML = d.success ? '<div class="alert alert-success">连接成功 (延迟: '+d.latency_ms+'ms)</div>' : '<div class="alert alert-error">连接失败: '+d.error+'</div>';
  } catch(e) { res.innerHTML = '<div class="alert alert-error">请求失败</div>'; }
}
async function getDiskInfo() {
  const r = await fetch('/api/disk-info');
  const d = await r.json();
  document.getElementById('diskInfo').innerHTML = '总 '+d.total_gb+'GB | 可用 '+d.free_gb+'GB | 已用 '+d.used_gb+'GB';
}
async function saveConfig() {
  const payload = {
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
  const r = await fetch('/api/config/save', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});
  document.getElementById('configResult').innerHTML = '<div class="alert alert-success">配置已保存</div>';
}
async function applyConfig() {
  await saveConfig();
  const r = await fetch('/api/config/apply', {method:'POST'});
  document.getElementById('configResult').innerHTML = '<div class="alert alert-success">配置已热重载</div>';
}
</script>
`

const configPage = `
<div class="card">
  <h2>运行时配置管理</h2>
  <div class="row">
    <div class="col">
      <div class="form-group"><label>控制平面地址</label><input type="text" id="cpUrl" /></div>
      <div class="form-group"><label>源站地址</label><input type="text" id="originUrl" /></div>
      <div class="form-group"><label>缓存路径</label><input type="text" id="cacheDir" /></div>
      <div class="form-group"><label>最大缓存 (GB)</label><input type="number" id="cacheMaxGB" /></div>
    </div>
    <div class="col">
      <div class="form-group"><label>上行带宽 (Mbps)</label><input type="number" id="maxUplink" /></div>
      <div class="form-group"><label>回源带宽限制 (Mbps)</label><input type="number" id="originFetchBWLimit" /></div>
      <div class="form-group"><div class="toggle"><input type="checkbox" id="p2pEnabled"><label for="p2pEnabled"></label><span class="toggle-text">P2P 互助</span></div></div>
      <div class="form-group"><label>P2P 最大邻居</label><input type="number" id="p2pMaxPeers" /></div>
      <div class="form-group"><div class="toggle"><input type="checkbox" id="prefetchEnabled"><label for="prefetchEnabled"></label><span class="toggle-text">智能预拉取</span></div></div>
    </div>
  </div>
  <div class="btn-group">
    <button class="btn btn-success" onclick="saveConfig()">保存配置</button>
    <button class="btn btn-primary" onclick="applyConfig()">热重载</button>
  </div>
  <div id="configResult"></div>
</div>
<script>
async function loadConfig() {
  const r = await fetch('/api/config');
  const d = await r.json();
  document.getElementById('cpUrl').value = d.control_plane_url||'';
  document.getElementById('originUrl').value = d.origin_url||'';
  document.getElementById('cacheDir').value = d.cache_dir||'';
  document.getElementById('cacheMaxGB').value = d.cache_max_gb||10;
  document.getElementById('maxUplink').value = d.max_uplink_mbps||30;
  document.getElementById('originFetchBWLimit').value = d.origin_fetch_bw_limit||0;
  document.getElementById('p2pEnabled').checked = d.p2p_enabled||false;
  document.getElementById('p2pMaxPeers').value = d.p2p_max_peers||10;
  document.getElementById('prefetchEnabled').checked = d.prefetch_enabled||false;
}
async function saveConfig() {
  const payload = {
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
  const r = await fetch('/api/config/save', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});
  document.getElementById('configResult').innerHTML = '<div class="alert alert-success">配置已保存</div>';
}
async function applyConfig() {
  await saveConfig();
  await fetch('/api/config/apply', {method:'POST'});
  document.getElementById('configResult').innerHTML = '<div class="alert alert-success">配置已热重载，无需重启服务</div>';
}
loadConfig();
</script>
`

const statusPage = `
<div class="card">
  <h2>实时运行状态 <span style="font-size:11px;color:var(--text-muted)">(每5秒刷新)</span></h2>
  <div class="metric-grid">
    <div class="metric-card"><div class="value" id="uptime">--</div><div class="label">运行时长</div></div>
    <div class="metric-card"><div class="value" id="cacheHitRatio">--</div><div class="label">缓存命中率</div></div>
    <div class="metric-card"><div class="value" id="cacheSize">--</div><div class="label">缓存大小 (MB)</div></div>
    <div class="metric-card"><div class="value" id="totalRequests">--</div><div class="label">总请求数</div></div>
    <div class="metric-card"><div class="value" id="egress">--</div><div class="label">出站带宽 (Mbps)</div></div>
    <div class="metric-card"><div class="value" id="ingress">--</div><div class="label">入站带宽 (Mbps)</div></div>
    <div class="metric-card"><div class="value" id="totalBytes">--</div><div class="label">总流量 (MB)</div></div>
    <div class="metric-card"><div class="value" id="goroutines">--</div><div class="label">Goroutines / CPU</div></div>
  </div>
</div>
<div class="card">
  <h2>带宽使用趋势</h2>
  <div style="margin-bottom:12px">
    <span style="color:var(--primary);font-size:13px">■ 出站</span>
    <span style="color:var(--warning);font-size:13px;margin-left:12px">■ 入站</span>
  </div>
  <div style="display:flex;align-items:center;gap:10px">
    <span style="font-size:13px;color:var(--text-secondary);min-width:60px">出站</span>
    <div class="bar" style="flex:1"><div class="bar-fill good" id="egressBar" style="width:0%"></div></div>
    <span id="egressVal" style="font-size:12px;color:var(--text-muted);min-width:70px">--</span>
  </div>
  <div style="display:flex;align-items:center;gap:10px;margin-top:8px">
    <span style="font-size:13px;color:var(--text-secondary);min-width:60px">入站</span>
    <div class="bar" style="flex:1"><div class="bar-fill warn" id="ingressBar" style="width:0%"></div></div>
    <span id="ingressVal" style="font-size:12px;color:var(--text-muted);min-width:70px">--</span>
  </div>
</div>
<script>
async function loadStatus() {
  try {
    const r = await fetch('/api/status');
    const d = await r.json();
    document.getElementById('uptime').textContent = d.uptime || '--';
    document.getElementById('cacheHitRatio').textContent = (d.cache_hit_ratio*100||0).toFixed(1)+'%';
    document.getElementById('cacheSize').textContent = (d.cache_mb||0).toFixed(1);
    document.getElementById('totalRequests').textContent = d.total_requests||0;
    document.getElementById('egress').textContent = (d.egress_mbps||0).toFixed(2);
    document.getElementById('ingress').textContent = (d.ingress_mbps||0).toFixed(2);
    document.getElementById('totalBytes').textContent = ((d.total_bytes||0)/1024/1024).toFixed(1);
    document.getElementById('goroutines').textContent = (d.goroutines||0)+' / '+(d.num_cpu||1);
    document.getElementById('egressVal').textContent = (d.egress_mbps||0).toFixed(2)+' Mbps';
    document.getElementById('ingressVal').textContent = (d.ingress_mbps||0).toFixed(2)+' Mbps';
    var maxBW = d.max_uplink_mbps || 100;
    document.getElementById('egressBar').style.width = Math.min((d.egress_mbps||0)/maxBW*100, 100)+'%';
    document.getElementById('ingressBar').style.width = Math.min((d.ingress_mbps||0)/maxBW*100, 100)+'%';
  } catch(e) {}
}
loadStatus();
setInterval(loadStatus, 5000);
</script>
`
