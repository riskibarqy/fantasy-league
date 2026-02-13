package httpapi

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var apiTracer = otel.Tracer("fantasy-league/internal/interfaces/httpapi")

func startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return apiTracer.Start(ctx, name)
}
