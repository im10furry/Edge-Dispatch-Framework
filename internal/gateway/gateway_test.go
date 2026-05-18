package gateway

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/darkinno/edge-dispatch-framework/internal/metrics"
	"github.com/darkinno/edge-dispatch-framework/internal/tunnel"
)

func TestSanitizeGatewayKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "normal path", key: "foo/bar/baz", want: "foo/bar/baz"},
		{name: "backslash to slash", key: `foo\bar\baz`, want: "foo/bar/baz"},
		{name: "mixed slashes", key: `foo/bar\baz`, want: "foo/bar/baz"},
		{name: "empty segments", key: "foo//bar///baz", want: "foo/bar/baz"},
		{name: "dot segment", key: "foo/./bar", want: "foo/bar"},
		{name: "double dot segment", key: "foo/../bar", want: "foo/bar"},
		{name: "leading trailing whitespace", key: "  foo/bar  ", want: "foo/bar"},
		{name: "whitespace in segments", key: "foo / bar", want: "foo/bar"},
		{name: "single dot only", key: ".", want: ""},
		{name: "double dot only", key: "..", want: ""},
		{name: "empty string", key: "", want: ""},
		{name: "whitespace only", key: "   ", want: ""},
		{name: "complex mixed", key: `foo\..\bar\.\\baz`, want: "foo/bar/baz"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeGatewayKey(tt.key)
			if got != tt.want {
				t.Errorf("sanitizeGatewayKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestExtractClientIP(t *testing.T) {
	t.Parallel()

	t.Run("X-Forwarded-For single", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Forwarded-For", "203.0.113.1")
		got := extractClientIP(r)
		if got != "203.0.113.1" {
			t.Errorf("got %q, want 203.0.113.1", got)
		}
	})

	t.Run("X-Forwarded-For multiple with spaces", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1, 192.0.2.1")
		got := extractClientIP(r)
		if got != "203.0.113.1" {
			t.Errorf("got %q, want 203.0.113.1", got)
		}
	})

	t.Run("X-Forwarded-For no spaces", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Forwarded-For", "203.0.113.1,198.51.100.1")
		got := extractClientIP(r)
		if got != "203.0.113.1" {
			t.Errorf("got %q, want 203.0.113.1", got)
		}
	})

	t.Run("X-Real-IP", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Real-IP", "198.51.100.1")
		got := extractClientIP(r)
		if got != "198.51.100.1" {
			t.Errorf("got %q, want 198.51.100.1", got)
		}
	})

	t.Run("RemoteAddr with port", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "192.168.1.1:54321"
		got := extractClientIP(r)
		if got != "192.168.1.1" {
			t.Errorf("got %q, want 192.168.1.1", got)
		}
	})

	t.Run("RemoteAddr without port", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "10.0.0.1"
		got := extractClientIP(r)
		if got != "10.0.0.1" {
			t.Errorf("got %q, want 10.0.0.1", got)
		}
	})

	t.Run("priority XFF over X-Real-IP", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Forwarded-For", "203.0.113.1")
		r.Header.Set("X-Real-IP", "198.51.100.1")
		r.RemoteAddr = "192.168.1.1:1234"
		got := extractClientIP(r)
		if got != "203.0.113.1" {
			t.Errorf("got %q, want 203.0.113.1 (XFF takes priority)", got)
		}
	})

	t.Run("priority X-Real-IP over RemoteAddr", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Real-IP", "198.51.100.1")
		r.RemoteAddr = "192.168.1.1:1234"
		got := extractClientIP(r)
		if got != "198.51.100.1" {
			t.Errorf("got %q, want 198.51.100.1", got)
		}
	})

	t.Run("IPv6 RemoteAddr", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "[::1]:54321"
		got := extractClientIP(r)
		if got != "::1" {
			t.Errorf("got %q, want ::1", got)
		}
	})
}

func TestStripHopByHopHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  map[string]string
		checks map[string]bool // true = should exist, false = should not exist
	}{
		{
			name: "all hop-by-hop headers removed",
			input: map[string]string{
				"Connection":          "keep-alive",
				"Keep-Alive":          "timeout=5",
				"Proxy-Authenticate":  "Basic",
				"Proxy-Authorization": "Bearer x",
				"TE":                  "trailers",
				"Trailer":             "X-Custom",
				"Transfer-Encoding":   "chunked",
				"Upgrade":             "websocket",
				"Content-Type":        "application/json",
				"X-Custom":            "value",
			},
			checks: map[string]bool{
				"Connection":          false,
				"Keep-Alive":          false,
				"Proxy-Authenticate":  false,
				"Proxy-Authorization": false,
				"TE":                  false,
				"Trailer":             false,
				"Transfer-Encoding":   false,
				"Upgrade":             false,
				"Content-Type":        true,
				"X-Custom":            true,
			},
		},
		{
			name: "no hop-by-hop headers",
			input: map[string]string{
				"Content-Type": "text/html",
				"X-Request-ID": "abc123",
			},
			checks: map[string]bool{
				"Content-Type": true,
				"X-Request-ID": true,
			},
		},
		{
			name:   "empty map",
			input:  map[string]string{},
			checks: map[string]bool{},
		},
		{
			name: "only hop-by-hop headers",
			input: map[string]string{
				"Connection": "close",
				"Upgrade":    "h2c",
			},
			checks: map[string]bool{
				"Connection": false,
				"Upgrade":    false,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stripHopByHopHeaders(tt.input)
			for k, shouldExist := range tt.checks {
				_, exists := tt.input[k]
				if shouldExist && !exists {
					t.Errorf("header %q should exist but was removed", k)
				}
				if !shouldExist && exists {
					t.Errorf("header %q should have been removed but still exists", k)
				}
			}
		})
	}
}

func TestGatewaySetTokenQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rawQuery  string
		token     string
		wantQuery string
	}{
		{name: "empty query", rawQuery: "", token: "abc123", wantQuery: "token=abc123"},
		{name: "existing other params", rawQuery: "a=1&b=2", token: "xyz", wantQuery: "a=1&b=2&token=xyz"},
		{name: "replaces existing token", rawQuery: "a=1&token=old", token: "new", wantQuery: "a=1&token=new"},
		{name: "token with special chars", rawQuery: "a=1", token: "new value", wantQuery: "a=1&token=new+value"},
		{name: "malformed query", rawQuery: "%gh%ij", token: "t", wantQuery: "token=t"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := setTokenQuery(tt.rawQuery, tt.token)
			if got != tt.wantQuery {
				t.Errorf("setTokenQuery(%q, %q) = %q, want %q", tt.rawQuery, tt.token, got, tt.wantQuery)
			}
		})
	}
}

func TestWithRecovery(t *testing.T) {
	t.Parallel()

	t.Run("no panic", func(t *testing.T) {
		t.Parallel()
		handler := withRecovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}))
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Body.String() != "ok" {
			t.Errorf("body = %q, want ok", rec.Body.String())
		}
	})

	t.Run("panic recovered", func(t *testing.T) {
		t.Parallel()
		handler := withRecovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}))
		req := httptest.NewRequest("GET", "/panic", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
		if !strings.Contains(rec.Body.String(), "internal server error") {
			t.Errorf("body = %q, want 'internal server error'", rec.Body.String())
		}
	})
}

func TestWithSecurityHeaders(t *testing.T) {
	t.Parallel()

	handler := withSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", got)
	}
}

func TestHandleHealth(t *testing.T) {
	t.Parallel()

	ts := tunnel.NewServer(tunnel.DefaultServerConfig(), nil)
	defer ts.Stop()

	gw := &Gateway{
		tunnelServer: ts,
	}

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	gw.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("body = %q, want to contain 'status':'ok'", body)
	}
	if !strings.Contains(body, `"tunnels":0`) {
		t.Errorf("body = %q, want to contain 'tunnels':0", body)
	}
}

func TestHandleMetrics(t *testing.T) {
	t.Parallel()

	t.Run("json format with tunnelServer", func(t *testing.T) {
		t.Parallel()
		ts := tunnel.NewServer(tunnel.DefaultServerConfig(), nil)
		defer ts.Stop()

		tunnelGauge := metrics.NewGauge("gateway_active_tunnels", "active tunnels")
		gw := &Gateway{
			tunnelServer: ts,
			promMetrics: &gwMetrics{
				tunnelGauge: tunnelGauge,
			},
		}

		req := httptest.NewRequest("GET", "/metrics?format=json", nil)
		rec := httptest.NewRecorder()
		gw.handleMetrics(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		body := rec.Body.String()
		if body != `{"active_tunnels":0}` {
			t.Errorf("body = %q, want {'active_tunnels':0}", body)
		}
	})

	t.Run("json format without tunnelServer", func(t *testing.T) {
		t.Parallel()
		gw := &Gateway{
			tunnelServer: nil,
		}

		req := httptest.NewRequest("GET", "/metrics?format=json", nil)
		rec := httptest.NewRecorder()
		gw.handleMetrics(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		body := rec.Body.String()
		if body != `{"active_tunnels":0}` {
			t.Errorf("body = %q, want {'active_tunnels':0}", body)
		}
	})

	t.Run("json format with tunnelServer but nil promMetrics", func(t *testing.T) {
		t.Parallel()
		ts := tunnel.NewServer(tunnel.DefaultServerConfig(), nil)
		defer ts.Stop()

		gw := &Gateway{
			tunnelServer: ts,
			promMetrics:  nil,
		}

		req := httptest.NewRequest("GET", "/metrics?format=json", nil)
		rec := httptest.NewRecorder()
		gw.handleMetrics(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if body != `{"active_tunnels":0}` {
			t.Errorf("body = %q, want {'active_tunnels':0}", body)
		}
	})

	t.Run("prometheus format", func(t *testing.T) {
		t.Parallel()
		ts := tunnel.NewServer(tunnel.DefaultServerConfig(), nil)
		defer ts.Stop()

		tunnelGauge := metrics.NewGauge("gateway_active_tunnels", "active tunnels")
		gw := &Gateway{
			tunnelServer: ts,
			promMetrics: &gwMetrics{
				tunnelGauge: tunnelGauge,
			},
		}

		req := httptest.NewRequest("GET", "/metrics", nil)
		rec := httptest.NewRecorder()
		gw.handleMetrics(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		ct := rec.Header().Get("Content-Type")
		if !strings.HasPrefix(ct, "text/plain") {
			t.Errorf("Content-Type = %q, want text/plain prefix", ct)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "gateway_active_tunnels") {
			t.Errorf("body = %q, want to contain gateway_active_tunnels", body)
		}
	})

	t.Run("prometheus format with nil tunnelServer", func(t *testing.T) {
		t.Parallel()
		gw := &Gateway{
			tunnelServer: nil,
		}

		req := httptest.NewRequest("GET", "/metrics", nil)
		rec := httptest.NewRecorder()
		gw.handleMetrics(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestExtractClientIP_EmptyHeaders(t *testing.T) {
	t.Parallel()

	t.Run("no headers falls back to RemoteAddr", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		got := extractClientIP(r)
		if got != host {
			t.Errorf("got %q, want %q (RemoteAddr host)", got, host)
		}
	})

	t.Run("X-Forwarded-For with trailing comma", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Forwarded-For", "203.0.113.1,")
		got := extractClientIP(r)
		if got != "203.0.113.1" {
			t.Errorf("got %q, want 203.0.113.1", got)
		}
	})
}
