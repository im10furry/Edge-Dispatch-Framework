package controlplane

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/im10furry/edge-dispatch-framework/internal/models"
	"github.com/im10furry/edge-dispatch-framework/internal/store"
)

type policyData struct {
	blockedIPs   map[string]bool
	blockedNodes map[string]bool
}

type Policy struct {
	data atomic.Pointer[policyData]
}

func NewPolicy() *Policy {
	p := &Policy{}
	p.data.Store(&policyData{
		blockedIPs:   make(map[string]bool),
		blockedNodes: make(map[string]bool),
	})
	return p
}

func (p *Policy) IsBlocked(ip string) bool {
	return p.data.Load().blockedIPs[ip]
}

func (p *Policy) BlockIP(ip string) {
	old := p.data.Load()
	newData := &policyData{
		blockedIPs:   copyMap(old.blockedIPs),
		blockedNodes: old.blockedNodes,
	}
	newData.blockedIPs[ip] = true
	p.data.Store(newData)
}

func (p *Policy) UnblockIP(ip string) {
	old := p.data.Load()
	newData := &policyData{
		blockedIPs:   copyMap(old.blockedIPs),
		blockedNodes: old.blockedNodes,
	}
	delete(newData.blockedIPs, ip)
	p.data.Store(newData)
}

func (p *Policy) IsNodeBlocked(nodeID string) bool {
	return p.data.Load().blockedNodes[nodeID]
}

func (p *Policy) BlockNode(nodeID string) {
	old := p.data.Load()
	newData := &policyData{
		blockedIPs:   old.blockedIPs,
		blockedNodes: copyMap(old.blockedNodes),
	}
	newData.blockedNodes[nodeID] = true
	p.data.Store(newData)
}

func (p *Policy) UnblockNode(nodeID string) {
	old := p.data.Load()
	newData := &policyData{
		blockedIPs:   old.blockedIPs,
		blockedNodes: copyMap(old.blockedNodes),
	}
	delete(newData.blockedNodes, nodeID)
	p.data.Store(newData)
}

type blockPolicyContent struct {
	BlockedIPs   []string `json:"blocked_ips"`
	BlockedNodes []string `json:"blocked_nodes"`
}

// SyncFromDB loads published block policies from the database and updates the in-memory state.
func (p *Policy) SyncFromDB(ctx context.Context, pg *store.PGStore) {
	policies, err := pg.ListAdminPolicies(ctx, "", "")
	if err != nil {
		slog.Warn("policy: failed to load policies from DB", "err", err)
		return
	}

	blockedIPs := make(map[string]bool)
	blockedNodes := make(map[string]bool)

	for _, pol := range policies {
		if !pol.IsPublished || pol.Type != models.AdminPolicyTypeBlock {
			continue
		}
		var content blockPolicyContent
		if err := json.Unmarshal(pol.Content, &content); err != nil {
			slog.Warn("policy: failed to parse block policy content", "policy_id", pol.PolicyID, "err", err)
			continue
		}
		for _, ip := range content.BlockedIPs {
			blockedIPs[ip] = true
		}
		for _, nodeID := range content.BlockedNodes {
			blockedNodes[nodeID] = true
		}
	}

	p.data.Store(&policyData{
		blockedIPs:   blockedIPs,
		blockedNodes: blockedNodes,
	})

	slog.Info("policy: synced from DB", "blocked_ips", len(blockedIPs), "blocked_nodes", len(blockedNodes))
}

// StartPolicySyncLoop periodically syncs policies from DB.
func StartPolicySyncLoop(ctx context.Context, policy *Policy, pg *store.PGStore, interval time.Duration) {
	policy.SyncFromDB(ctx, pg)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			policy.SyncFromDB(ctx, pg)
		}
	}
}

func copyMap(src map[string]bool) map[string]bool {
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
