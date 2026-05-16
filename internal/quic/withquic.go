//go:build quic

package quic

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

func init() {
	Enabled = true
	slog.Info("quic: HTTP/3 support enabled")
}

// ServerConfig holds QUIC server configuration.
type ServerConfig struct {
	Addr      string
	Handler   http.Handler
	TLSConfig *tls.Config
}

// Server wraps an http3.Server to serve HTTP/3 traffic.
type Server struct {
	srv *http3.Server
	cfg ServerConfig
	ln  *quic.Listener
	mu  sync.RWMutex
}

// NewServer creates a new QUIC server.
func NewServer(cfg ServerConfig) *Server {
	if cfg.TLSConfig == nil {
		slog.Warn("quic: no TLS config provided, HTTP/3 requires TLS")
		return &Server{cfg: cfg}
	}
	tlsConfig := http3.ConfigureTLSConfig(cfg.TLSConfig)
	cfg.TLSConfig = tlsConfig
	return &Server{
		cfg: cfg,
		srv: &http3.Server{
			Addr:       cfg.Addr,
			Handler:    cfg.Handler,
			TLSConfig:  tlsConfig,
			QUICConfig: &quic.Config{EnableDatagrams: true},
		},
	}
}

// ListenAndServe starts the QUIC server on the configured address.
func (s *Server) ListenAndServe() error {
	if s.srv == nil {
		return fmt.Errorf("quic: server not initialized (missing TLS config)")
	}

	udpAddr, err := net.ResolveUDPAddr("udp", s.cfg.Addr)
	if err != nil {
		return fmt.Errorf("quic: resolve udp addr: %w", err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("quic: listen udp: %w", err)
	}

	tr := &quic.Transport{Conn: udpConn}
	ln, err := tr.Listen(s.cfg.TLSConfig, &quic.Config{
		EnableDatagrams: true,
	})
	if err != nil {
		udpConn.Close()
		return fmt.Errorf("quic: create listener: %w", err)
	}
	s.mu.Lock()
	s.ln = ln
	s.mu.Unlock()

	slog.Info("quic: HTTP/3 server listening", "addr", s.cfg.Addr)
	return s.srv.ServeListener(ln)
}

// Close gracefully shuts down the QUIC server.
func (s *Server) Close() error {
	s.mu.RLock()
	ln := s.ln
	s.mu.RUnlock()
	if ln != nil {
		return ln.Close()
	}
	if s.srv != nil {
		return s.srv.Close()
	}
	return nil
}

// Addr returns the listen address.
func (s *Server) Addr() string {
	s.mu.RLock()
	ln := s.ln
	s.mu.RUnlock()
	if ln != nil && ln.Addr() != nil {
		return ln.Addr().String()
	}
	return s.cfg.Addr
}
