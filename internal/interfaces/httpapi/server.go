package httpapi

import (
	"log/slog"
	"net/http"
)

func NewRouter(
	handler *Handler,
	verifier TokenVerifier,
	logger *slog.Logger,
	swaggerEnabled bool,
	corsAllowedOrigins []string,
	internalJobToken string,
) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()
	registerSystemRoutes(mux, handler, swaggerEnabled)
	registerPublicDomainRoutes(mux, handler)
	registerAuthorizedRoutes(mux, handler, verifier)
	registerInternalJobRoutes(mux, handler, internalJobToken)

	return RequestTracing(RequestLogging(logger, CORS(corsAllowedOrigins, recoverPanic(logger, mux))))
}

func recoverPanic(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := startSpan(r.Context(), "httpapi.recoverPanic")
		defer span.End()

		defer func() {
			if rec := recover(); rec != nil {
				logger.ErrorContext(ctx, "panic recovered", "panic", rec)
				writeInternalError(ctx, w)
			}
		}()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
