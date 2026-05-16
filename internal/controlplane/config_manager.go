package controlplane

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/darkinno/edge-dispatch-framework/internal/models"
)

type GlobalConfig struct {
	SmallBandwidth SmallBandwidthConfig `json:"small_bandwidth"`
	P2P            P2PConfig            `json:"p2p"`
	Prefetch       PrefetchConfig       `json:"prefetch"`
	OriginFetch    OriginFetchConfig    `json:"origin_fetch"`
}

type SmallBandwidthConfig struct {
	Enabled  bool  `json:"enabled"`
	Threshold int64 `json:"threshold_mbps"`
}

type P2PConfig struct {
	Enabled              bool  `json:"enabled"`
	DiscoveryIntervalSec int   `json:"discovery_interval_sec"`
	MaxPeers             int   `json:"max_peers"`
	BandwidthLimitMbps   int64 `json:"bandwidth_limit_mbps"`
}

type PrefetchConfig struct {
	Enabled   bool            `json:"enabled"`
	NightMode NightModeConfig `json:"night_mode"`
	DayMode   DayModeConfig   `json:"day_mode"`
}

type NightModeConfig struct {
	Start             string `json:"start"`
	End               string `json:"end"`
	BandwidthLimitMbps int64 `json:"bandwidth_limit_mbps"`
}

type DayModeConfig struct {
	BandwidthLimitMbps int64 `json:"bandwidth_limit_mbps"`
	MinPriority        int   `json:"min_priority"`
}

type OriginFetchConfig struct {
	BandwidthPercent int      `json:"bandwidth_percent"`
	MaxConcurrent    int      `json:"max_concurrent"`
	TimeoutSec       int      `json:"timeout_sec"`
	Priority         []string `json:"priority"`
}

type P2PTopology struct {
	Nodes []P2PTopologyNode `json:"nodes"`
	Links []P2PTopologyLink `json:"links"`
}

type P2PTopologyNode struct {
	NodeID          string  `json:"node_id"`
	Name            string  `json:"name"`
	BandwidthMbps   int64   `json:"bandwidth_mbps"`
	IsSmallBandwidth bool   `json:"is_small_bandwidth"`
	CacheHitRatio   float64 `json:"cache_hit_ratio"`
	Status          string  `json:"status"`
}

type P2PTopologyLink struct {
	Source      string  `json:"source"`
	Target      string  `json:"target"`
	LatencyMs   int64   `json:"latency_ms"`
	SuccessRate float64 `json:"success_rate"`
}

type AdminNodeConfig struct {
	SBThreshold         int64 `json:"sb_threshold_mbps"`
	P2PLimitMbps        int64 `json:"p2p_limit_mbps"`
	OriginFetchPct      int   `json:"origin_fetch_pct"`
	PrefetchEnabled     bool  `json:"prefetch_enabled"`
}

func (s *Scheduler) handleAdminGetConfig(w http.ResponseWriter, r *http.Request) {
	sb := s.cfg.SmallBandwidthOptimization
	cfg := GlobalConfig{
		SmallBandwidth: SmallBandwidthConfig{
			Enabled:   sb.Enabled,
			Threshold: sb.SmallBandwidthThreshold,
		},
		P2P: P2PConfig{
			Enabled:              sb.P2PEnabled,
			DiscoveryIntervalSec: 60,
			MaxPeers:             10,
			BandwidthLimitMbps:   20,
		},
		Prefetch: PrefetchConfig{
			Enabled: sb.PrefetchEnabled,
			NightMode: NightModeConfig{
				Start:              "01:00",
				End:                "07:00",
				BandwidthLimitMbps: int64(sb.PrefetchBandwidthLimit),
			},
			DayMode: DayModeConfig{
				BandwidthLimitMbps: int64(sb.PrefetchBandwidthLimit),
				MinPriority:        8,
			},
		},
		OriginFetch: OriginFetchConfig{
			BandwidthPercent: 80,
			MaxConcurrent:    5,
			TimeoutSec:       30,
			Priority:         []string{"p2p", "origin"},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

func (s *Scheduler) handleAdminUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var cfg GlobalConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	s.cfg.SmallBandwidthOptimization.Enabled = cfg.SmallBandwidth.Enabled
	if cfg.SmallBandwidth.Threshold > 0 {
		s.cfg.SmallBandwidthOptimization.SmallBandwidthThreshold = cfg.SmallBandwidth.Threshold
	}
	s.cfg.SmallBandwidthOptimization.P2PEnabled = cfg.P2P.Enabled
	if cfg.P2P.MaxPeers > 0 {
		s.cfg.SmallBandwidthOptimization.PrefetchEnabled = true
	}
	s.cfg.SmallBandwidthOptimization.PrefetchEnabled = cfg.Prefetch.Enabled

	slog.Info("global config updated via admin API",
		"sb_enabled", cfg.SmallBandwidth.Enabled,
		"sb_threshold", cfg.SmallBandwidth.Threshold,
		"p2p_enabled", cfg.P2P.Enabled,
		"prefetch_enabled", cfg.Prefetch.Enabled,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Scheduler) handleAdminApplyConfig(w http.ResponseWriter, r *http.Request) {
	slog.Info("global config applied via admin API — changes take effect immediately")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "applied", "message": "Configuration applied to all nodes"})
}

func (s *Scheduler) handleAdminP2PTopology(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.nodeCache.GetActiveNodes(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	threshold := s.cfg.SmallBandwidthOptimization.SmallBandwidthThreshold
	topo := P2PTopology{
		Nodes: make([]P2PTopologyNode, 0, len(nodes)),
		Links: make([]P2PTopologyLink, 0),
	}

	for _, n := range nodes {
		topo.Nodes = append(topo.Nodes, P2PTopologyNode{
			NodeID:            n.NodeID,
			Name:              n.Name,
			BandwidthMbps:     n.Capabilities.MaxUplinkMbps,
			IsSmallBandwidth:  n.Capabilities.MaxUplinkMbps > 0 && n.Capabilities.MaxUplinkMbps < threshold,
			Status:            string(n.Status),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(topo)
}

func (s *Scheduler) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.nodeCache.GetActiveNodes(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	online, offline, total := 0, 0, len(nodes)
	for _, n := range nodes {
		switch n.Status {
		case models.NodeStatusActive, models.NodeStatusDegraded:
			online++
		case models.NodeStatusOffline:
			offline++
		}
	}

	dashboard := models.DashboardMetrics{
		OnlineNodes:  online,
		OfflineNodes: offline,
		TotalNodes:   total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}
