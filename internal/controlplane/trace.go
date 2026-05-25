package controlplane

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

const (
	HeaderTraceID = "X-Trace-Id"
	HeaderSpanID  = "X-Span-Id"
)

func WithTraceContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(HeaderTraceID)
		if traceID == "" {
			traceID = uuid.New().String()
		}
		spanID := uuid.New().String()[:8]

		r.Header.Set(HeaderTraceID, traceID)
		r.Header.Set(HeaderSpanID, spanID)
		w.Header().Set(HeaderTraceID, traceID)

		slog.Info("request", "method", r.Method, "path", r.URL.Path, "trace_id", traceID, "span_id", spanID)

		next.ServeHTTP(w, r)
	})
}
