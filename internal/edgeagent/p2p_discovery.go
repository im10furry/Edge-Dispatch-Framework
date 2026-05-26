package edgeagent

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	p2pChannel      = "edf:p2p:announce"
	p2pAnnounceTTL  = 90 * time.Second
)

// P2PDiscovery manages peer discovery via Redis Pub/Sub.
type P2PDiscovery struct {
	nodeID      string
	endpoint    string
	isShield    bool
	fetcher     *P2PFetcher
	client      *redis.Client
	interval    time.Duration
	mu          sync.Mutex
	knownPeers  map[string]*PeerInfo
	ctx         context.Context
	cancel      context.CancelFunc
}

// P2PAnnounce is the message published when a node announces itself.
type P2PAnnounce struct {
	NodeID   string `json:"node_id"`
	Endpoint string `json:"endpoint"`
	IsShield bool   `json:"is_shield"`
	TS       int64  `json:"ts"`
}

// NewP2PDiscovery creates a new P2P discovery service.
func NewP2PDiscovery(client *redis.Client, fetcher *P2PFetcher, nodeID, endpoint string, isShield bool, interval time.Duration) *P2PDiscovery {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &P2PDiscovery{
		nodeID:     nodeID,
		endpoint:   endpoint,
		isShield:   isShield,
		fetcher:    fetcher,
		client:     client,
		interval:   interval,
		knownPeers: make(map[string]*PeerInfo),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins the P2P discovery loop (publish + subscribe).
func (d *P2PDiscovery) Start() {
	go d.publishLoop()
	go d.subscribeLoop()
	go d.cleanupLoop()
	slog.Info("p2p discovery started", "node_id", d.nodeID, "interval", d.interval)
}

// Stop stops the P2P discovery service.
func (d *P2PDiscovery) Stop() {
	d.cancel()
}

func (d *P2PDiscovery) publishLoop() {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	d.publish()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.publish()
		}
	}
}

func (d *P2PDiscovery) publish() {
	announce := P2PAnnounce{
		NodeID:   d.nodeID,
		Endpoint: d.endpoint,
		IsShield: d.isShield,
		TS:       time.Now().Unix(),
	}
	data, err := json.Marshal(announce)
	if err != nil {
		slog.Warn("p2p: failed to marshal announce", "err", err)
		return
	}
	if err := d.client.Publish(d.ctx, p2pChannel, data).Err(); err != nil {
		slog.Warn("p2p: failed to publish announce", "err", err)
	}
}

func (d *P2PDiscovery) subscribeLoop() {
	sub := d.client.Subscribe(d.ctx, p2pChannel)
	ch := sub.Channel()

	for {
		select {
		case <-d.ctx.Done():
			sub.Close()
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			d.handleAnnounce(msg.Payload)
		}
	}
}

func (d *P2PDiscovery) handleAnnounce(payload string) {
	var announce P2PAnnounce
	if err := json.Unmarshal([]byte(payload), &announce); err != nil {
		slog.Warn("p2p: failed to unmarshal announce", "err", err)
		return
	}

	if announce.NodeID == d.nodeID {
		return
	}

	peer := &PeerInfo{
		NodeID:   announce.NodeID,
		Endpoint: announce.Endpoint,
		IsShield: announce.IsShield,
		LastSeen: time.Now(),
	}

	d.mu.Lock()
	d.knownPeers[announce.NodeID] = peer
	d.mu.Unlock()

	d.fetcher.UpdatePeers([]*PeerInfo{peer})

	slog.Debug("p2p: discovered peer",
		"peer_id", announce.NodeID,
		"endpoint", announce.Endpoint,
		"is_shield", announce.IsShield,
	)
}

func (d *P2PDiscovery) cleanupLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.cleanup()
		}
	}
}

func (d *P2PDiscovery) cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	cutoff := time.Now().Add(-p2pAnnounceTTL)
	for id, peer := range d.knownPeers {
		if peer.LastSeen.Before(cutoff) {
			delete(d.knownPeers, id)
			slog.Info("p2p: removed stale peer", "peer_id", id)
		}
	}
}

// GetPeers returns the current list of known peers.
func (d *P2PDiscovery) GetPeers() []*PeerInfo {
	d.mu.Lock()
	defer d.mu.Unlock()

	peers := make([]*PeerInfo, 0, len(d.knownPeers))
	for _, p := range d.knownPeers {
		peers = append(peers, p)
	}
	return peers
}
