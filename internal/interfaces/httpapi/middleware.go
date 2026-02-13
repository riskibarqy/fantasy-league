package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/user"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// TokenVerifier verifies bearer tokens against account service.
type TokenVerifier interface {
	VerifyAccessToken(ctx context.Context, token string) (user.Principal, error)
}

func RequireAuth(verifier TokenVerifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := startSpan(r.Context(), "httpapi.RequireAuth")
		defer span.End()

		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			writeError(ctx, w, fmt.Errorf("%w: missing Authorization header", usecase.ErrUnauthorized))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			writeError(ctx, w, fmt.Errorf("%w: invalid Authorization header format", usecase.ErrUnauthorized))
			return
		}

		principal, err := verifier.VerifyAccessToken(ctx, strings.TrimSpace(parts[1]))
		if err != nil {
			writeError(ctx, w, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(withPrincipal(ctx, principal)))
	})
}

func RequestLogging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := startSpan(r.Context(), "httpapi.RequestLogging")
		defer span.End()

		started := time.Now()
		next.ServeHTTP(w, r.WithContext(ctx))

		spanContext := trace.SpanContextFromContext(ctx)
		traceID := ""
		spanID := ""
		if spanContext.IsValid() {
			traceID = spanContext.TraceID().String()
			spanID = spanContext.SpanID().String()
		}

		logger.InfoContext(ctx, "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"duration_ms", time.Since(started).Milliseconds(),
			"trace_id", traceID,
			"span_id", spanID,
		)
	})
}

func RequestTracing(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, "fantasy-league-http",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)
}
