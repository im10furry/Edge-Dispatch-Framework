package tracing

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTPMiddleware wraps an HTTP handler with OpenTelemetry tracing.
func HTTPMiddleware(operation string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, operation)
	}
}

// HTTPHandler wraps a single HTTP handler function with tracing.
func HTTPHandler(operation string, handler http.HandlerFunc) http.Handler {
	return otelhttp.NewHandler(handler, operation)
}
