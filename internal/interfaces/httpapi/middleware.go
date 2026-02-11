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
)

// TokenVerifier verifies bearer tokens against account service.
type TokenVerifier interface {
	VerifyAccessToken(ctx context.Context, token string) (user.Principal, error)
}

func RequireAuth(verifier TokenVerifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			writeError(w, fmt.Errorf("%w: missing Authorization header", usecase.ErrUnauthorized))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			writeError(w, fmt.Errorf("%w: invalid Authorization header format", usecase.ErrUnauthorized))
			return
		}

		principal, err := verifier.VerifyAccessToken(r.Context(), strings.TrimSpace(parts[1]))
		if err != nil {
			writeError(w, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal)))
	})
}

func RequestLogging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		logger.InfoContext(r.Context(), "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}
