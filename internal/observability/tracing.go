package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerConfig holds configuration for the OTel tracer.
type TracerConfig struct {
	ServiceName string
	Version     string
	Env         string
}

// InitTracer initialises the OpenTelemetry trace pipeline with an OTLP/gRPC
// exporter. It reads standard OTel env vars (OTEL_EXPORTER_OTLP_ENDPOINT,
// OTEL_TRACES_SAMPLER_ARG, etc.) and falls back to sensible defaults.
// Returns a shutdown function that flushes remaining spans.
func InitTracer(cfg TracerConfig) (func(context.Context) error, error) {
	ctx := context.Background()

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector:4317"
	}

	timeoutMS := 10000
	if v := os.Getenv("OTEL_EXPORTER_OTLP_TIMEOUT"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			timeoutMS = parsed
		}
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithTimeout(time.Duration(timeoutMS)*time.Millisecond),
	)
	if err != nil {
		return nil, fmt.Errorf("otlp exporter: %w", err)
	}

	env := cfg.Env
	if env == "" {
		env = os.Getenv("NSELF_ENV")
		if env == "" {
			env = "dev"
		}
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.Version),
			attribute.String("env", env),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	// Sampling: parentbased_traceidratio. Ratio from env or default (0.1 prod, 1.0 dev).
	ratio := 1.0
	if env == "prod" || env == "production" {
		ratio = 0.1
	}
	if v := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			ratio = parsed
		}
	}
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	slog.Info("otel tracer initialized",
		"service", cfg.ServiceName,
		"endpoint", endpoint,
		"sampler_ratio", ratio,
	)

	return tp.Shutdown, nil
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
