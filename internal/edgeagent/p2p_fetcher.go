package edgeagent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type PeerInfo struct {
	NodeID     string
	Endpoint   string
	LatencyMs  int64
	HasContent map[string]bool
	LastSeen   time.Time
}

type P2PFetcher struct {
	peers      map[string]*PeerInfo
	peersMu    sync.RWMutex
	client     *http.Client
	maxRetries int
}

func NewP2PFetcher() *P2PFetcher {
	return &P2PFetcher{
		peers: make(map[string]*PeerInfo),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     60 * time.Second,
			},
		},
		maxRetries: 2,
	}
}

func (p *P2PFetcher) FetchFromPeer(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	p.peersMu.RLock()
	peers := make([]*PeerInfo, 0, len(p.peers))
	for _, peer := range p.peers {
		if peer.HasContent[key] {
			peers = append([]*PeerInfo{peer}, peers...)
		} else {
			peers = append(peers, peer)
		}
	}
	p.peersMu.RUnlock()

	for _, peer := range peers {
		body, size, err := p.tryFetch(ctx, peer, key)
		if err == nil {
			slog.Info("p2p fetch success", "key", key, "peer", peer.NodeID, "size", size)
			return body, size, nil
		}
		slog.Warn("p2p fetch failed, trying next peer", "peer", peer.NodeID, "err", err)
	}
	return nil, 0, fmt.Errorf("p2p fetch failed from all peers")
}

func (p *P2PFetcher) tryFetch(ctx context.Context, peer *PeerInfo, key string) (io.ReadCloser, int64, error) {
	url := fmt.Sprintf("%s/internal/p2p/obj/%s", peer.Endpoint, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("peer returned %d", resp.StatusCode)
	}
	return resp.Body, resp.ContentLength, nil
}

func (p *P2PFetcher) UpdatePeers(peers []*PeerInfo) {
	p.peersMu.Lock()
	defer p.peersMu.Unlock()
	for _, peer := range peers {
		peer.LastSeen = time.Now()
		p.peers[peer.NodeID] = peer
	}
}

func (p *P2PFetcher) HasPeerWithContent(key string) bool {
	p.peersMu.RLock()
	defer p.peersMu.RUnlock()
	for _, peer := range p.peers {
		if peer.HasContent[key] {
			return true
		}
	}
	return false
}

func (p *P2PFetcher) PeerCount() int {
	p.peersMu.RLock()
	defer p.peersMu.RUnlock()
	return len(p.peers)
}
