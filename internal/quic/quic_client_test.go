//go:build quic

package quic

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tlsCfg := generateTestTLSConfig(t)
	c := NewClient(ClientConfig{TLSConfig: tlsCfg})
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.rt == nil {
		t.Fatal("client RoundTripper is nil")
	}
	c.Close()
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient(ClientConfig{})
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.rt == nil {
		t.Fatal("client RoundTripper is nil without TLS config")
	}
	c.Close()
}

func TestNewHTTPClient(t *testing.T) {
	tlsCfg := generateTestTLSConfig(t)
	c := NewHTTPClient(ClientConfig{TLSConfig: tlsCfg})
	if c == nil {
		t.Fatal("NewHTTPClient returned nil")
	}
	if c.Transport == nil {
		t.Fatal("http.Client Transport is nil")
	}
	if c.Timeout != 30*time.Second {
		t.Errorf("default timeout = %v, want 30s", c.Timeout)
	}
}

func TestDefaultHTTPClient(t *testing.T) {
	c := DefaultHTTPClient()
	if c == nil {
		t.Fatal("DefaultHTTPClient returned nil")
	}
	if c.Transport == nil {
		t.Fatal("DefaultHTTPClient Transport is nil")
	}
}

func TestQUICClientRoundtrip(t *testing.T) {
	tlsCfg := generateTestTLSConfig(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("world"))
	})

	srv := NewServer(ServerConfig{
		Addr:      "127.0.0.1:0",
		Handler:   mux,
		TLSConfig: tlsCfg,
	})
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}

	go func() {
		srv.ListenAndServe()
	}()
	time.Sleep(100 * time.Millisecond)
	defer srv.Close()

	client := NewHTTPClient(ClientConfig{
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	})
	defer client.CloseIdleConnections()

	addr := srv.Addr()
	resp, err := client.Get("https://" + addr + "/hello")
	if err != nil {
		t.Fatalf("HTTP/3 GET via client: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	buf := make([]byte, 5)
	n, _ := resp.Body.Read(buf)
	if string(buf[:n]) != "world" {
		t.Errorf("body = %q, want world", string(buf[:n]))
	}
}

func TestDefaultHTTPClientRoundtrip(t *testing.T) {
	tlsCfg := generateTestTLSConfig(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := NewServer(ServerConfig{
		Addr:      "127.0.0.1:0",
		Handler:   mux,
		TLSConfig: tlsCfg,
	})

	go func() {
		srv.ListenAndServe()
	}()
	time.Sleep(100 * time.Millisecond)
	defer srv.Close()

	client := DefaultHTTPClient()
	addr := srv.Addr()

	resp, err := client.Get("https://" + addr + "/status")
	if err != nil {
		t.Fatalf("DefaultHTTPClient roundtrip: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want 204", resp.StatusCode)
	}
}

func TestClientRoundTripError(t *testing.T) {
	client := NewClient(ClientConfig{
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	})
	defer client.Close()

	req, _ := http.NewRequest("GET", "https://127.0.0.1:99999/none", nil)
	_, err := client.RoundTrip(req)
	if err == nil {
		t.Error("expected error for unreachable address")
	}
}

func TestClientClose(t *testing.T) {
	c := NewClient(ClientConfig{})
	if err := c.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	// Double close should be idempotent
	if err := c.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestClientConfigCustomTimeout(t *testing.T) {
	c := NewClient(ClientConfig{
		HandshakeTimeout: 5 * time.Second,
		MaxIdleTimeout:   60 * time.Second,
		KeepAlivePeriod:  10 * time.Second,
	})
	if c == nil || c.rt == nil {
		t.Fatal("NewClient returned nil with custom config")
	}
	c.Close()
}
