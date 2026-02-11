package httpapi

import (
	"log/slog"
	"net/http"
)

func NewRouter(handler *Handler, verifier TokenVerifier, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.Healthz)
	mux.HandleFunc("GET /v1/leagues", handler.ListLeagues)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/teams", handler.ListTeamsByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/players", handler.ListPlayersByLeague)
	mux.Handle("POST /v1/fantasy/squads", RequireAuth(verifier, http.HandlerFunc(handler.UpsertSquad)))
	mux.Handle("GET /v1/fantasy/squads/me", RequireAuth(verifier, http.HandlerFunc(handler.GetMySquad)))

	return RequestLogging(logger, recoverPanic(logger, mux))
}

func recoverPanic(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.ErrorContext(r.Context(), "panic recovered", "panic", rec)
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"error": map[string]any{
						"code":    "INTERNAL_ERROR",
						"message": "internal server error",
					},
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
