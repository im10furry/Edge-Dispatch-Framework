package controlplane

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/im10furry/edge-dispatch-framework/internal/auth"
	"github.com/im10furry/edge-dispatch-framework/internal/config"
	"github.com/im10furry/edge-dispatch-framework/internal/models"
)

func BenchmarkScore(b *testing.B) {
	s := &Scheduler{cfg: &config.ControlPlaneConfig{}}
	node := models.Node{
		Region: "cn-bj",
		ISP:    "cmcc",
		Scores: models.NodeScores{
			ReachableScore: 85,
			HealthScore:    90,
			RiskScore:      5,
		},
	}
	client := models.ClientInfo{Region: "cn-bj", ISP: "cmcc"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.score(&node, client)
	}
}

func BenchmarkFilter(b *testing.B) {
	s := &Scheduler{cfg: &config.ControlPlaneConfig{}}
	nodes := make([]*models.Node, 100)
	for i := range nodes {
		nodes[i] = &models.Node{
			NodeID: fmt.Sprintf("n-%d", i),
			Capabilities: models.Capabilities{
				InboundReachable: i%3 != 0,
			},
			Scores: models.NodeScores{ReachableScore: float64(i % 20)},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.filter(nil, nodes)
	}
}

func BenchmarkFilterForSmallBandwidth(b *testing.B) {
	s := &Scheduler{cfg: &config.ControlPlaneConfig{
		SmallBandwidthOptimization: config.SmallBandwidthConfig{
			Enabled:                 true,
			SmallBandwidthThreshold: 50,
		},
	}}
	nodes := make([]*models.Node, 100)
	for i := range nodes {
		bw := int64(100)
		if i%5 == 0 {
			bw = 20
		}
		nodes[i] = &models.Node{
			NodeID: fmt.Sprintf("n-%d", i),
			Capabilities: models.Capabilities{
				MaxUplinkMbps: bw,
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.filterForSmallBandwidth(nodes, "test-key")
	}
}

func BenchmarkScoreKeyFast(b *testing.B) {
	s := &Scheduler{cfg: &config.ControlPlaneConfig{
		SmallBandwidthOptimization: config.SmallBandwidthConfig{
			Enabled:                 true,
			SmallBandwidthThreshold: 50,
		},
	}}
	node := &models.Node{
		NodeID: "n-1",
		Region: "cn-bj",
		ISP:    "cmcc",
		Scores: models.NodeScores{
			ReachableScore: 85,
			HealthScore:    90,
			RiskScore:      5,
		},
		Capabilities: models.Capabilities{
			InboundReachable: true,
			MaxUplinkMbps:   100,
		},
	}
	client := models.ClientInfo{Region: "cn-bj", ISP: "cmcc"}
	hotNodes := map[string]bool{"n-1": true}
	bloomNodes := map[string]bool{"n-2": true}
	streamNodes := map[string]bool{"n-3": true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.scoreKeyFast(node, client, "test-key", hotNodes, bloomNodes, streamNodes)
	}
}

type mockNodeCache struct {
	nodes []*models.Node
}

func (m *mockNodeCache) GetActiveNodes(ctx context.Context) ([]*models.Node, error) {
	return m.nodes, nil
}

func newMockNodeCache(nodes []*models.Node) *NodeCache {
	nc := &NodeCache{ttl: int64(time.Hour)}
	nc.nodes.Store(nodes)
	nc.cachedAt.Store(time.Now().Add(time.Hour).UnixNano())
	return nc
}

func BenchmarkResolve(b *testing.B) {
	nodes := make([]*models.Node, 50)
	for i := range nodes {
		nodes[i] = &models.Node{
			NodeID:   fmt.Sprintf("n-%d", i),
			Name:     fmt.Sprintf("edge-%d", i),
			Region:   []string{"cn-sh", "cn-bj", "cn-gd", "us-west", "eu-west"}[i%5],
			ISP:      []string{"ctcc", "cucc", "cmcc", "aws", "gcp"}[i%5],
			Status:   models.NodeStatusActive,
			Weight:   100,
			Endpoints: []models.Endpoint{{Scheme: "http", Host: fmt.Sprintf("10.0.0.%d", i), Port: 9090}},
			Capabilities: models.Capabilities{
				InboundReachable: true,
				MaxUplinkMbps:   100,
			},
			Scores: models.NodeScores{
				ReachableScore: 80,
				HealthScore:    90,
				RiskScore:      5,
			},
		}
	}

	signer := auth.NewSigner("bench-secret")
	cfg := &config.ControlPlaneConfig{
		TokenSecret:     "bench-secret",
		MaxCandidates:   5,
		DefaultTTLMs:    30000,
		DegradeToOrigin: true,
		OriginURL:       "http://origin:7070",
	}
	s := NewScheduler(newMockNodeCache(nodes), signer, cfg)

	req := models.DispatchRequest{
		Client: models.ClientInfo{
			IP:     "101.226.125.1",
			Region: "cn-sh",
			ISP:    "ctcc",
		},
		Resource: models.ResourceInfo{
			Type: "object",
			Key:  "test-file.bin",
		},
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.Resolve(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolveParallel(b *testing.B) {
	nodes := make([]*models.Node, 50)
	for i := range nodes {
		nodes[i] = &models.Node{
			NodeID:   fmt.Sprintf("n-%d", i),
			Name:     fmt.Sprintf("edge-%d", i),
			Region:   []string{"cn-sh", "cn-bj", "cn-gd", "us-west", "eu-west"}[i%5],
			ISP:      []string{"ctcc", "cucc", "cmcc", "aws", "gcp"}[i%5],
			Status:   models.NodeStatusActive,
			Weight:   100,
			Endpoints: []models.Endpoint{{Scheme: "http", Host: fmt.Sprintf("10.0.0.%d", i), Port: 9090}},
			Capabilities: models.Capabilities{
				InboundReachable: true,
				MaxUplinkMbps:   100,
			},
			Scores: models.NodeScores{
				ReachableScore: 80,
				HealthScore:    90,
				RiskScore:      5,
			},
		}
	}

	signer := auth.NewSigner("bench-secret")
	cfg := &config.ControlPlaneConfig{
		TokenSecret:     "bench-secret",
		MaxCandidates:   5,
		DefaultTTLMs:    30000,
		DegradeToOrigin: true,
		OriginURL:       "http://origin:7070",
	}
	s := NewScheduler(newMockNodeCache(nodes), signer, cfg)

	req := models.DispatchRequest{
		Client: models.ClientInfo{
			IP:     "101.226.125.1",
			Region: "cn-sh",
			ISP:    "ctcc",
		},
		Resource: models.ResourceInfo{
			Type: "object",
			Key:  "test-file.bin",
		},
	}

	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := s.Resolve(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
