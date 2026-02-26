package httpapi

import (
	"fmt"
	"github.com/riskibarqy/fantasy-league/internal/platform/logging"
	"net/http"

	"go.opentelemetry.io/otel/codes"
)

func NewRouter(
	handler *Handler,
	verifier TokenVerifier,
	logger *logging.Logger,
	swaggerEnabled bool,
	corsAllowedOrigins []string,
	internalJobToken string,
	traceRequestBody bool,
	traceRequestBodyMaxBytes int,
) http.Handler {
	if logger == nil {
		logger = logging.Default()
	}

	mux := http.NewServeMux()
	registerSystemRoutes(mux, handler, swaggerEnabled)
	registerPublicDomainRoutes(mux, handler)
	registerAuthorizedRoutes(mux, handler, verifier)
	registerInternalJobRoutes(mux, handler, internalJobToken)

	stack := RequestLogging(logger, CORS(corsAllowedOrigins, recoverPanic(logger, mux)))
	stack = RequestBodyTracing(traceRequestBody, traceRequestBodyMaxBytes, stack)
	return RequestTracing(stack)
}

func recoverPanic(logger *logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := startSpan(r.Context(), "httpapi.recoverPanic")
		defer span.End()

		defer func() {
			if rec := recover(); rec != nil {
				panicErr := fmt.Errorf("panic recovered: %v", rec)
				span.RecordError(panicErr)
				span.SetStatus(codes.Error, "panic")
				logger.ErrorContext(ctx, "panic recovered",
					"event", "panic_recovered",
					"error_code", "panic",
					"panic", rec,
				)
				writeInternalError(ctx, w)
			}
		}()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
