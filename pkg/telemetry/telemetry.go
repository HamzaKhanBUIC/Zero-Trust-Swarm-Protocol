// Package telemetry initializes OpenTelemetry for distributed swarm tracing.
package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// InitTracer initializes an OpenTelemetry tracer that outputs beautifully to stdout.
func InitTracer(agentID string) (*sdktrace.TracerProvider, error) {
	// Use stdout exporter to avoid requiring an external Jaeger server for the demo
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize stdout exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(agentID),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

// ExtractContext reconstructs a trace context from an incoming TraceID and SpanID.
func ExtractContext(ctx context.Context, traceIDHex, spanIDHex string) context.Context {
	if traceIDHex == "" || spanIDHex == "" {
		return ctx
	}

	traceID, err := trace.TraceIDFromHex(traceIDHex)
	if err != nil {
		return ctx
	}
	spanID, err := trace.SpanIDFromHex(spanIDHex)
	if err != nil {
		return ctx
	}

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})

	return trace.ContextWithRemoteSpanContext(ctx, spanContext)
}
