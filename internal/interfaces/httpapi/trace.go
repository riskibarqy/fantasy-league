package httpapi

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var apiTracer = otel.Tracer("fantasy-league/internal/interfaces/httpapi")

func startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	parent := trace.SpanFromContext(ctx)
	if !parent.SpanContext().IsValid() {
		// No parent span in context (e.g. filtered route like /healthz):
		// avoid creating standalone root spans for internal helpers.
		return ctx, parent
	}
	return apiTracer.Start(ctx, name)
}
