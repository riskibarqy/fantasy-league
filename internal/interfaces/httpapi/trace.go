package httpapi

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var apiTracer = otel.Tracer("fantasy-league/internal/interfaces/httpapi")
var noopSpan = trace.SpanFromContext(context.Background())

func startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	parent := trace.SpanFromContext(ctx)
	if !parent.SpanContext().IsValid() {
		// No parent span in context (e.g. filtered route like /healthz):
		// avoid creating standalone root spans for internal helpers.
		return ctx, noopSpan
	}
	if !shouldCreateHTTPAPISpan(name) {
		return ctx, noopSpan
	}
	return apiTracer.Start(ctx, name)
}

func shouldCreateHTTPAPISpan(name string) bool {
	return strings.HasPrefix(name, "httpapi.Handler.")
}
