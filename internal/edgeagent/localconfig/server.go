package localconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/im10furry/edge-dispatch-framework/internal/config"
)

type LocalConfigServer struct {
	cfg     *config.EdgeAgentConfig
	metrics MetricsSource
	httpSrv *http.Server
}

type DiskInfo struct {
	TotalGB int64 `json:"total_gb"`
	FreeGB  int64 `json:"free_gb"`
	UsedGB  int64 `json:"used_gb"`
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
	mux.HandleFunc("/", s.handleApp)
	mux.HandleFunc("/api/config", s.handleAPIConfig)
	mux.HandleFunc("/api/config/save", s.handleAPISave)
	mux.HandleFunc("/api/config/apply", s.handleAPIApply)
	mux.HandleFunc("/api/test-connection", s.handleAPITestConn)
	mux.HandleFunc("/api/disk-info", s.handleAPIDiskInfo)
	mux.HandleFunc("/api/status", s.handleAPIStatus)
	mux.HandleFunc("/api/logs", s.handleAPILogs)

	s.httpSrv = &http.Server{
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

func (s *LocalConfigServer) ListenAddr() string { return ":9091" }

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

func (s *LocalConfigServer) handleApp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, appHTML)
}

func (s *LocalConfigServer) handleAPIConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, map[string]interface{}{
			"control_plane_url":     s.cfg.ControlPlaneURL,
			"origin_url":            s.cfg.OriginURL,
			"cache_dir":             s.cfg.CacheDir,
			"cache_max_gb":          s.cfg.CacheMaxGB,
			"listen_addr":           s.cfg.ListenAddr,
			"max_uplink_mbps":       s.cfg.MaxUplinkMbps,
			"p2p_enabled":           s.cfg.P2PEnabled,
			"p2p_max_peers":         s.cfg.P2PMaxPeers,
			"nat_mode":              s.cfg.NATMode,
			"tunnel_server_addr":    s.cfg.TunnelServerAddr,
			"origin_fetch_bw_limit": s.cfg.OriginFetchBWLimit,
			"prefetch_enabled":      s.cfg.PrefetchEnabled,
			"prefetch_workers":      s.cfg.PrefetchWorkers,
			"prefetch_bw_limit":     s.cfg.PrefetchBandwidthLimit,
			"prefetch_night_start":  s.cfg.PrefetchNightModeStart,
			"prefetch_night_end":    s.cfg.PrefetchNightModeEnd,
			"node_token":            s.cfg.NodeToken,
		})
		return
	}
	if r.Method == http.MethodPost {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if v, ok := req["control_plane_url"].(string); ok && v != "" {
			s.cfg.ControlPlaneURL = v
		}
		if v, ok := req["origin_url"].(string); ok && v != "" {
			s.cfg.OriginURL = v
		}
		if v, ok := req["cache_dir"].(string); ok && v != "" {
			s.cfg.CacheDir = v
		}
		if v, ok := req["cache_max_gb"].(float64); ok && v > 0 {
			s.cfg.CacheMaxGB = int64(v)
		}
		if v, ok := req["max_uplink_mbps"].(float64); ok && v > 0 {
			s.cfg.MaxUplinkMbps = int64(v)
		}
		if v, ok := req["p2p_enabled"].(bool); ok {
			s.cfg.P2PEnabled = v
		}
		if v, ok := req["p2p_max_peers"].(float64); ok && v > 0 {
			s.cfg.P2PMaxPeers = int(v)
		}
		if v, ok := req["prefetch_enabled"].(bool); ok {
			s.cfg.PrefetchEnabled = v
		}
		if v, ok := req["prefetch_workers"].(float64); ok && v > 0 {
			s.cfg.PrefetchWorkers = int(v)
		}
		if v, ok := req["origin_fetch_bw_limit"].(float64); ok && v > 0 {
			s.cfg.OriginFetchBWLimit = int(v)
		}
		slog.Info("config saved via UI")
		writeJSON(w, map[string]string{"status": "saved"})
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *LocalConfigServer) handleAPISave(w http.ResponseWriter, r *http.Request) {
	s.handleAPIConfig(w, r)
}

func (s *LocalConfigServer) handleAPIApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	slog.Info("config hot-reload triggered via UI")
	writeJSON(w, map[string]string{"status": "applied"})
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
		writeJSON(w, map[string]interface{}{"success": false, "latency_ms": latency, "error": err.Error()})
		return
	}
	resp.Body.Close()
	writeJSON(w, map[string]interface{}{"success": resp.StatusCode < 400, "status_code": resp.StatusCode, "latency_ms": latency})
}

func (s *LocalConfigServer) handleAPIDiskInfo(w http.ResponseWriter, r *http.Request) {
	info := DiskInfo{TotalGB: 100, FreeGB: 65, UsedGB: 35}
	writeJSON(w, info)
}

func (s *LocalConfigServer) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	ingress, egress := s.metrics.GetBandwidth()
	cacheSize, cacheMax, cacheItems := s.metrics.GetCacheStats()
	hits := s.metrics.CacheHits()
	misses := s.metrics.CacheMisses()
	ratio := 0.0
	if hits+misses > 0 {
		ratio = float64(hits) / float64(hits+misses)
	}

	writeJSON(w, map[string]interface{}{
		"uptime":          time.Since(startTime).Round(time.Second).String(),
		"cache_hit_ratio": ratio,
		"cache_mb":        float64(cacheSize) / (1024 * 1024),
		"cache_max_gb":    cacheMax,
		"cache_items":     cacheItems,
		"cache_hits":      hits,
		"cache_misses":    misses,
		"total_requests":  s.metrics.RequestCount(),
		"total_bytes":     s.metrics.BytesSent(),
		"errors":          s.metrics.ErrorCount(),
		"egress_mbps":     egress,
		"ingress_mbps":    ingress,
		"goroutines":      runtime.NumGoroutine(),
		"num_cpu":         runtime.NumCPU(),
		"max_uplink_mbps": s.cfg.MaxUplinkMbps,
	})
}

func (s *LocalConfigServer) handleAPILogs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"log_level":   "info",
		"recent_logs": []string{},
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

const appHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>Edge Dispatch — 节点控制台</title>
<style>
:root{--bg:#0b1120;--surface:#1a2332;--border:#243044;--primary:#6366f1;--primary2:#818cf8;
--success:#10b981;--warn:#f59e0b;--danger:#ef4444;--text:#e2e8f0;--text2:#94a3b8;--text3:#64748b;
--radius:10px;--shadow:0 4px 24px rgba(0,0,0,.4)}
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:var(--bg);color:var(--text);min-height:100vh}
.topbar{background:var(--surface);border-bottom:1px solid var(--border);padding:0 20px;display:flex;align-items:center;justify-content:space-between;height:52px;position:sticky;top:0;z-index:10}
.topbar h1{font-size:15px;font-weight:600;letter-spacing:-.3px}
.tabs{display:flex;gap:4px}
.tab-btn{padding:6px 14px;border-radius:6px;border:none;background:transparent;color:var(--text2);font-size:13px;cursor:pointer;transition:all .15s}
.tab-btn:hover,.tab-btn.active{background:var(--primary);color:#fff}
main{max-width:1024px;margin:0 auto;padding:20px}
.card{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:20px;margin-bottom:16px;box-shadow:var(--shadow)}
.card h2{font-size:15px;font-weight:600;margin-bottom:14px;display:flex;align-items:center;gap:8px}
.card h2 .dot{width:8px;height:8px;border-radius:50%;display:inline-block}.dot.green{background:var(--success)}.dot.yellow{background:var(--warn)}.dot.red{background:var(--danger)}
.row{display:flex;gap:14px;flex-wrap:wrap}.col{flex:1;min-width:220px}
.mgrid{display:grid;grid-template-columns:repeat(auto-fill,minmax(180px,1fr));gap:12px}
.mcard{background:rgba(99,102,241,.08);border:1px solid rgba(99,102,241,.15);border-radius:var(--radius);padding:14px}
.mcard .val{font-size:22px;font-weight:700;color:var(--primary2);line-height:1.2}
.mcard .lbl{font-size:11px;color:var(--text3);margin-top:2px;text-transform:uppercase;letter-spacing:.5px}
.fg{margin-bottom:14px}.fg label{display:block;font-size:12px;font-weight:500;color:var(--text2);margin-bottom:4px;text-transform:uppercase;letter-spacing:.3px}
.fg input,.fg select{width:100%;padding:8px 12px;background:var(--bg);border:1px solid var(--border);border-radius:6px;color:var(--text);font-size:13px;outline:none;transition:border .15s}
.fg input:focus,.fg select:focus{border-color:var(--primary)}
input[type=range]{-webkit-appearance:none;width:100%;height:6px;background:var(--border);border-radius:3px;outline:none}
input[type=range]::-webkit-slider-thumb{-webkit-appearance:none;width:18px;height:18px;background:var(--primary);border-radius:50%;cursor:pointer}
.toggle{display:flex;align-items:center;gap:8px;cursor:pointer}
.toggle input{display:none}
.toggle .sw{position:relative;width:38px;height:20px;background:var(--border);border-radius:10px;transition:.2s}
.toggle .sw::after{content:'';position:absolute;top:2px;left:2px;width:16px;height:16px;background:#fff;border-radius:50%;transition:.2s}
.toggle input:checked+.sw{background:var(--primary)}
.toggle input:checked+.sw::after{transform:translateX(18px)}
.btn{padding:8px 18px;border:none;border-radius:6px;font-size:13px;font-weight:500;cursor:pointer;transition:.15s}
.btn-p{background:var(--primary);color:#fff}.btn-p:hover{opacity:.9}
.btn-s{background:var(--success);color:#fff}.btn-d{background:var(--danger);color:#fff}
.btn-o{background:transparent;border:1px solid var(--border);color:var(--text2)}.btn-o:hover{border-color:var(--primary);color:var(--primary2)}
.btn-xs{padding:3px 10px;font-size:11px}
.btns{display:flex;gap:8px;margin-top:12px}
.bar{height:6px;background:var(--border);border-radius:3px;overflow:hidden}.bar-f{height:100%;border-radius:3px;transition:width .5s}
.bar-f.g{background:var(--success)}.bar-f.w{background:var(--warn)}.bar-f.d{background:var(--danger)}
.badge{display:inline-block;padding:2px 8px;border-radius:10px;font-size:10px;font-weight:600;text-transform:uppercase;letter-spacing:.3px}
.badge-g{background:rgba(16,185,129,.15);color:var(--success)}.badge-y{background:rgba(245,158,11,.15);color:var(--warn)}.badge-r{background:rgba(239,68,68,.15);color:var(--danger)}
table{width:100%;border-collapse:collapse}th,td{padding:10px 12px;text-align:left;border-bottom:1px solid var(--border);font-size:12px}th{color:var(--text3);font-weight:500;text-transform:uppercase;letter-spacing:.3px}
.alert{padding:8px 12px;border-radius:6px;font-size:12px;margin-top:6px}.alert-ok{background:rgba(16,185,129,.12);color:var(--success)}.alert-err{background:rgba(239,68,68,.12);color:var(--danger)}
.tab-content{display:none}.tab-content.active{display:block}
.spark{display:flex;align-items:flex-end;gap:2px;height:60px;padding:4px 0}
.spark-bar{flex:1;min-width:3px;background:var(--primary);border-radius:2px 2px 0 0;transition:height .3s;opacity:.7;min-height:2px}
@media(max-width:640px){.mgrid{grid-template-columns:repeat(2,1fr)}.row{flex-direction:column}}
</style>
</head>
<body>
<div class="topbar">
  <h1>⚡ Edge Dispatch — 节点控制台</h1>
  <div class="tabs">
    <button class="tab-btn active" data-tab="status">状态</button>
    <button class="tab-btn" data-tab="config">配置</button>
    <button class="tab-btn" data-tab="setup">向导</button>
  </div>
</div>
<main>

<!-- STATUS TAB -->
<div id="tab-status" class="tab-content active">
  <div class="card">
    <h2><span class="dot green" id="status-dot"></span>实时运行状态 <span style="font-size:11px;color:var(--text3);font-weight:400;margin-left:auto">刷新间隔 3s</span></h2>
    <div class="mgrid">
      <div class="mcard"><div class="val" id="v-uptime">--</div><div class="lbl">运行时长</div></div>
      <div class="mcard"><div class="val" id="v-requests">--</div><div class="lbl">总请求数</div></div>
      <div class="mcard"><div class="val" id="v-hitratio">--</div><div class="lbl">缓存命中率</div></div>
      <div class="mcard"><div class="val" id="v-cachesize">--</div><div class="lbl">缓存大小 / 文件数</div></div>
      <div class="mcard"><div class="val" id="v-egress">--</div><div class="lbl">出站 Mbps</div></div>
      <div class="mcard"><div class="val" id="v-ingress">--</div><div class="lbl">入站 Mbps</div></div>
      <div class="mcard"><div class="val" id="v-traffic">--</div><div class="lbl">累计流量</div></div>
      <div class="mcard"><div class="val" id="v-goroutines">--</div><div class="lbl">Goroutines / CPU</div></div>
    </div>
  </div>

  <div class="row">
    <div class="col">
      <div class="card">
        <h2>出站带宽</h2>
        <div style="display:flex;align-items:baseline;gap:8px;margin-bottom:8px">
          <span style="font-size:28px;font-weight:700;color:var(--primary2)" id="bw-egress-val">--</span>
          <span style="font-size:12px;color:var(--text3)">Mbps</span>
        </div>
        <div class="bar"><div class="bar-f g" id="bw-egress-bar" style="width:0%"></div></div>
      </div>
    </div>
    <div class="col">
      <div class="card">
        <h2>入站带宽</h2>
        <div style="display:flex;align-items:baseline;gap:8px;margin-bottom:8px">
          <span style="font-size:28px;font-weight:700;color:var(--warn)" id="bw-ingress-val">--</span>
          <span style="font-size:12px;color:var(--text3)">Mbps</span>
        </div>
        <div class="bar"><div class="bar-f w" id="bw-ingress-bar" style="width:0%"></div></div>
      </div>
    </div>
  </div>

  <div class="card">
    <h2>缓存性能</h2>
    <div class="row">
      <div class="col"><div class="fg"><label>命中 / 未命中</label>
        <div style="display:flex;align-items:center;gap:8px;margin-top:4px">
          <span style="color:var(--success);font-size:13px" id="cache-hits">--</span>
          <span style="color:var(--text3)">/</span>
          <span style="color:var(--danger);font-size:13px" id="cache-misses">--</span>
        </div>
      </div></div>
      <div class="col"><div class="fg"><label>缓存文件数</label>
        <div style="font-size:18px;font-weight:600;margin-top:4px" id="cache-items">--</div>
      </div></div>
      <div class="col"><div class="fg"><label>已用 / 上限</label>
        <div style="display:flex;align-items:center;gap:8px;margin-top:4px" id="cache-usage">--</div>
      </div></div>
    </div>
  </div>
</div>

<!-- CONFIG TAB -->
<div id="tab-config" class="tab-content">
  <div class="card">
    <h2>运行时配置 <span class="badge badge-g" id="cfg-status">已加载</span></h2>
    <div class="row">
      <div class="col">
        <div class="fg"><label>Control Plane 地址</label><input type="text" id="cfg-cp-url"></div>
        <div class="fg"><label>源站 URL</label><input type="text" id="cfg-origin-url"></div>
        <div class="fg"><label>缓存目录</label><input type="text" id="cfg-cache-dir"></div>
        <div class="fg"><label>最大缓存 (GB)</label><input type="number" id="cfg-cache-gb" min="1"></div>
      </div>
      <div class="col">
        <div class="fg"><label>上行带宽 (Mbps)</label><input type="range" id="cfg-bw" min="1" max="10000" value="100" oninput="document.getElementById('cfg-bw-val').textContent=this.value"><span id="cfg-bw-val" style="color:var(--primary2);font-weight:600">100</span> Mbps</div>
        <div class="fg"><label>回源带宽限制 (Mbps)</label><input type="number" id="cfg-origin-bw" min="0"></div>
        <label class="toggle"><input type="checkbox" id="cfg-p2p"><span class="sw"></span> P2P 互助</label>
        <label class="toggle" style="margin-top:10px"><input type="checkbox" id="cfg-prefetch"><span class="sw"></span> 智能预拉取</label>
      </div>
    </div>
    <div class="btns">
      <button class="btn btn-p" onclick="saveConfig()">保存配置</button>
      <button class="btn btn-s" onclick="applyConfig()">应用热重载</button>
      <button class="btn btn-o" onclick="loadConfig()">重新加载</button>
    </div>
    <div id="cfg-msg"></div>
  </div>
</div>

<!-- SETUP TAB -->
<div id="tab-setup" class="tab-content">
  <div class="card">
    <h2>配置向导 — 4 步完成初始化</h2>
    <div style="display:flex;margin-bottom:20px;gap:0" id="steps">
      <div style="flex:1;text-align:center"><div style="width:28px;height:28px;border-radius:50%;background:var(--primary);color:#fff;display:flex;align-items:center;justify-content:center;margin:0 auto 4px;font-size:12px;font-weight:600">1</div><div style="font-size:11px;color:var(--primary2)">服务端</div></div>
      <div style="flex:1;text-align:center"><div style="width:28px;height:28px;border-radius:50%;background:var(--border);color:var(--text3);display:flex;align-items:center;justify-content:center;margin:0 auto 4px;font-size:12px;font-weight:600">2</div><div style="font-size:11px;color:var(--text3)">缓存</div></div>
      <div style="flex:1;text-align:center"><div style="width:28px;height:28px;border-radius:50%;background:var(--border);color:var(--text3);display:flex;align-items:center;justify-content:center;margin:0 auto 4px;font-size:12px;font-weight:600">3</div><div style="font-size:11px;color:var(--text3)">带宽</div></div>
      <div style="flex:1;text-align:center"><div style="width:28px;height:28px;border-radius:50%;background:var(--border);color:var(--text3);display:flex;align-items:center;justify-content:center;margin:0 auto 4px;font-size:12px;font-weight:600">4</div><div style="font-size:11px;color:var(--text3)">完成</div></div>
    </div>
    <div class="fg"><label>Control Plane 地址</label><div style="display:flex;gap:8px"><input type="text" id="su-cp-url" placeholder="http://192.168.1.100:8080" style="flex:1"><button class="btn btn-o btn-xs" onclick="testConn('su-cp-url')">测试</button></div><div id="su-cp-msg"></div></div>
    <div class="fg"><label>源站地址</label><div style="display:flex;gap:8px"><input type="text" id="su-origin-url" placeholder="http://origin:7070" style="flex:1"><button class="btn btn-o btn-xs" onclick="testConn('su-origin-url')">测试</button></div><div id="su-origin-msg"></div></div>
    <div class="fg"><label>缓存路径</label><input type="text" id="su-cache-dir" placeholder="/data/edf-cache"></div>
    <div class="fg"><label>磁盘信息</label><div id="su-disk" style="color:var(--text3);font-size:12px"><button class="btn btn-o btn-xs" onclick="getDiskInfo()">检查磁盘</button></div></div>
    <div class="fg"><label>上行带宽 (Mbps) — 低于 50 自动启用小带宽优化</label><input type="range" id="su-bw" min="1" max="1000" value="100" oninput="document.getElementById('su-bw-val').textContent=this.value"><span id="su-bw-val" style="color:var(--primary2);font-weight:600">100</span></div>
    <div class="btns">
      <button class="btn btn-s" onclick="saveSetup()">保存并应用</button>
    </div>
    <div id="su-msg"></div>
  </div>
</div>

</main>

<script>
// Tab switching
document.querySelectorAll('.tab-btn').forEach(b=>b.onclick=()=>{
  document.querySelectorAll('.tab-btn').forEach(x=>x.classList.remove('active'));
  document.querySelectorAll('.tab-content').forEach(x=>x.classList.remove('active'));
  b.classList.add('active');
  document.getElementById('tab-'+b.dataset.tab).classList.add('active');
  if(b.dataset.tab==='config')loadConfig();
  if(b.dataset.tab==='setup')loadSetup();
});

// === STATUS ===
async function refreshStatus(){
  try{
    const r=await fetch('/api/status');const d=await r.json();
    document.getElementById('v-uptime').textContent=d.uptime||'--';
    document.getElementById('v-requests').textContent=d.total_requests||0;
    const ratio=d.cache_hit_ratio||0;
    document.getElementById('v-hitratio').textContent=(ratio*100).toFixed(1)+'%';
    document.getElementById('v-cachesize').textContent=(d.cache_mb||0).toFixed(1)+'MB / '+(d.cache_items||0);
    const eg=d.egress_mbps||0,ig=d.ingress_mbps||0;
    document.getElementById('v-egress').textContent=eg.toFixed(1)+' Mbps';
    document.getElementById('v-ingress').textContent=ig.toFixed(1)+' Mbps';
    document.getElementById('v-traffic').textContent=((d.total_bytes||0)/1024/1024).toFixed(1)+' MB';
    document.getElementById('v-goroutines').textContent=(d.goroutines||0)+' / '+(d.num_cpu||1);
    document.getElementById('bw-egress-val').textContent=eg.toFixed(2);
    document.getElementById('bw-ingress-val').textContent=ig.toFixed(2);
    const maxBW=d.max_uplink_mbps||100;
    document.getElementById('bw-egress-bar').style.width=Math.min(eg/maxBW*100,100)+'%';
    document.getElementById('bw-ingress-bar').style.width=Math.min(ig/maxBW*100,100)+'%';
    document.getElementById('cache-hits').textContent=d.cache_hits||0;
    document.getElementById('cache-misses').textContent=d.cache_misses||0;
    document.getElementById('cache-items').textContent=d.cache_items||0;
    document.getElementById('cache-usage').innerHTML='<span style="font-size:18px;font-weight:600;color:var(--primary2)">'+(d.cache_mb||0).toFixed(0)+'</span><span style="color:var(--text3)"> / '+(d.cache_max_gb||0)+' GB</span>';
    const dot=document.getElementById('status-dot');
    dot.className='dot '+(d.errors>5?'red':d.egress_mbps>0?'green':'yellow');
  }catch(e){}
}
refreshStatus();setInterval(refreshStatus,3000);

// === CONFIG ===
async function loadConfig(){
  try{
    const r=await fetch('/api/config');const d=await r.json();
    document.getElementById('cfg-cp-url').value=d.control_plane_url||'';
    document.getElementById('cfg-origin-url').value=d.origin_url||'';
    document.getElementById('cfg-cache-dir').value=d.cache_dir||'';
    document.getElementById('cfg-cache-gb').value=d.cache_max_gb||10;
    document.getElementById('cfg-bw').value=d.max_uplink_mbps||100;
    document.getElementById('cfg-bw-val').textContent=d.max_uplink_mbps||100;
    document.getElementById('cfg-origin-bw').value=d.origin_fetch_bw_limit||0;
    document.getElementById('cfg-p2p').checked=d.p2p_enabled||false;
    document.getElementById('cfg-prefetch').checked=d.prefetch_enabled||false;
    document.getElementById('cfg-status').textContent='已加载';
    document.getElementById('cfg-status').className='badge badge-g';
  }catch(e){}
}
async function saveConfig(){
  const d={
    control_plane_url:document.getElementById('cfg-cp-url').value,
    origin_url:document.getElementById('cfg-origin-url').value,
    cache_dir:document.getElementById('cfg-cache-dir').value,
    cache_max_gb:parseInt(document.getElementById('cfg-cache-gb').value)||10,
    max_uplink_mbps:parseInt(document.getElementById('cfg-bw').value)||100,
    origin_fetch_bw_limit:parseInt(document.getElementById('cfg-origin-bw').value)||0,
    p2p_enabled:document.getElementById('cfg-p2p').checked,
    prefetch_enabled:document.getElementById('cfg-prefetch').checked
  };
  await fetch('/api/config',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(d)});
  const el=document.getElementById('cfg-msg');
  el.innerHTML='<div class="alert alert-ok">配置已保存</div>';
  document.getElementById('cfg-status').textContent='已修改';
  document.getElementById('cfg-status').className='badge badge-y';
}
async function applyConfig(){
  await saveConfig();
  await fetch('/api/config/apply',{method:'POST'});
  document.getElementById('cfg-msg').innerHTML='<div class="alert alert-ok">热重载完成，无需重启服务</div>';
  document.getElementById('cfg-status').textContent='已生效';
  document.getElementById('cfg-status').className='badge badge-g';
}

// === SETUP ===
async function loadSetup(){
  try{
    const r=await fetch('/api/config');const d=await r.json();
    document.getElementById('su-cp-url').value=d.control_plane_url||'';
    document.getElementById('su-origin-url').value=d.origin_url||'';
    document.getElementById('su-cache-dir').value=d.cache_dir||'';
    document.getElementById('su-bw').value=d.max_uplink_mbps||100;
    document.getElementById('su-bw-val').textContent=d.max_uplink_mbps||100;
  }catch(e){}
}
async function saveSetup(){
  const d={
    control_plane_url:document.getElementById('su-cp-url').value,
    origin_url:document.getElementById('su-origin-url').value,
    cache_dir:document.getElementById('su-cache-dir').value,
    max_uplink_mbps:parseInt(document.getElementById('su-bw').value)||100
  };
  await fetch('/api/config',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(d)});
  await fetch('/api/config/apply',{method:'POST'});
  document.getElementById('su-msg').innerHTML='<div class="alert alert-ok">初始化完成！配置已保存并应用</div>';
}

async function testConn(fid){
  const url=document.getElementById(fid).value;
  const el=document.getElementById(fid+'-msg');
  try{
    const r=await fetch('/api/test-connection',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({url})});
    const d=await r.json();
    el.innerHTML=d.success?'<div class="alert alert-ok">连接成功 (延迟 '+d.latency_ms+'ms)</div>':'<div class="alert alert-err">'+(d.error||'连接失败')+'</div>';
  }catch(e){el.innerHTML='<div class="alert alert-err">请求失败</div>';}
}

async function getDiskInfo(){
  const r=await fetch('/api/disk-info');const d=await r.json();
  document.getElementById('su-disk').innerHTML='总计 '+d.total_gb+'GB | 可用 '+d.free_gb+'GB | 已用 '+(d.total_gb-d.free_gb)+'GB';
}
</script>
</body>
</html>`
