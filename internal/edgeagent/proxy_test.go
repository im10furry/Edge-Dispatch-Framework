package edgeagent

import (
	"net/http"
	"testing"
)

func newTestProxyHandler(ws, grpc bool) *ProxyHandler {
	return NewProxyHandler("http://origin.example.com", "test-token", ws, grpc)
}

func TestIsWebSocket(t *testing.T) {
	tests := []struct {
		name       string
		wsEnabled  bool
		headers    map[string]string
		wantWS     bool
	}{
		{
			name:      "standard websocket upgrade",
			wsEnabled: true,
			headers: map[string]string{
				"Upgrade":    "websocket",
				"Connection": "Upgrade",
			},
			wantWS: true,
		},
		{
			name:      "upgrade case insensitive",
			wsEnabled: true,
			headers: map[string]string{
				"Upgrade":    "WebSocket",
				"Connection": "upgrade",
			},
			wantWS: true,
		},
		{
			name:      "connection with upgrade in list",
			wsEnabled: true,
			headers: map[string]string{
				"Upgrade":    "websocket",
				"Connection": "keep-alive, Upgrade",
			},
			wantWS: true,
		},
		{
			name:      "connection mixed case list",
			wsEnabled: true,
			headers: map[string]string{
				"Upgrade":    "websocket",
				"Connection": "keep-alive, UpGrAdE",
			},
			wantWS: true,
		},
		{
			name:      "missing upgrade header",
			wsEnabled: true,
			headers: map[string]string{
				"Connection": "Upgrade",
			},
			wantWS: false,
		},
		{
			name:      "missing connection header",
			wsEnabled: true,
			headers: map[string]string{
				"Upgrade": "websocket",
			},
			wantWS: false,
		},
		{
			name:      "wrong upgrade value",
			wsEnabled: true,
			headers: map[string]string{
				"Upgrade":    "h2c",
				"Connection": "Upgrade",
			},
			wantWS: false,
		},
		{
			name:      "connection without upgrade token",
			wsEnabled: true,
			headers: map[string]string{
				"Upgrade":    "websocket",
				"Connection": "keep-alive",
			},
			wantWS: false,
		},
		{
			name:      "ws disabled",
			wsEnabled: false,
			headers: map[string]string{
				"Upgrade":    "websocket",
				"Connection": "Upgrade",
			},
			wantWS: false,
		},
		{
			name:      "no headers at all",
			wsEnabled: true,
			headers:    map[string]string{},
			wantWS:     false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := newTestProxyHandler(tt.wsEnabled, false)
			req, _ := http.NewRequest(http.MethodGet, "http://example.com/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			got := p.IsWebSocket(req)
			if got != tt.wantWS {
				t.Errorf("IsWebSocket = %v, want %v", got, tt.wantWS)
			}
		})
	}
}

func TestIsGRPC(t *testing.T) {
	tests := []struct {
		name        string
		grpcEnabled bool
		contentType string
		wantGRPC    bool
	}{
		{
			name:        "application/grpc",
			grpcEnabled: true,
			contentType: "application/grpc",
			wantGRPC:    true,
		},
		{
			name:        "application/grpc+proto",
			grpcEnabled: true,
			contentType: "application/grpc+proto",
			wantGRPC:    true,
		},
		{
			name:        "application/grpc-web",
			grpcEnabled: true,
			contentType: "application/grpc-web",
			wantGRPC:    true,
		},
		{
			name:        "application/grpc-web+proto",
			grpcEnabled: true,
			contentType: "application/grpc-web+proto",
			wantGRPC:    true,
		},
		{
			name:        "application/json",
			grpcEnabled: true,
			contentType: "application/json",
			wantGRPC:    false,
		},
		{
			name:        "text/plain",
			grpcEnabled: true,
			contentType: "text/plain",
			wantGRPC:    false,
		},
		{
			name:        "empty content type",
			grpcEnabled: true,
			contentType: "",
			wantGRPC:    false,
		},
		{
			name:        "grpc substring not prefix",
			grpcEnabled: true,
			contentType: "x-application/grpc",
			wantGRPC:    false,
		},
		{
			name:        "grpc disabled",
			grpcEnabled: false,
			contentType: "application/grpc",
			wantGRPC:    false,
		},
		{
			name:        "grpc with charset matches prefix",
			grpcEnabled: true,
			contentType: "application/grpc; charset=utf-8",
			wantGRPC:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := newTestProxyHandler(false, tt.grpcEnabled)
			req, _ := http.NewRequest(http.MethodPost, "http://example.com/", nil)
			req.Header.Set("Content-Type", tt.contentType)
			got := p.IsGRPC(req)
			if got != tt.wantGRPC {
				t.Errorf("IsGRPC = %v, want %v", got, tt.wantGRPC)
			}
		})
	}
}

func TestCopyHeaders(t *testing.T) {
	tests := []struct {
		name     string
		src      map[string][]string
		wantKeys []string
		skipKeys []string
	}{
		{
			name: "copies allowed headers",
			src: map[string][]string{
				"Content-Type":   {"application/json"},
				"X-Custom":       {"value1"},
				"Authorization":  {"Bearer token"},
			},
			wantKeys: []string{"Content-Type", "X-Custom", "Authorization"},
		},
		{
			name: "skips connection header",
			src: map[string][]string{
				"Content-Type": {"text/plain"},
				"Connection":   {"keep-alive"},
			},
			wantKeys: []string{"Content-Type"},
			skipKeys: []string{"Connection"},
		},
		{
			name: "skips keep-alive header",
			src: map[string][]string{
				"Content-Type": {"text/plain"},
				"Keep-Alive":   {"timeout=120"},
			},
			wantKeys: []string{"Content-Type"},
			skipKeys: []string{"Keep-Alive"},
		},
		{
			name: "skips proxy-connection header",
			src: map[string][]string{
				"Content-Type":     {"text/plain"},
				"Proxy-Connection": {"keep-alive"},
			},
			wantKeys: []string{"Content-Type"},
			skipKeys: []string{"Proxy-Connection"},
		},
		{
			name: "skips transfer-encoding header",
			src: map[string][]string{
				"Content-Type":      {"text/plain"},
				"Transfer-Encoding": {"chunked"},
			},
			wantKeys: []string{"Content-Type"},
			skipKeys: []string{"Transfer-Encoding"},
		},
		{
			name: "skips upgrade header",
			src: map[string][]string{
				"Content-Type": {"text/plain"},
				"Upgrade":      {"websocket"},
			},
			wantKeys: []string{"Content-Type"},
			skipKeys: []string{"Upgrade"},
		},
		{
			name: "skips hop-by-hop case insensitive",
			src: map[string][]string{
				"Content-Type": {"text/plain"},
				"connection":   {"close"},
				"UPGRADE":      {"h2c"},
			},
			wantKeys: []string{"Content-Type"},
			skipKeys: []string{"connection", "UPGRADE"},
		},
		{
			name: "multiple values for same key",
			src: map[string][]string{
				"Accept": {"text/html", "application/json"},
			},
			wantKeys: []string{"Accept"},
		},
		{
			name:     "empty source",
			src:      map[string][]string{},
			wantKeys: []string{},
		},
		{
			name: "all hop-by-hop removed",
			src: map[string][]string{
				"Connection":       {"keep-alive"},
				"Keep-Alive":       {"timeout=120"},
				"Proxy-Connection": {"close"},
				"Transfer-Encoding": {"chunked"},
				"Upgrade":          {"websocket"},
			},
			wantKeys: []string{},
			skipKeys: []string{"Connection", "Keep-Alive", "Proxy-Connection", "Transfer-Encoding", "Upgrade"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := newTestProxyHandler(false, false)
			src := make(http.Header)
			for k, vv := range tt.src {
				for _, v := range vv {
					src.Add(k, v)
				}
			}
			dst := make(http.Header)
			p.copyHeaders(dst, src)

			for _, k := range tt.wantKeys {
				srcVals := src.Values(k)
				dstVals := dst.Values(k)
				if len(dstVals) != len(srcVals) {
					t.Errorf("header %q: got %d values, want %d", k, len(dstVals), len(srcVals))
					continue
				}
				for i, v := range srcVals {
					if dstVals[i] != v {
						t.Errorf("header %q[%d] = %q, want %q", k, i, dstVals[i], v)
					}
				}
			}

			for _, k := range tt.skipKeys {
				if dst.Get(k) != "" {
					t.Errorf("header %q should be removed, got %q", k, dst.Get(k))
				}
			}
		})
	}
}
