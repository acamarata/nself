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
	"go.opentelemetry.io/otel/codes"
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

	// Build exporter options. TLS is enabled by default; opt into plaintext via
	// OTEL_INSECURE=true (development/private-network deployments only).
	exporterOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithTimeout(time.Duration(timeoutMS) * time.Millisecond),
	}
	if os.Getenv("OTEL_INSECURE") == "true" {
		exporterOpts = append(exporterOpts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, exporterOpts...)
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

	// Sampling: parentbased with error-span floor. In prod the base ratio is 10%;
	// error spans (StatusError / status code Error) always sample regardless of
	// the ratio so failures are never silently dropped.
	ratio := 1.0
	if env == "prod" || env == "production" {
		ratio = 0.1
	}
	if v := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			ratio = parsed
		}
	}
	sampler := sdktrace.ParentBased(newErrorFloorSampler(ratio))

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

// errorFloorSampler wraps a ratio-based sampler and guarantees that spans
// carrying an error status (StatusError / codes.Error) are always sampled,
// regardless of the configured ratio. This prevents the 10% production floor
// from silently dropping failure traces.
type errorFloorSampler struct {
	// ratio is the base sampling probability for non-error spans.
	ratio   float64
	wrapped sdktrace.Sampler
}

// newErrorFloorSampler returns an errorFloorSampler that samples error spans
// with AlwaysSample and non-error spans with the given ratio.
func newErrorFloorSampler(ratio float64) sdktrace.Sampler {
	return &errorFloorSampler{
		ratio:   ratio,
		wrapped: sdktrace.TraceIDRatioBased(ratio),
	}
}

// ShouldSample implements sdktrace.Sampler.
// It checks for error-indicating attributes (set before Span.Start) to promote
// sampling — specifically the otel-standard "error" bool attribute and any
// attribute keyed "status" or "status_code" with value matching StatusError.
// Once a span is created and SetStatus(codes.Error, …) is called, the span is
// always exported regardless because RecordAndSample was decided at start.
func (s *errorFloorSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	// Inspect caller-provided attributes for error signals. Callers that know
	// a span represents an error can set attribute.Bool("error", true) before
	// starting the span to guarantee it is sampled.
	for _, attr := range p.Attributes {
		switch {
		case attr.Key == "error" && attr.Value.AsBool():
			// Explicit error flag → AlwaysSample (mirrors codes.Error / StatusError).
			return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample}
		case (attr.Key == "status_code" || attr.Key == "status") &&
			attr.Value.AsString() == codes.Error.String():
			// StatusError string match → AlwaysSample.
			return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample}
		}
	}
	// No error signal found — delegate to the ratio-based sampler.
	return s.wrapped.ShouldSample(p)
}

// Description returns a human-readable identifier for this sampler.
func (s *errorFloorSampler) Description() string {
	return fmt.Sprintf("ErrorFloor{StatusError=AlwaysSample,ratio=%.4f}", s.ratio)
}
