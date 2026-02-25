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

func RequireInternalJobToken(token string, next http.Handler) http.Handler {
	expectedToken := strings.TrimSpace(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := startSpan(r.Context(), "httpapi.RequireInternalJobToken")
		defer span.End()

		if expectedToken == "" {
			writeError(ctx, w, fmt.Errorf("%w: internal job token is not configured", usecase.ErrDependencyUnavailable))
			return
		}

		providedToken := strings.TrimSpace(r.Header.Get("X-Internal-Job-Token"))
		if providedToken == "" || providedToken != expectedToken {
			writeError(ctx, w, fmt.Errorf("%w: invalid internal job token", usecase.ErrUnauthorized))
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
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
		otelhttp.WithFilter(func(r *http.Request) bool {
			return shouldTraceRequest(r.URL.Path)
		}),
	)
}

func shouldTraceRequest(path string) bool {
	normalized := strings.ToLower(strings.TrimSpace(path))
	switch normalized {
	case "/healthz", "/health", "/livez", "/readyz":
		return false
	default:
		return true
	}
}

func CORS(allowedOrigins []string, next http.Handler) http.Handler {
	allowAll := false
	allowMap := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		candidate := strings.TrimSpace(origin)
		if candidate == "" {
			continue
		}
		if candidate == "*" {
			allowAll = true
			continue
		}
		allowMap[candidate] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := startSpan(r.Context(), "httpapi.CORS")
		defer span.End()

		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		allowed := allowAll
		if !allowed {
			_, allowed = allowMap[origin]
		}
		if allowed {
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept")
			w.Header().Set("Access-Control-Max-Age", "600")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
