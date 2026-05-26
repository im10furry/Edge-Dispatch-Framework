package waf

import (
	"log/slog"
	"net/http"

	"github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/types"
)

// Config holds WAF configuration.
type Config struct {
	Enabled            bool
	RequestBodyLimit   int64
	ResponseBodyAccess bool
	DenyStatusCode     int
}

// DefaultConfig returns sensible WAF defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:            false,
		RequestBodyLimit:   131072,
		ResponseBodyAccess: false,
		DenyStatusCode:     http.StatusForbidden,
	}
}

// New creates a Coraza WAF instance with sensible defaults.
func New(cfg Config) (coraza.WAF, error) {
	wafConfig := coraza.NewWAFConfig().
		WithRequestBodyAccess().
		WithRequestBodyLimit(int(cfg.RequestBodyLimit)).
		WithRequestBodyInMemoryLimit(int(cfg.RequestBodyLimit))

	if cfg.ResponseBodyAccess {
		wafConfig = wafConfig.WithResponseBodyAccess()
	}

	rules := []string{
		"SecRule REQUEST_URI \"@rx (?:\\.\\./|\\.\\\\.)\" \"id:1,phase:1,deny,status:403,msg:Path traversal attempt\"",
		"SecRule ARGS|ARGS_NAMES|REQUEST_COOKIES|REQUEST_HEADERS \"@rx (?i:(?:union\\s+select|insert\\s+into|delete\\s+from|drop\\s+table|update\\s+\\w+\\s+set))\" \"id:2,phase:2,deny,status:403,msg:SQL injection attempt\"",
		"SecRule ARGS|ARGS_NAMES \"@rx (?:<script|javascript:)\" \"id:3,phase:2,deny,status:403,msg:XSS attempt\"",
		"SecRule ARGS \"@rx (?:https?|ftp|php|data)://\" \"id:4,phase:2,deny,status:403,msg:Remote file inclusion attempt\"",
	}

	for _, rule := range rules {
		wafConfig = wafConfig.WithDirectives(rule)
	}

	return coraza.NewWAF(wafConfig)
}

// Middleware returns an HTTP middleware that inspects requests with the WAF.
func Middleware(waf coraza.WAF, denyStatusCode int) func(http.Handler) http.Handler {
	if waf == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tx := waf.NewTransaction()
			defer func() {
				if err := tx.Close(); err != nil {
					slog.Warn("waf: failed to close transaction", "err", err)
				}
			}()

			tx.ProcessConnection(r.RemoteAddr, 0, r.Host, 0)
			tx.ProcessURI(r.URL.String(), r.Method, r.Proto)

			for k, vals := range r.Header {
				for _, v := range vals {
					tx.AddRequestHeader(k, v)
				}
			}
			tx.AddRequestHeader("Host", r.Host)

			if it := processHeaders(tx, r); it != nil {
				slog.Warn("waf: request denied",
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
					"rule_id", it.RuleID,
				)
				http.Error(w, "Forbidden", denyStatusCode)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func processHeaders(tx types.Transaction, r *http.Request) *types.Interruption {
	return tx.ProcessRequestHeaders()
}
