//go:build !quic

package quic

import (
	"net/http"
	"testing"
)

func TestNewClientStub(t *testing.T) {
	c := NewClient(ClientConfig{})
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if err := c.Close(); err != nil {
		t.Errorf("Close should not error: %v", err)
	}
}

func TestClientRoundTripStub(t *testing.T) {
	c := NewClient(ClientConfig{})
	defer c.Close()

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	_, err := c.RoundTrip(req)
	if err == nil {
		t.Error("expected error from stub RoundTrip")
	}
}

func TestNewHTTPClientStub(t *testing.T) {
	hc := NewHTTPClient(ClientConfig{})
	if hc != nil {
		t.Error("NewHTTPClient should return nil when QUIC not enabled")
	}
}

func TestDefaultHTTPClientStub(t *testing.T) {
	hc := DefaultHTTPClient()
	if hc != nil {
		t.Error("DefaultHTTPClient should return nil when QUIC not enabled")
	}
}

func TestDefaultListenAddr(t *testing.T) {
	if DefaultListenAddr != ":9443" {
		t.Errorf("DefaultListenAddr = %q, want :9443", DefaultListenAddr)
	}
}
