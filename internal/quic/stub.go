//go:build !quic

package quic

import (
	"crypto/tls"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

func init() {
	Enabled = false
}

// ServerConfig holds QUIC server configuration.
type ServerConfig struct {
	Addr      string
	Handler   http.Handler
	TLSConfig *tls.Config
}

// ClientConfig holds QUIC client configuration (no-op when quic tag absent).
type ClientConfig struct {
	TLSConfig      *tls.Config
	HandshakeTimeout time.Duration
	MaxIdleTimeout   time.Duration
	KeepAlivePeriod  time.Duration
}

// Server is a no-op QUIC server placeholder.
type Server struct{}

// NewServer returns a no-op server.
func NewServer(cfg ServerConfig) *Server {
	slog.Debug("quic: server not available, build with -tags quic to enable HTTP/3")
	return &Server{}
}

// ListenAndServe returns an error since QUIC is not enabled.
func (s *Server) ListenAndServe() error {
	return errors.New("quic: not available (build with -tags quic)")
}

// Close is a no-op.
func (s *Server) Close() error { return nil }

// Addr returns empty string.
func (s *Server) Addr() string { return "" }

// Client is a no-op QUIC client placeholder.
type Client struct{}

// NewClient returns a no-op client.
func NewClient(cfg ClientConfig) *Client {
	return &Client{}
}

// RoundTrip returns an error since QUIC is not enabled.
func (c *Client) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("quic: not available (build with -tags quic)")
}

// Close is a no-op.
func (c *Client) Close() error { return nil }

// NewHTTPClient returns nil since QUIC is not enabled.
func NewHTTPClient(cfg ClientConfig) *http.Client {
	return nil
}

// DefaultHTTPClient returns nil since QUIC is not enabled.
func DefaultHTTPClient() *http.Client {
	return nil
}
