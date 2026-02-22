package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/playerstats"
	"github.com/riskibarqy/fantasy-league/internal/domain/teamstats"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func (h *Handler) ListFixturesByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListFixturesByLeague")
	defer span.End()

	leagueID := r.PathValue("leagueID")
	fixtures, err := h.fixtureService.ListByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list fixtures failed", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while mapping fixtures", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teamLogoByID := make(map[string]string, len(teams))
	for _, t := range teams {
		teamLogoByID[t.ID] = teamLogoWithFallback(ctx, t.Name, t.ImageURL)
	}

	items := make([]fixtureDTO, 0, len(fixtures))
	for _, f := range fixtures {
		items = append(items, fixtureToDTO(ctx, f, teamLogoByID))
	}

	writeSuccess(ctx, w, http.StatusOK, items)
}

func (h *Handler) ListFixtureEventsByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListFixtureEventsByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	fixtureID := strings.TrimSpace(r.PathValue("fixtureID"))

	items, err := h.playerStatsService.ListFixtureEvents(ctx, leagueID, fixtureID)
	if err != nil {
		h.logger.WarnContext(ctx, "list fixture events failed", "league_id", leagueID, "fixture_id", fixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}

	out := make([]fixtureEventDTO, 0, len(items))
	for _, item := range items {
		out = append(out, fixtureEventDTO{
			EventID:        item.EventID,
			FixtureID:      item.FixtureID,
			TeamID:         item.TeamID,
			PlayerID:       item.PlayerID,
			AssistPlayerID: item.AssistPlayerID,
			EventType:      item.EventType,
			Detail:         item.Detail,
			Minute:         item.Minute,
			ExtraMinute:    item.ExtraMinute,
		})
	}

	writeSuccess(ctx, w, http.StatusOK, out)
}

func (h *Handler) IngestPlayerFixtureStats(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.IngestPlayerFixtureStats")
	defer span.End()

	var req ingestPlayerFixtureStatsRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	stats := make([]playerstats.FixtureStat, 0, len(req.Stats))
	for _, item := range req.Stats {
		stats = append(stats, playerstats.FixtureStat{
			PlayerID:      item.PlayerID,
			TeamID:        item.TeamID,
			MinutesPlayed: item.MinutesPlayed,
			Goals:         item.Goals,
			Assists:       item.Assists,
			CleanSheet:    item.CleanSheet,
			YellowCards:   item.YellowCards,
			RedCards:      item.RedCards,
			Saves:         item.Saves,
			FantasyPoints: item.FantasyPoints,
		})
	}

	if err := h.ingestionService.UpsertPlayerFixtureStats(ctx, req.FixtureID, stats); err != nil {
		h.logger.WarnContext(ctx, "ingest player fixture stats failed", "fixture_id", req.FixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, map[string]any{
		"fixture_id": req.FixtureID,
		"count":      len(stats),
		"updated":    true,
	})
}

func (h *Handler) IngestFixtures(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.IngestFixtures")
	defer span.End()

	var req ingestFixturesRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	items := make([]fixture.Fixture, 0, len(req.Fixtures))
	for _, item := range req.Fixtures {
		kickoffAt, err := time.Parse(time.RFC3339, strings.TrimSpace(item.KickoffAt))
		if err != nil {
			writeError(ctx, w, fmt.Errorf("%w: invalid kickoff_at for fixture_id=%s", usecase.ErrInvalidInput, item.ID))
			return
		}
		var finishedAt *time.Time
		if strings.TrimSpace(item.FinishedAt) != "" {
			v, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(item.FinishedAt))
			if parseErr != nil {
				writeError(ctx, w, fmt.Errorf("%w: invalid finished_at for fixture_id=%s", usecase.ErrInvalidInput, item.ID))
				return
			}
			finishedAt = &v
		}

		items = append(items, fixture.Fixture{
			ID:           item.ID,
			LeagueID:     item.LeagueID,
			Gameweek:     item.Gameweek,
			HomeTeam:     item.HomeTeam,
			AwayTeam:     item.AwayTeam,
			HomeTeamID:   item.HomeTeamID,
			AwayTeamID:   item.AwayTeamID,
			FixtureRefID: item.FixtureRefID,
			KickoffAt:    kickoffAt.UTC(),
			Venue:        item.Venue,
			HomeScore:    item.HomeScore,
			AwayScore:    item.AwayScore,
			Status:       item.Status,
			WinnerTeamID: item.WinnerTeamID,
			FinishedAt:   finishedAt,
		})
	}

	if err := h.ingestionService.UpsertFixtures(ctx, items); err != nil {
		h.logger.WarnContext(ctx, "ingest fixtures failed", "count", len(items), "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, map[string]any{
		"count":   len(items),
		"updated": true,
	})
}

func (h *Handler) IngestTeamFixtureStats(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.IngestTeamFixtureStats")
	defer span.End()

	var req ingestTeamFixtureStatsRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	stats := make([]teamstats.FixtureStat, 0, len(req.Stats))
	for _, item := range req.Stats {
		stats = append(stats, teamstats.FixtureStat{
			TeamID:        item.TeamID,
			PossessionPct: item.PossessionPct,
			Shots:         item.Shots,
			ShotsOnTarget: item.ShotsOnTarget,
			Corners:       item.Corners,
			Fouls:         item.Fouls,
			Offsides:      item.Offsides,
		})
	}

	if err := h.ingestionService.UpsertTeamFixtureStats(ctx, req.FixtureID, stats); err != nil {
		h.logger.WarnContext(ctx, "ingest team fixture stats failed", "fixture_id", req.FixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, map[string]any{
		"fixture_id": req.FixtureID,
		"count":      len(stats),
		"updated":    true,
	})
}

func (h *Handler) IngestFixtureEvents(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.IngestFixtureEvents")
	defer span.End()

	var req ingestFixtureEventsRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	events := make([]playerstats.FixtureEvent, 0, len(req.Events))
	for _, item := range req.Events {
		events = append(events, playerstats.FixtureEvent{
			EventID:        item.EventID,
			TeamID:         item.TeamID,
			PlayerID:       item.PlayerID,
			AssistPlayerID: item.AssistPlayerID,
			EventType:      item.EventType,
			Detail:         item.Detail,
			Minute:         item.Minute,
			ExtraMinute:    item.ExtraMinute,
		})
	}

	if err := h.ingestionService.ReplaceFixtureEvents(ctx, req.FixtureID, events); err != nil {
		h.logger.WarnContext(ctx, "ingest fixture events failed", "fixture_id", req.FixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, map[string]any{
		"fixture_id": req.FixtureID,
		"count":      len(events),
		"updated":    true,
	})
}
