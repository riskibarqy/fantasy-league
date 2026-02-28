package usecase

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var usecaseTracer = otel.Tracer("fantasy-league/internal/usecase")
var usecaseNoopSpan = trace.SpanFromContext(context.Background())

func startUsecaseSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if strings.TrimSpace(name) == "" {
		return ctx, usecaseNoopSpan
	}
	parent := trace.SpanFromContext(ctx)
	if !parent.SpanContext().IsValid() {
		return ctx, usecaseNoopSpan
	}
	return usecaseTracer.Start(ctx, name)
}
