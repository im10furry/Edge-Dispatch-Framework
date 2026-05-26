package tracing

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds OpenTelemetry tracing configuration.
type Config struct {
	Enabled       bool
	OTLPEndpoint  string
	ServiceName   string
	SampleRate    float64
	BatchTimeout  time.Duration
	ExportTimeout time.Duration
}

// DefaultConfig returns default tracing config (disabled).
func DefaultConfig() Config {
	return Config{
		Enabled:       false,
		OTLPEndpoint:  "localhost:4317",
		ServiceName:   "edge-dispatch",
		SampleRate:    1.0,
		BatchTimeout:  5 * time.Second,
		ExportTimeout: 30 * time.Second,
	}
}

// Init initializes OpenTelemetry tracing. Returns a shutdown function.
// If not enabled, returns a no-op shutdown function.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if !cfg.Enabled {
		otel.SetTracerProvider(otel.GetTracerProvider())
		return func(context.Context) error { return nil }, nil
	}

	if cfg.ServiceName == "" {
		cfg.ServiceName = "edge-dispatch"
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 1.0
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(cfg.BatchTimeout),
			sdktrace.WithExportTimeout(cfg.ExportTimeout),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	slog.Info("opentelemetry tracing initialized",
		"service", cfg.ServiceName,
		"endpoint", cfg.OTLPEndpoint,
		"sample_rate", cfg.SampleRate,
	)

	return tp.Shutdown, nil
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
