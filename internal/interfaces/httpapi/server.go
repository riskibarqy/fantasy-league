package httpapi

import (
	"log/slog"
	"net/http"
)

func NewRouter(handler *Handler, verifier TokenVerifier, logger *slog.Logger, swaggerEnabled bool, corsAllowedOrigins []string) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.Healthz)
	if swaggerEnabled {
		mux.HandleFunc("GET /openapi.yaml", handler.OpenAPI)
		mux.HandleFunc("GET /docs", handler.SwaggerUI)
		mux.HandleFunc("GET /docs/", handler.SwaggerUI)
	}
	mux.HandleFunc("GET /v1/dashboard", handler.GetDashboard)
	mux.HandleFunc("GET /v1/leagues", handler.ListLeagues)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/teams", handler.ListTeamsByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/teams/{teamID}", handler.GetTeamDetailsByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/teams/{teamID}/history", handler.GetTeamHistoryByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/teams/{teamID}/stats", handler.GetTeamStatsByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/fixtures", handler.ListFixturesByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/fixtures/{fixtureID}/events", handler.ListFixtureEventsByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/players", handler.ListPlayersByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/players/{playerID}", handler.GetPlayerDetailsByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/players/{playerID}/history", handler.GetPlayerHistoryByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/lineup", handler.GetLineupByLeague)
	mux.HandleFunc("PUT /v1/leagues/{leagueID}/lineup", handler.SaveLineupByLeague)
	mux.Handle("POST /v1/fantasy/squads", RequireAuth(verifier, http.HandlerFunc(handler.UpsertSquad)))
	mux.Handle("POST /v1/fantasy/squads/picks", RequireAuth(verifier, http.HandlerFunc(handler.PickSquad)))
	mux.Handle("GET /v1/fantasy/squads/me/players", RequireAuth(verifier, http.HandlerFunc(handler.ListMySquadPlayers)))
	mux.Handle("POST /v1/fantasy/squads/me/players", RequireAuth(verifier, http.HandlerFunc(handler.AddPlayerToMySquad)))
	mux.Handle("GET /v1/fantasy/squads/me", RequireAuth(verifier, http.HandlerFunc(handler.GetMySquad)))
	mux.Handle("POST /v1/custom-leagues", RequireAuth(verifier, http.HandlerFunc(handler.CreateCustomLeague)))
	mux.Handle("GET /v1/custom-leagues", RequireAuth(verifier, http.HandlerFunc(handler.ListMyCustomLeagues)))
	mux.Handle("GET /v1/custom-leagues/me", RequireAuth(verifier, http.HandlerFunc(handler.ListMyCustomLeagues)))
	mux.Handle("GET /v1/custom-leagues/{groupID}", RequireAuth(verifier, http.HandlerFunc(handler.GetCustomLeague)))
	mux.Handle("PUT /v1/custom-leagues/{groupID}", RequireAuth(verifier, http.HandlerFunc(handler.UpdateCustomLeague)))
	mux.Handle("DELETE /v1/custom-leagues/{groupID}", RequireAuth(verifier, http.HandlerFunc(handler.DeleteCustomLeague)))
	mux.Handle("POST /v1/custom-leagues/join", RequireAuth(verifier, http.HandlerFunc(handler.JoinCustomLeagueByInvite)))
	mux.Handle("GET /v1/custom-leagues/{groupID}/standings", RequireAuth(verifier, http.HandlerFunc(handler.ListCustomLeagueStandings)))

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
