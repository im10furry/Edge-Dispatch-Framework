package dns

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/im10furry/edge-dispatch-framework/internal/config"
	"github.com/im10furry/edge-dispatch-framework/internal/models"
)

func TestDispatchResolveSendsAuthToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer dns-secret" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(models.DispatchResponse{}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer ts.Close()

	s := NewServer(&config.DNSAdapterConfig{
		ControlPlaneURL: ts.URL,
		TokenSecret:     "dns-secret",
	})

	if _, err := s.dispatchResolve(context.Background(), models.DispatchRequest{}); err != nil {
		t.Fatalf("dispatchResolve: %v", err)
	}
}

func TestExtractIPHandlesIPv6Endpoint(t *testing.T) {
	got := extractIP("http://[2001:db8::1]:9090")
	want := net.ParseIP("2001:db8::1")
	if !got.Equal(want) {
		t.Fatalf("extractIP() = %v, want %v", got, want)
	}
}
