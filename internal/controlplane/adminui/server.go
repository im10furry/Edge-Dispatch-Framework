package adminui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	OtherNodes    int                 `json:"other_nodes"`
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
	mux.HandleFunc("/prewarm", ui.handlePrewarmPage)
	mux.HandleFunc("/api/dashboard", ui.handleAPIDashboard)
	mux.HandleFunc("/api/nodes", ui.handleAPINodes)
	mux.HandleFunc("/api/prewarm", ui.handleAPIPrewarm)

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

func (ui *AdminUI) handlePrewarmPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, cpHeader+cpPrewarm+cpFooter)
}

func (ui *AdminUI) handleNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, cpHeader+cpNodes+cpFooter)
}

func (ui *AdminUI) handleAPIDashboard(w http.ResponseWriter, r *http.Request) {
	nodes, _ := ui.metrics.GetNodes()
	online, offline, other := 0, 0, 0
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
		case models.NodeStatusRegistered:
			other++
		case models.NodeStatusOffline:
			offline++
		default:
			other++
		}
		summaries = append(summaries, s)
	}
	data := DashboardData{
		OnlineNodes:  online,
		OfflineNodes: offline,
		OtherNodes:   other,
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

func (ui *AdminUI) handleAPIPrewarm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, _ := io.ReadAll(r.Body)
	port := portFromAddr(ui.cfg.ListenAddr)
	req, _ := http.NewRequest("POST", "http://localhost:"+port+"/v1/tasks/prewarm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ui.cfg.TokenSecret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "prewarm failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func portFromAddr(addr string) string {
	_, port, _ := net.SplitHostPort(addr)
	if port == "" {
		return "8080"
	}
	return port
}

const cpHeader = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>Edge Dispatch — 控制平面</title>
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
radial-gradient(ellipse 40% 60% at 80% 80%,rgba(52,211,153,.04),transparent);
}
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
header .logo .dot{width:8px;height:8px;border-radius:50%;background:var(--success);animation:glow 2s infinite}
header h1{font-size:15px;font-weight:700;letter-spacing:-.3px}
nav{display:flex;gap:2px}
nav a{color:var(--text3);text-decoration:none;padding:7px 14px;border-radius:8px;font-size:12.5px;font-weight:500;transition:all var(--t)}
nav a:hover{color:var(--text);background:rgba(129,140,248,.1)}
nav a.active{color:#fff;background:linear-gradient(135deg,var(--primary-2),var(--primary));box-shadow:0 2px 12px rgba(99,102,241,.35)}
main{max-width:1100px;margin:0 auto;padding:24px 20px 40px}

.card{
background:var(--surface);backdrop-filter:blur(20px);
-webkit-backdrop-filter:blur(20px);
border:1px solid var(--border);border-radius:var(--r);
padding:26px 28px;margin-bottom:18px;
animation:slideUp .45s ease both;
}
.card:nth-child(2){animation-delay:.08s}
.card:nth-child(3){animation-delay:.16s}
.card:hover{border-color:var(--border-hover)}
.card h2{font-size:17px;font-weight:700;margin-bottom:18px;display:flex;align-items:center;gap:8px}

.stats-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:14px}
.stat-card{
background:var(--surface2);border:1px solid var(--border);border-radius:var(--r);
padding:18px 20px;transition:all var(--t);animation:slideUp .35s ease both;
position:relative;overflow:hidden;
}
.stat-card::before{content:'';position:absolute;top:0;left:0;right:0;height:2px;background:linear-gradient(90deg,transparent,var(--primary),transparent);opacity:0;transition:opacity var(--t)}
.stat-card:hover::before{opacity:.6}
.stat-card:hover{border-color:var(--border-hover);transform:translateY(-2px)}
.stat-card .value{font-size:26px;font-weight:800;letter-spacing:-.5px;margin-bottom:3px}
.stat-card .label{font-size:11.5px;color:var(--text4);font-weight:500}

table{width:100%;border-collapse:collapse;margin-top:10px}
th,td{padding:10px 12px;text-align:left;border-bottom:1px solid var(--border);font-size:13px}
th{color:var(--text4);font-weight:600;font-size:10.5px;text-transform:uppercase;letter-spacing:.5px;padding-bottom:10px}
tr{transition:background .15s}
tr:hover{background:rgba(129,140,248,.03)}
.status{display:inline-flex;align-items:center;gap:6px}
.status-dot{width:7px;height:7px;border-radius:50%;flex-shrink:0}
.dot-active{background:var(--success);box-shadow:0 0 6px rgba(52,211,153,.5)}
.dot-registered{background:var(--warning);box-shadow:0 0 6px rgba(251,191,36,.3)}
.dot-offline{background:var(--danger)}
.dot-other{background:var(--text4)}

.badge{display:inline-flex;align-items:center;gap:4px;padding:3px 9px;border-radius:20px;font-size:10.5px;font-weight:600}
.badge-success{background:rgba(52,211,153,.12);color:var(--success)}
.badge-warning{background:rgba(251,191,36,.12);color:var(--warning)}
.badge-danger{background:rgba(248,113,113,.12);color:var(--danger)}
.badge-sb{background:rgba(251,191,36,.1);color:var(--warning)}

.form-group{margin-bottom:14px}
.form-group label{display:block;font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:.6px;color:var(--text4);margin-bottom:5px}
.form-group input,textarea{width:100%;padding:9px 13px;background:var(--bg2);border:1px solid var(--border);border-radius:var(--r3);color:var(--text);font-size:13.5px;outline:none;transition:all var(--t);font-family:inherit}
.form-group input:focus,textarea:focus{border-color:var(--primary);box-shadow:0 0 0 3px rgba(129,140,248,.12)}
textarea{resize:vertical}

.btn{display:inline-flex;align-items:center;gap:6px;padding:9px 18px;border:none;border-radius:var(--r3);font-size:13px;font-weight:600;cursor:pointer;transition:all var(--t);font-family:inherit}
.btn-primary{background:linear-gradient(135deg,var(--primary-2),var(--primary));color:#fff;box-shadow:0 2px 10px rgba(99,102,241,.3)}
.btn-primary:hover{transform:translateY(-1px);box-shadow:0 4px 18px rgba(99,102,241,.4)}
.btn:disabled{opacity:.5;pointer-events:none}
.btn-group{display:flex;gap:10px;margin-top:16px;flex-wrap:wrap}

.alert{padding:11px 15px;border-radius:var(--r3);font-size:13px;margin-top:10px;display:flex;align-items:center;gap:8px;animation:slideUp .25s ease}
.alert::before{content:'';width:7px;height:7px;border-radius:50%;flex-shrink:0}
.alert-success{background:rgba(52,211,153,.06);color:var(--success);border:1px solid rgba(52,211,153,.15)}
.alert-success::before{background:var(--success)}
.alert-error{background:rgba(248,113,113,.06);color:var(--danger);border:1px solid rgba(248,113,113,.15)}
.alert-error::before{background:var(--danger)}
.alert-info{background:rgba(129,140,248,.06);color:var(--primary);border:1px solid rgba(129,140,248,.15)}
.alert-info::before{background:var(--primary)}

.spinner{width:18px;height:18px;border:2px solid var(--border);border-top-color:var(--primary);border-radius:50%;animation:spin .6s linear infinite;display:inline-block}

.emptystate{text-align:center;padding:36px 20px;color:var(--text4)}
.emptystate .icon{font-size:40px;margin-bottom:10px;opacity:.3}

@media(max-width:640px){
header{padding:0 14px}header h1{font-size:13px}nav a{padding:5px 9px;font-size:11px}
.card{padding:18px 16px}
}
</style>
</head>
<body>
<header>
<div class="logo"><div class="dot"></div><h1>Edge Dispatch 控制平面</h1></div>
<nav><a href="/">仪表盘</a><a href="/nodes">节点管理</a><a href="/prewarm">预热下发</a></nav>
</header>
<main>
`

const cpFooter = `</main></body></html>`

const cpDashboard = `
<div class="card">
  <h2>&#128202; 集群概览</h2>
  <div class="stats-grid">
    <div class="stat-card"><div class="value" id="online" style="color:var(--success)">--</div><div class="label">&#128994; 在线节点</div></div>
    <div class="stat-card"><div class="value" id="offline" style="color:var(--danger)">--</div><div class="label">&#128308; 离线节点</div></div>
    <div class="stat-card"><div class="value" id="other" style="color:var(--warning)">--</div><div class="label">&#128993; 其他状态</div></div>
    <div class="stat-card"><div class="value" id="total" style="color:var(--primary)">--</div><div class="label">&#128179; 总节点数</div></div>
  </div>
</div>
<div class="card">
  <h2>&#128187; 活跃节点</h2>
  <table><thead><tr><th>节点 ID</th><th>名称</th><th>区域</th><th>ISP</th><th>带宽</th><th>P2P</th><th>状态</th></tr></thead><tbody id="nodeTable"></tbody></table>
</div>
<script>
function dotClass(s){return s==='ACTIVE'||s==='DEGRADED'?'dot-active':s==='REGISTERED'?'dot-registered':s==='OFFLINE'?'dot-offline':'dot-other'}
async function load(){
try{
var r=await fetch('/api/dashboard');var d=await r.json();
document.getElementById('online').textContent=d.online_nodes||0;
document.getElementById('offline').textContent=d.offline_nodes||0;
document.getElementById('other').textContent=d.other_nodes||0;
document.getElementById('total').textContent=d.total_nodes||0;
var rows='';
(d.nodes||[]).forEach(function(n){
var bw=n.bandwidth_mbps;var bwTag=bw>0&&bw<50?' <span class="badge badge-sb">SB</span>':'';
rows+='<tr><td style="font-family:monospace;font-size:11.5px">'+n.node_id.substring(0,16)+'</td><td><strong>'+n.name+'</strong></td><td>'+n.region+'</td><td>'+n.isp+'</td><td>'+bw+' Mbps'+bwTag+'</td><td>'+(n.p2p_enabled?'&#9989;':'&#10060;')+'</td><td><span class="status"><span class="status-dot '+dotClass(n.status)+'"></span>'+n.status+'</span></td></tr>';
});
document.getElementById('nodeTable').innerHTML=rows||'<tr><td colspan="7"><div class="emptystate"><div class="icon">&#128187;</div><p>暂无节点数据</p></div></td></tr>';
}catch(e){}
}
load();setInterval(load,8000);
</script>
`

const cpNodes = `
<div class="card">
  <h2>&#128269; 全部节点</h2>
  <table><thead><tr><th>Node ID</th><th>名称</th><th>区域</th><th>ISP</th><th>带宽</th><th>P2P</th><th>状态</th></tr></thead><tbody id="nodeTable"></tbody></table>
</div>
<script>
function dotClass(s){return s==='ACTIVE'||s==='DEGRADED'?'dot-active':s==='REGISTERED'?'dot-registered':s==='OFFLINE'?'dot-offline':'dot-other'}
async function load(){
try{
var r=await fetch('/api/nodes');var d=await r.json();
var rows='';
(d||[]).forEach(function(n){
rows+='<tr><td style="font-family:monospace;font-size:11.5px">'+n.node_id+'</td><td><strong>'+n.name+'</strong></td><td>'+n.region+'</td><td>'+n.isp+'</td><td>'+n.bandwidth_mbps+' Mbps</td><td>'+(n.p2p_enabled?'&#9989;':'&#10060;')+'</td><td><span class="status"><span class="status-dot '+dotClass(n.status)+'"></span>'+n.status+'</span></td></tr>';
});
document.getElementById('nodeTable').innerHTML=rows||'<tr><td colspan="7"><div class="emptystate"><div class="icon">&#128187;</div><p>暂无节点</p></div></td></tr>';
}catch(e){}
}
load();setInterval(load,8000);
</script>
`

const cpPrewarm = `
<div class="card">
  <h2>&#128293; 内容预热下发</h2>
  <p style="color:var(--text3);font-size:13px;margin-bottom:16px">将指定内容主动推送到所有活跃边缘节点，提前缓存热门资源。</p>
  <div class="form-group"><label>资源 Key 列表（每行一个）</label>
  <textarea id="keys" rows="5" placeholder="test_1mb.bin&#10;video/stream.m3u8&#10;images/hero.png"></textarea></div>
  <div class="form-group"><label>目标节点 ID（留空 = 全部活跃节点）</label><input type="text" id="nodeId" placeholder="留空推送全部节点" /></div>
  <div class="btn-group"><button class="btn btn-primary" onclick="doPrewarm()" id="btnPrewarm">&#128640; 开始预热下发</button></div>
  <div id="result"></div>
  <div id="log" style="margin-top:14px;font-family:monospace;font-size:12px;color:var(--text4);max-height:300px;overflow-y:auto"></div>
</div>
<script>
async function doPrewarm(){
var keys=document.getElementById('keys').value.split('\n').map(function(k){return k.trim()}).filter(function(k){return k.length>0});
if(keys.length===0){return}
var btn=document.getElementById('btnPrewarm');btn.disabled=true;btn.innerHTML='<span class="spinner"></span> 下发中...';
document.getElementById('result').innerHTML='<div class="alert alert-info"><span class="spinner"></span> 正在推送到边缘节点...</div>';
try{
var r=await fetch('/api/prewarm',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({keys:keys,node_id:document.getElementById('nodeId').value})});
var d=await r.json();
var ok=(d.node_results||[]).filter(function(n){return n.status==='pushed'}).length;
var fail=(d.node_results||[]).length-ok;
document.getElementById('result').innerHTML='<div class="alert alert-success">&#10003; 推送完成：'+d.keys_total+' 个 Key → '+ok+' 个节点成功'+(fail>0?'，'+fail+' 失败':'')+'</div>';
var logHtml='';
(d.node_results||[]).forEach(function(n){logHtml+='<div style="padding:2px 0">['+n.node_id.substring(0,12)+'] '+(n.status==='pushed'?'&#9989;':'&#10060;')+' '+n.status+' '+(n.error||'')+'</div>'});
document.getElementById('log').innerHTML=logHtml;
}catch(e){document.getElementById('result').innerHTML='<div class="alert alert-error">&#10007; '+e.message+'</div>'}
btn.disabled=false;btn.innerHTML='&#128640; 开始预热下发';
}
</script>
`
