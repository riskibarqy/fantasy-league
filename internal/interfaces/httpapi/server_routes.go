package httpapi

import "net/http"

func registerSystemRoutes(mux *http.ServeMux, handler *Handler, swaggerEnabled bool) {
	mux.HandleFunc("GET /healthz", handler.Healthz)
	if !swaggerEnabled {
		return
	}

	mux.HandleFunc("GET /openapi.yaml", handler.OpenAPI)
	mux.HandleFunc("GET /docs", handler.SwaggerUI)
	mux.HandleFunc("GET /docs/", handler.SwaggerUI)
}

func registerPublicDomainRoutes(mux *http.ServeMux, handler *Handler) {
	mux.HandleFunc("GET /v1/leagues", handler.ListLeagues)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/teams", handler.ListTeamsByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/teams/{teamID}", handler.GetTeamDetailsByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/teams/{teamID}/history", handler.GetTeamHistoryByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/teams/{teamID}/stats", handler.GetTeamStatsByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/players", handler.ListPlayersByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/topscorers/season/{season}", handler.ListTopScorerByLeagueAndSeason)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/players/{playerID}", handler.GetPlayerDetailsByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/players/{playerID}/history", handler.GetPlayerHistoryByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/fixtures", handler.ListFixturesByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/standings", handler.ListLeagueStandings)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/standings/live", handler.ListLiveLeagueStandings)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/fixtures/{fixtureID}", handler.GetFixtureDetailsByLeague)
	mux.HandleFunc("GET /v1/leagues/{leagueID}/fixtures/{fixtureID}/events", handler.ListFixtureEventsByLeague)
}

func registerAuthorizedRoutes(mux *http.ServeMux, handler *Handler, verifier TokenVerifier) {
	registerAuthorizedDashboardRoutes(mux, handler, verifier)
	registerAuthorizedFantasyRoutes(mux, handler, verifier)
	registerAuthorizedOnboardingRoutes(mux, handler, verifier)
	registerAuthorizedCustomLeagueRoutes(mux, handler, verifier)
	registerAuthorizedIngestionRoutes(mux, handler, verifier)
}

func registerInternalJobRoutes(mux *http.ServeMux, handler *Handler, internalJobToken string) {
	mux.Handle("POST /v1/internal/jobs/bootstrap", RequireInternalJobToken(internalJobToken, http.HandlerFunc(handler.RunBootstrapJob)))
	mux.Handle("POST /v1/internal/jobs/sync-schedule", RequireInternalJobToken(internalJobToken, http.HandlerFunc(handler.RunSyncScheduleJob)))
	mux.Handle("POST /v1/internal/jobs/sync-live", RequireInternalJobToken(internalJobToken, http.HandlerFunc(handler.RunSyncLiveJob)))
}

func registerAuthorizedDashboardRoutes(mux *http.ServeMux, handler *Handler, verifier TokenVerifier) {
	mux.Handle("GET /v1/dashboard", RequireAuth(verifier, http.HandlerFunc(handler.GetDashboard)))
}

func registerAuthorizedFantasyRoutes(mux *http.ServeMux, handler *Handler, verifier TokenVerifier) {
	mux.Handle("GET /v1/leagues/{leagueID}/lineup", RequireAuth(verifier, http.HandlerFunc(handler.GetLineupByLeague)))
	mux.Handle("PUT /v1/leagues/{leagueID}/lineup", RequireAuth(verifier, http.HandlerFunc(handler.SaveLineupByLeague)))
	mux.Handle("POST /v1/fantasy/squads", RequireAuth(verifier, http.HandlerFunc(handler.UpsertSquad)))
	mux.Handle("POST /v1/fantasy/squads/picks", RequireAuth(verifier, http.HandlerFunc(handler.PickSquad)))
	mux.Handle("GET /v1/fantasy/squads/me/players", RequireAuth(verifier, http.HandlerFunc(handler.ListMySquadPlayers)))
	mux.Handle("POST /v1/fantasy/squads/me/players", RequireAuth(verifier, http.HandlerFunc(handler.AddPlayerToMySquad)))
	mux.Handle("GET /v1/fantasy/squads/me", RequireAuth(verifier, http.HandlerFunc(handler.GetMySquad)))
	mux.Handle("GET /v1/fantasy/points/summary", RequireAuth(verifier, http.HandlerFunc(handler.GetMySeasonPointsSummary)))
	mux.Handle("GET /v1/fantasy/points/players", RequireAuth(verifier, http.HandlerFunc(handler.ListMyPlayerPointsByGameweek)))
}

func registerAuthorizedOnboardingRoutes(mux *http.ServeMux, handler *Handler, verifier TokenVerifier) {
	mux.Handle("PUT /v1/onboarding/favorite-club", RequireAuth(verifier, http.HandlerFunc(handler.SaveOnboardingFavoriteClub)))
	mux.Handle("POST /v1/onboarding/pick-squad", RequireAuth(verifier, http.HandlerFunc(handler.CompleteOnboardingPickSquad)))
}

func registerAuthorizedCustomLeagueRoutes(mux *http.ServeMux, handler *Handler, verifier TokenVerifier) {
	mux.Handle("POST /v1/custom-leagues", RequireAuth(verifier, http.HandlerFunc(handler.CreateCustomLeague)))
	mux.Handle("GET /v1/custom-leagues", RequireAuth(verifier, http.HandlerFunc(handler.ListMyCustomLeagues)))
	mux.Handle("GET /v1/custom-leagues/me", RequireAuth(verifier, http.HandlerFunc(handler.ListMyCustomLeagues)))
	mux.Handle("GET /v1/custom-leagues/{groupID}", RequireAuth(verifier, http.HandlerFunc(handler.GetCustomLeague)))
	mux.Handle("PUT /v1/custom-leagues/{groupID}", RequireAuth(verifier, http.HandlerFunc(handler.UpdateCustomLeague)))
	mux.Handle("DELETE /v1/custom-leagues/{groupID}", RequireAuth(verifier, http.HandlerFunc(handler.DeleteCustomLeague)))
	mux.Handle("POST /v1/custom-leagues/join", RequireAuth(verifier, http.HandlerFunc(handler.JoinCustomLeagueByInvite)))
	mux.Handle("GET /v1/custom-leagues/{groupID}/standings", RequireAuth(verifier, http.HandlerFunc(handler.ListCustomLeagueStandings)))
}

func registerAuthorizedIngestionRoutes(mux *http.ServeMux, handler *Handler, verifier TokenVerifier) {
	mux.Handle("POST /v1/internal/ingestion/fixtures", RequireAuth(verifier, http.HandlerFunc(handler.IngestFixtures)))
	mux.Handle("POST /v1/internal/ingestion/player-stats", RequireAuth(verifier, http.HandlerFunc(handler.IngestPlayerFixtureStats)))
	mux.Handle("POST /v1/internal/ingestion/team-stats", RequireAuth(verifier, http.HandlerFunc(handler.IngestTeamFixtureStats)))
	mux.Handle("POST /v1/internal/ingestion/fixture-events", RequireAuth(verifier, http.HandlerFunc(handler.IngestFixtureEvents)))
	mux.Handle("POST /v1/internal/ingestion/raw-payloads", RequireAuth(verifier, http.HandlerFunc(handler.IngestRawPayloads)))
	mux.Handle("POST /v1/internal/ingestion/standings", RequireAuth(verifier, http.HandlerFunc(handler.IngestLeagueStandings)))
	mux.Handle("POST /v1/internal/sync/schedule", RequireAuth(verifier, http.HandlerFunc(handler.RunSyncScheduleDirect)))
	mux.Handle("POST /v1/internal/sync/resync", RequireAuth(verifier, http.HandlerFunc(handler.RunResync)))
	// Master data sync for season initialization (teams + players + stat types catalogs).
	mux.Handle("POST /v1/internal/sync/master-data", RequireAuth(verifier, http.HandlerFunc(handler.RunSyncMasterData)))
	// Team schedule sync focused on fixtures/timeline refresh.
	mux.Handle("POST /v1/internal/sync/team-schedule", RequireAuth(verifier, http.HandlerFunc(handler.RunSyncTeamSchedule)))
	// Reconcile sync for repairing data mismatches across fixtures/stats/standings.
	mux.Handle("POST /v1/internal/sync/reconcile", RequireAuth(verifier, http.HandlerFunc(handler.RunSyncReconcile)))
	// Get a previously executed sync result by run id.
	mux.Handle("GET /v1/internal/sync/runs/{runID}", RequireAuth(verifier, http.HandlerFunc(handler.GetSyncRun)))
}
