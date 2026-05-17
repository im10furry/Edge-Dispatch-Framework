package adminui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/darkinno/edge-dispatch-framework/internal/config"
	"github.com/darkinno/edge-dispatch-framework/internal/models"
)

type AdminUI struct {
	cfg     *config.ControlPlaneConfig
	metrics MetricsProvider
	httpSrv *http.Server
}

type MetricsProvider interface {
	GetNodes() ([]models.Node, error)
}

type DashboardData struct {
	OnlineNodes   int                 `json:"online_nodes"`
	OfflineNodes  int                 `json:"offline_nodes"`
	TotalNodes    int                 `json:"total_nodes"`
	Nodes         []NodeSummary       `json:"nodes"`
	Config        GlobalConfigSummary `json:"config"`
}

type NodeSummary struct {
	NodeID        string  `json:"node_id"`
	Name          string  `json:"name"`
	Region        string  `json:"region"`
	ISP           string  `json:"isp"`
	Status        string  `json:"status"`
	BandwidthMbps int64   `json:"bandwidth_mbps"`
	CacheHitRatio float64 `json:"cache_hit_ratio"`
	P2PEnabled    bool    `json:"p2p_enabled"`
}

type GlobalConfigSummary struct {
	SBEnabled  bool  `json:"sb_enabled"`
	SBThreshold int64 `json:"sb_threshold"`
	P2PEnabled bool  `json:"p2p_enabled"`
	MaxCandidates int `json:"max_candidates"`
}

func New(cfg *config.ControlPlaneConfig, metrics MetricsProvider) *AdminUI {
	ui := &AdminUI{cfg: cfg, metrics: metrics}
	mux := http.NewServeMux()
	mux.HandleFunc("/", ui.handleIndex)
	mux.HandleFunc("/nodes", ui.handleNodes)
	mux.HandleFunc("/api/dashboard", ui.handleAPIDashboard)
	mux.HandleFunc("/api/nodes", ui.handleAPINodes)

	ui.httpSrv = &http.Server{
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return ui
}

func (ui *AdminUI) Start() error {
	ln, err := net.Listen("tcp", ":8082")
	if err != nil {
		return fmt.Errorf("admin ui listen: %w", err)
	}
	slog.Info("control plane admin UI started", "addr", ":8082")
	go ui.httpSrv.Serve(ln)
	return nil
}

func (ui *AdminUI) Shutdown(ctx context.Context) error {
	return ui.httpSrv.Shutdown(ctx)
}

func (ui *AdminUI) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, cpHeader+cpDashboard+cpFooter)
}

func (ui *AdminUI) handleNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, cpHeader+cpNodes+cpFooter)
}

func (ui *AdminUI) handleAPIDashboard(w http.ResponseWriter, r *http.Request) {
	nodes, _ := ui.metrics.GetNodes()
	online, offline := 0, 0
	var summaries []NodeSummary
	for _, n := range nodes {
		s := NodeSummary{
			NodeID:        n.NodeID,
			Name:          n.Name,
			Region:        n.Region,
			ISP:           n.ISP,
			Status:        string(n.Status),
			BandwidthMbps: n.Capabilities.MaxUplinkMbps,
			P2PEnabled:    n.Capabilities.SupportsP2P,
		}
		switch n.Status {
		case models.NodeStatusActive, models.NodeStatusDegraded:
			online++
		case models.NodeStatusOffline:
			offline++
		}
		summaries = append(summaries, s)
	}
	data := DashboardData{
		OnlineNodes:  online,
		OfflineNodes: offline,
		TotalNodes:   len(nodes),
		Nodes:        summaries,
		Config: GlobalConfigSummary{
			SBEnabled:     ui.cfg.SmallBandwidthOptimization.Enabled,
			SBThreshold:   ui.cfg.SmallBandwidthOptimization.SmallBandwidthThreshold,
			P2PEnabled:    ui.cfg.SmallBandwidthOptimization.P2PEnabled,
			MaxCandidates: ui.cfg.MaxCandidates,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (ui *AdminUI) handleAPINodes(w http.ResponseWriter, r *http.Request) {
	nodes, _ := ui.metrics.GetNodes()
	var summaries []NodeSummary
	for _, n := range nodes {
		summaries = append(summaries, NodeSummary{
			NodeID:        n.NodeID,
			Name:          n.Name,
			Region:        n.Region,
			ISP:           n.ISP,
			Status:        string(n.Status),
			BandwidthMbps: n.Capabilities.MaxUplinkMbps,
			P2PEnabled:    n.Capabilities.SupportsP2P,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

const cpHeader = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Edge Dispatch — 控制平面</title>
<style>
:root {
  --bg:#0a0e1a;--bg2:#111827;--surface:rgba(17,25,45,0.7);
  --border:rgba(99,102,241,0.15);--primary:#6366f1;--primary-glow:rgba(99,102,241,0.3);
  --success:#10b981;--warning:#f59e0b;--danger:#ef4444;
  --text:#e2e8f0;--text-secondary:#94a3b8;--text-muted:#64748b;
  --radius:12px;--radius-sm:8px;--transition:0.3s cubic-bezier(0.4,0,0.2,1);
}
*{box-sizing:border-box;margin:0;padding:0}
body{
  font-family:'Inter',-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;
  background:var(--bg);
  background-image:radial-gradient(ellipse 80% 50% at 50% -20%,rgba(99,102,241,0.12),transparent);
  color:var(--text);min-height:100vh;animation:fadeIn 0.6s ease;
}
@keyframes fadeIn{from{opacity:0}to{opacity:1}}
@keyframes slideUp{from{opacity:0;transform:translateY(24px)}to{opacity:1;transform:translateY(0)}}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.6}}
header{
  background:rgba(10,14,26,0.85);backdrop-filter:blur(20px);
  border-bottom:1px solid var(--border);padding:0 32px;
  display:flex;align-items:center;justify-content:space-between;height:60px;
  position:sticky;top:0;z-index:100;
}
header .logo{display:flex;align-items:center;gap:10px}
header .logo .dot{width:10px;height:10px;border-radius:50%;background:var(--primary);box-shadow:0 0 12px var(--primary-glow);animation:pulse 2s infinite}
header h1{font-size:16px;font-weight:600;letter-spacing:-0.3px}
nav{display:flex;gap:4px}
nav a{color:var(--text-secondary);text-decoration:none;padding:8px 16px;border-radius:var(--radius-sm);font-size:13px;font-weight:500;transition:all var(--transition)}
nav a:hover{color:#fff;background:rgba(99,102,241,0.12)}
nav a.active{color:#fff;background:var(--primary);box-shadow:0 2px 12px var(--primary-glow)}
main{max-width:1100px;margin:0 auto;padding:28px 24px}
.card{
  background:var(--surface);backdrop-filter:blur(12px);
  border:1px solid var(--border);border-radius:var(--radius);
  padding:28px;margin-bottom:20px;
  animation:slideUp 0.5s ease both;
}
.card:nth-child(2){animation-delay:0.1s}
.card:hover{border-color:var(--primary);box-shadow:0 0 24px rgba(99,102,241,0.08)}
.card h2{font-size:18px;font-weight:600;margin-bottom:20px}
.metric-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:16px}
.metric-card{
  background:rgba(17,25,45,0.5);border:1px solid var(--border);
  border-radius:var(--radius);padding:20px;transition:all var(--transition)
}
.metric-card:hover{border-color:var(--primary);transform:translateY(-2px)}
.metric-card .value{font-size:32px;font-weight:800;letter-spacing:-1px}
.metric-card .label{font-size:12px;color:var(--text-muted);margin-top:6px;font-weight:500}
table{width:100%;border-collapse:collapse;margin-top:12px}
th,td{padding:12px 14px;text-align:left;border-bottom:1px solid var(--border);font-size:13px}
th{color:var(--text-muted);font-weight:600;font-size:11px;text-transform:uppercase;letter-spacing:0.5px}
tr{transition:background 0.2s}
tr:hover{background:rgba(99,102,241,0.04)}
td .status-dot{display:inline-block;width:8px;height:8px;border-radius:50%;margin-right:6px}
.status-active{background:var(--success);box-shadow:0 0 8px rgba(16,185,129,0.5)}
.status-offline{background:var(--danger)}
.status-registered{background:var(--warning)}
.badge{display:inline-flex;align-items:center;gap:4px;padding:3px 10px;border-radius:20px;font-size:11px;font-weight:600}
.badge-small{background:rgba(245,158,11,0.15);color:var(--warning)}
</style>
</head>
<body>
<header>
  <div class="logo"><div class="dot"></div><h1>Edge Dispatch 控制平面</h1></div>
  <nav><a href="/">仪表盘</a><a href="/nodes">节点管理</a></nav>
</header>
<main>
`

const cpFooter = `</main></body></html>`

const cpDashboard = `
<div class="card">
  <h2>&#128202; 集群概览</h2>
  <div class="metric-grid">
    <div class="metric-card"><div class="value" id="online" style="color:var(--success)">--</div><div class="label">&#128994; 在线节点</div></div>
    <div class="metric-card"><div class="value" id="offline" style="color:var(--danger)">--</div><div class="label">&#128308; 离线节点</div></div>
    <div class="metric-card"><div class="value" id="total" style="color:var(--primary)">--</div><div class="label">&#128179; 总节点数</div></div>
    <div class="metric-card"><div class="value" id="sbThreshold" style="color:var(--warning)">--</div><div class="label">&#9881; SB阈值 (Mbps)</div></div>
  </div>
</div>
<div class="card">
  <h2>&#128187; 活跃节点</h2>
  <table><thead><tr><th>节点</th><th>区域</th><th>ISP</th><th>带宽</th><th>P2P</th><th>状态</th></tr></thead><tbody id="nodeTable"></tbody></table>
</div>
<script>
async function load() {
  try {
    var r = await fetch('/api/dashboard');
    var d = await r.json();
    document.getElementById('online').textContent = d.online_nodes||0;
    document.getElementById('offline').textContent = d.offline_nodes||0;
    document.getElementById('total').textContent = d.total_nodes||0;
    document.getElementById('sbThreshold').textContent = (d.config&&d.config.sb_threshold)||'--';
    var tbody = document.getElementById('nodeTable');
    var rows = '';
    (d.nodes||[]).forEach(function(n) {
      var sc = n.status==='ACTIVE'?'status-active':n.status==='OFFLINE'?'status-offline':'status-registered';
      var bw = n.bandwidth_mbps;
      var bwTag = bw>0&&bw<50?' <span class="badge badge-small">SB</span>':'';
      rows += '<tr><td><span class="status-dot '+sc+'"></span><strong>'+n.name+'</strong><br><span style="font-size:11px;color:var(--text-muted)">'+n.node_id.substring(0,12)+'...</span></td><td>'+n.region+'</td><td>'+n.isp+'</td><td>'+bw+' Mbps'+bwTag+'</td><td>'+(n.p2p_enabled?'&#9989;':'&#10060;')+'</td><td>'+n.status+'</td></tr>';
    });
    tbody.innerHTML = rows || '<tr><td colspan="6" style="color:var(--text-muted);text-align:center">暂无节点</td></tr>';
  } catch(e) {}
}
load();
setInterval(load, 5000);
</script>
`

const cpNodes = `
<div class="card">
  <h2>&#128187; 全部节点</h2>
  <table><thead><tr><th>Node ID</th><th>名称</th><th>区域</th><th>ISP</th><th>带宽</th><th>P2P</th><th>状态</th></tr></thead><tbody id="nodeTable"></tbody></table>
</div>
<script>
async function load() {
  try {
    var r = await fetch('/api/nodes');
    var d = await r.json();
    var tbody = document.getElementById('nodeTable');
    var rows = '';
    (d||[]).forEach(function(n) {
      var sc = n.status==='ACTIVE'?'status-active':n.status==='OFFLINE'?'status-offline':'status-registered';
      rows += '<tr><td style="font-family:monospace;font-size:12px">'+n.node_id+'</td><td><strong>'+n.name+'</strong></td><td>'+n.region+'</td><td>'+n.isp+'</td><td>'+n.bandwidth_mbps+' Mbps</td><td>'+(n.p2p_enabled?'&#9989;':'&#10060;')+'</td><td><span class="status-dot '+sc+'"></span>'+n.status+'</td></tr>';
    });
    tbody.innerHTML = rows || '<tr><td colspan="7" style="color:var(--text-muted);text-align:center">暂无节点</td></tr>';
  } catch(e) {}
}
load();
setInterval(load, 5000);
</script>
`
