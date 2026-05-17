package edgeagent

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ProxyHandler struct {
	originURL string
	nodeToken string
	wsEnabled bool
	grpcEnabled bool
}

func NewProxyHandler(originURL, nodeToken string, wsEnabled, grpcEnabled bool) *ProxyHandler {
	return &ProxyHandler{
		originURL:   originURL,
		nodeToken:   nodeToken,
		wsEnabled:   wsEnabled,
		grpcEnabled: grpcEnabled,
	}
}

func (p *ProxyHandler) IsWebSocket(r *http.Request) bool {
	if !p.wsEnabled {
		return false
	}
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

func (p *ProxyHandler) IsGRPC(r *http.Request) bool {
	if !p.grpcEnabled {
		return false
	}
	return strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc")
}

func (p *ProxyHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request, key string) {
	originURL, err := p.buildURL(key)
	if err != nil {
		slog.Error("ws proxy build url", "err", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}

	originReq, err := http.NewRequestWithContext(r.Context(), r.Method, originURL, nil)
	if err != nil {
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}

	p.copyHeaders(originReq.Header, r.Header)
	if p.nodeToken != "" {
		originReq.Header.Set("Authorization", "Bearer "+p.nodeToken)
	}
	originReq.Header.Set("X-Forwarded-For", r.RemoteAddr)

	dialer := net.Dialer{Timeout: 10 * time.Second}
	originConn, err := dialer.DialContext(r.Context(), "tcp", originReq.URL.Host)
	if err != nil {
		slog.Error("ws proxy dial origin", "err", err, "host", originReq.URL.Host)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}
	defer originConn.Close()

	if err := originReq.Write(originConn); err != nil {
		slog.Error("ws proxy write request", "err", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket not supported", http.StatusInternalServerError)
		return
	}
	clientConn, clientBuf, err := hj.Hijack()
	if err != nil {
		slog.Error("ws hijack failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	originResp, err := http.ReadResponse(clientBuf.Reader, originReq)
	if err != nil {
		slog.Error("ws read origin response", "err", err)
		return
	}

	if originResp.StatusCode != http.StatusSwitchingProtocols {
		originResp.Write(clientConn)
		originResp.Body.Close()
		return
	}

	if err := originResp.Write(clientConn); err != nil {
		return
	}

	slog.Info("ws proxy established", "key", key)

	errCh := make(chan error, 2)
	go func() { _, e := io.Copy(originConn, clientConn); errCh <- e }()
	go func() { _, e := io.Copy(clientConn, originConn); errCh <- e }()

	<-errCh
	slog.Debug("ws proxy closed", "key", key)
}

func (p *ProxyHandler) HandleGRPC(w http.ResponseWriter, r *http.Request, key string) {
	originURL, err := p.buildURL(key)
	if err != nil {
		slog.Error("grpc proxy build url", "err", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}

	originReq, err := http.NewRequestWithContext(r.Context(), r.Method, originURL, r.Body)
	if err != nil {
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}

	p.copyHeaders(originReq.Header, r.Header)
	originReq.Header.Set("Te", "trailers")
	if p.nodeToken != "" {
		originReq.Header.Set("Authorization", "Bearer "+p.nodeToken)
	}
	originReq.Header.Set("X-Forwarded-For", r.RemoteAddr)

	client := &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}

	originResp, err := client.Do(originReq)
	if err != nil {
		slog.Error("grpc proxy request failed", "err", err, "url", originURL)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}
	defer originResp.Body.Close()

	p.copyHeaders(w.Header(), originResp.Header)
	w.Header().Set("Trailer", "Grpc-Status, Grpc-Message, Grpc-Status-Details-Bin")
	w.WriteHeader(originResp.StatusCode)

	written, copyErr := io.Copy(w, originResp.Body)

	grpcStatus := originResp.Trailer.Get("Grpc-Status")
	if grpcStatus != "" {
		for k, vv := range originResp.Trailer {
			for _, v := range vv {
				w.Header().Set(k, v)
			}
		}
	}

	if copyErr != nil {
		slog.Debug("grpc proxy copy error", "err", copyErr, "written", written)
	}
}

func (p *ProxyHandler) buildURL(key string) (string, error) {
	base, err := url.Parse(p.originURL)
	if err != nil {
		return "", fmt.Errorf("parse origin URL: %w", err)
	}
	base.Path = strings.TrimSuffix(base.Path, "/") + "/obj/" + key
	return base.String(), nil
}

func (p *ProxyHandler) copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		kl := strings.ToLower(k)
		switch kl {
		case "connection", "keep-alive", "proxy-connection", "transfer-encoding", "upgrade":
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, key string) {
	if p.IsWebSocket(r) {
		p.HandleWebSocket(w, r, key)
		return
	}
	if p.IsGRPC(r) {
		p.HandleGRPC(w, r, key)
		return
	}
}
