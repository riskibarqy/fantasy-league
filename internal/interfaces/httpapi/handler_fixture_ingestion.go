package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	sonic "github.com/bytedance/sonic"
	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/leaguestanding"
	"github.com/riskibarqy/fantasy-league/internal/domain/playerstats"
	"github.com/riskibarqy/fantasy-league/internal/domain/rawdata"
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

func (h *Handler) GetFixtureDetailsByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetFixtureDetailsByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	fixtureID := strings.TrimSpace(r.PathValue("fixtureID"))

	fixtureItem, err := h.fixtureService.GetByLeagueAndID(ctx, leagueID, fixtureID)
	if err != nil {
		h.logger.WarnContext(ctx, "get fixture details failed", "league_id", leagueID, "fixture_id", fixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while getting fixture details", "league_id", leagueID, "fixture_id", fixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}
	teamNameByID := make(map[string]string, len(teams))
	teamNameByRefID := make(map[int64]string, len(teams))
	teamLogoByID := make(map[string]string, len(teams))
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
		if t.TeamRefID > 0 {
			teamNameByRefID[t.TeamRefID] = t.Name
		}
		teamLogoByID[t.ID] = teamLogoWithFallback(ctx, t.Name, t.ImageURL)
	}

	players, err := h.playerService.ListPlayersByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list players failed while getting fixture details", "league_id", leagueID, "fixture_id", fixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}
	playerNameByID := make(map[string]string, len(players))
	for _, p := range players {
		playerNameByID[p.ID] = p.Name
	}

	teamStats, err := h.teamService.ListFixtureStatsByLeague(ctx, leagueID, fixtureID)
	if err != nil {
		h.logger.WarnContext(ctx, "list team fixture stats failed", "league_id", leagueID, "fixture_id", fixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}

	playerStats, err := h.playerStatsService.ListFixtureStats(ctx, leagueID, fixtureID)
	if err != nil {
		h.logger.WarnContext(ctx, "list player fixture stats failed", "league_id", leagueID, "fixture_id", fixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}

	events, err := h.playerStatsService.ListFixtureEvents(ctx, leagueID, fixtureID)
	if err != nil {
		h.logger.WarnContext(ctx, "list fixture events failed", "league_id", leagueID, "fixture_id", fixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teamStatsDTO := make([]fixtureTeamStatsDTO, 0, len(teamStats))
	for _, item := range teamStats {
		teamName := strings.TrimSpace(teamNameByID[item.TeamID])
		if teamName == "" && item.TeamExternalID > 0 {
			teamName = strings.TrimSpace(teamNameByRefID[item.TeamExternalID])
		}
		teamStatsDTO = append(teamStatsDTO, fixtureTeamStatsDTO{
			TeamID:         item.TeamID,
			TeamExternalID: item.TeamExternalID,
			TeamName:       teamName,
			PossessionPct:  item.PossessionPct,
			Shots:          item.Shots,
			ShotsOnTarget:  item.ShotsOnTarget,
			Corners:        item.Corners,
			Fouls:          item.Fouls,
			Offsides:       item.Offsides,
			AdvancedStats:  item.AdvancedStats,
		})
	}

	playerStatsDTO := make([]fixturePlayerStatsDTO, 0, len(playerStats))
	for _, item := range playerStats {
		teamName := strings.TrimSpace(teamNameByID[item.TeamID])
		if teamName == "" && item.TeamExternalID > 0 {
			teamName = strings.TrimSpace(teamNameByRefID[item.TeamExternalID])
		}
		playerStatsDTO = append(playerStatsDTO, fixturePlayerStatsDTO{
			PlayerID:         item.PlayerID,
			PlayerExternalID: item.PlayerExternalID,
			PlayerName:       strings.TrimSpace(playerNameByID[item.PlayerID]),
			TeamID:           item.TeamID,
			TeamExternalID:   item.TeamExternalID,
			TeamName:         teamName,
			MinutesPlayed:    item.MinutesPlayed,
			Goals:            item.Goals,
			Assists:          item.Assists,
			CleanSheet:       item.CleanSheet,
			YellowCards:      item.YellowCards,
			RedCards:         item.RedCards,
			Saves:            item.Saves,
			FantasyPoints:    item.FantasyPoints,
			AdvancedStats:    item.AdvancedStats,
		})
	}

	eventsDTO := make([]fixtureEventDTO, 0, len(events))
	for _, item := range events {
		eventsDTO = append(eventsDTO, fixtureEventDTO{
			EventID:                item.EventID,
			FixtureID:              item.FixtureID,
			FixtureExternalID:      item.FixtureExternalID,
			TeamID:                 item.TeamID,
			TeamExternalID:         item.TeamExternalID,
			PlayerID:               item.PlayerID,
			PlayerExternalID:       item.PlayerExternalID,
			AssistPlayerID:         item.AssistPlayerID,
			AssistPlayerExternalID: item.AssistPlayerExternalID,
			EventType:              item.EventType,
			Detail:                 item.Detail,
			Minute:                 item.Minute,
			ExtraMinute:            item.ExtraMinute,
			Metadata:               item.Metadata,
		})
	}

	writeSuccess(ctx, w, http.StatusOK, fixtureDetailsDTO{
		Fixture:     fixtureToDTO(ctx, fixtureItem, teamLogoByID),
		TeamStats:   teamStatsDTO,
		PlayerStats: playerStatsDTO,
		Events:      eventsDTO,
	})
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
			EventID:                item.EventID,
			FixtureID:              item.FixtureID,
			FixtureExternalID:      item.FixtureExternalID,
			TeamID:                 item.TeamID,
			TeamExternalID:         item.TeamExternalID,
			PlayerID:               item.PlayerID,
			PlayerExternalID:       item.PlayerExternalID,
			AssistPlayerID:         item.AssistPlayerID,
			AssistPlayerExternalID: item.AssistPlayerExternalID,
			EventType:              item.EventType,
			Detail:                 item.Detail,
			Minute:                 item.Minute,
			ExtraMinute:            item.ExtraMinute,
			Metadata:               item.Metadata,
		})
	}

	writeSuccess(ctx, w, http.StatusOK, out)
}

func (h *Handler) IngestPlayerFixtureStats(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.IngestPlayerFixtureStats")
	defer span.End()

	var req ingestPlayerFixtureStatsRequest
	decoder := sonic.ConfigDefault.NewDecoder(r.Body)
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
	rawItems := make([]rawdata.Payload, 0, len(req.Stats))
	for _, item := range req.Stats {
		playerKey := strings.TrimSpace(item.PlayerID)
		if playerKey == "" && item.ExternalPlayerID > 0 {
			playerKey = fmt.Sprintf("ext:%d", item.ExternalPlayerID)
		}

		stats = append(stats, playerstats.FixtureStat{
			PlayerID:          item.PlayerID,
			PlayerExternalID:  item.ExternalPlayerID,
			TeamID:            item.TeamID,
			TeamExternalID:    item.ExternalTeamID,
			FixtureExternalID: item.ExternalFixtureID,
			MinutesPlayed:     item.MinutesPlayed,
			Goals:             item.Goals,
			Assists:           item.Assists,
			CleanSheet:        item.CleanSheet,
			YellowCards:       item.YellowCards,
			RedCards:          item.RedCards,
			Saves:             item.Saves,
			FantasyPoints:     item.FantasyPoints,
			AdvancedStats:     item.AdvancedStats,
		})
		if len(item.Payload) > 0 {
			payloadJSON, err := marshalPayloadJSON(ctx, item.Payload)
			if err != nil {
				writeError(ctx, w, fmt.Errorf("%w: invalid player fixture stat payload: %v", usecase.ErrInvalidInput, err))
				return
			}
			rawItems = append(rawItems, rawdata.Payload{
				EntityType:      "player_fixture_stat",
				EntityKey:       req.FixtureID + ":" + playerKey,
				FixturePublicID: req.FixtureID,
				PlayerPublicID:  item.PlayerID,
				TeamPublicID:    item.TeamID,
				PayloadJSON:     payloadJSON,
			})
		}
	}

	if err := h.ingestionService.UpsertPlayerFixtureStats(ctx, req.FixtureID, stats); err != nil {
		h.logger.WarnContext(ctx, "ingest player fixture stats failed", "fixture_id", req.FixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}
	if len(rawItems) > 0 {
		if err := h.ingestionService.UpsertRawPayloads(ctx, "sportmonks", rawItems); err != nil {
			h.logger.WarnContext(ctx, "ingest player fixture stat payloads failed", "fixture_id", req.FixtureID, "error", err)
			writeError(ctx, w, err)
			return
		}
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
	decoder := sonic.ConfigDefault.NewDecoder(r.Body)
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
	rawItems := make([]rawdata.Payload, 0, len(req.Fixtures))
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
		if len(item.Payload) > 0 {
			payloadJSON, err := marshalPayloadJSON(ctx, item.Payload)
			if err != nil {
				writeError(ctx, w, fmt.Errorf("%w: invalid fixture payload for fixture_id=%s", usecase.ErrInvalidInput, item.ID))
				return
			}
			rawItems = append(rawItems, rawdata.Payload{
				EntityType:      "fixture",
				EntityKey:       item.ID,
				LeaguePublicID:  item.LeagueID,
				FixturePublicID: item.ID,
				PayloadJSON:     payloadJSON,
			})
		}
	}

	if err := h.ingestionService.UpsertFixtures(ctx, items); err != nil {
		h.logger.WarnContext(ctx, "ingest fixtures failed", "count", len(items), "error", err)
		writeError(ctx, w, err)
		return
	}
	if len(rawItems) > 0 {
		if err := h.ingestionService.UpsertRawPayloads(ctx, "sportmonks", rawItems); err != nil {
			h.logger.WarnContext(ctx, "ingest fixture payloads failed", "count", len(rawItems), "error", err)
			writeError(ctx, w, err)
			return
		}
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
	decoder := sonic.ConfigDefault.NewDecoder(r.Body)
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
	rawItems := make([]rawdata.Payload, 0, len(req.Stats))
	for _, item := range req.Stats {
		teamKey := strings.TrimSpace(item.TeamID)
		if teamKey == "" && item.ExternalTeamID > 0 {
			teamKey = fmt.Sprintf("ext:%d", item.ExternalTeamID)
		}

		stats = append(stats, teamstats.FixtureStat{
			TeamID:            item.TeamID,
			TeamExternalID:    item.ExternalTeamID,
			FixtureExternalID: item.ExternalFixtureID,
			PossessionPct:     item.PossessionPct,
			Shots:             item.Shots,
			ShotsOnTarget:     item.ShotsOnTarget,
			Corners:           item.Corners,
			Fouls:             item.Fouls,
			Offsides:          item.Offsides,
			AdvancedStats:     item.AdvancedStats,
		})
		if len(item.Payload) > 0 {
			payloadJSON, err := marshalPayloadJSON(ctx, item.Payload)
			if err != nil {
				writeError(ctx, w, fmt.Errorf("%w: invalid team fixture stat payload: %v", usecase.ErrInvalidInput, err))
				return
			}
			rawItems = append(rawItems, rawdata.Payload{
				EntityType:      "team_fixture_stat",
				EntityKey:       req.FixtureID + ":" + teamKey,
				FixturePublicID: req.FixtureID,
				TeamPublicID:    item.TeamID,
				PayloadJSON:     payloadJSON,
			})
		}
	}

	if err := h.ingestionService.UpsertTeamFixtureStats(ctx, req.FixtureID, stats); err != nil {
		h.logger.WarnContext(ctx, "ingest team fixture stats failed", "fixture_id", req.FixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}
	if len(rawItems) > 0 {
		if err := h.ingestionService.UpsertRawPayloads(ctx, "sportmonks", rawItems); err != nil {
			h.logger.WarnContext(ctx, "ingest team fixture stat payloads failed", "fixture_id", req.FixtureID, "error", err)
			writeError(ctx, w, err)
			return
		}
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
	decoder := sonic.ConfigDefault.NewDecoder(r.Body)
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
	rawItems := make([]rawdata.Payload, 0, len(req.Events))
	for _, item := range req.Events {
		events = append(events, playerstats.FixtureEvent{
			EventID:                item.EventID,
			FixtureExternalID:      item.ExternalFixtureID,
			TeamID:                 item.TeamID,
			TeamExternalID:         item.ExternalTeamID,
			PlayerID:               item.PlayerID,
			PlayerExternalID:       item.ExternalPlayerID,
			AssistPlayerID:         item.AssistPlayerID,
			AssistPlayerExternalID: item.ExternalAssistPlayerID,
			EventType:              item.EventType,
			Detail:                 item.Detail,
			Minute:                 item.Minute,
			ExtraMinute:            item.ExtraMinute,
			Metadata:               item.Metadata,
		})
		if len(item.Payload) > 0 {
			payloadJSON, err := marshalPayloadJSON(ctx, item.Payload)
			if err != nil {
				writeError(ctx, w, fmt.Errorf("%w: invalid fixture event payload: %v", usecase.ErrInvalidInput, err))
				return
			}
			eventKey := fmt.Sprintf("%s:%d:%d:%s:%s", req.FixtureID, item.Minute, item.ExtraMinute, item.EventType, item.PlayerID)
			if item.EventID > 0 {
				eventKey = fmt.Sprintf("%d", item.EventID)
			}
			rawItems = append(rawItems, rawdata.Payload{
				EntityType:      "fixture_event",
				EntityKey:       eventKey,
				FixturePublicID: req.FixtureID,
				TeamPublicID:    item.TeamID,
				PlayerPublicID:  item.PlayerID,
				PayloadJSON:     payloadJSON,
			})
		}
	}

	if err := h.ingestionService.ReplaceFixtureEvents(ctx, req.FixtureID, events); err != nil {
		h.logger.WarnContext(ctx, "ingest fixture events failed", "fixture_id", req.FixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}
	if len(rawItems) > 0 {
		if err := h.ingestionService.UpsertRawPayloads(ctx, "sportmonks", rawItems); err != nil {
			h.logger.WarnContext(ctx, "ingest fixture event payloads failed", "fixture_id", req.FixtureID, "error", err)
			writeError(ctx, w, err)
			return
		}
	}

	writeSuccess(ctx, w, http.StatusOK, map[string]any{
		"fixture_id": req.FixtureID,
		"count":      len(events),
		"updated":    true,
	})
}

func (h *Handler) IngestRawPayloads(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.IngestRawPayloads")
	defer span.End()

	var req ingestRawPayloadsRequest
	decoder := sonic.ConfigDefault.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	items := make([]rawdata.Payload, 0, len(req.Records))
	for _, record := range req.Records {
		payloadJSON, err := marshalPayloadJSON(ctx, record.Payload)
		if err != nil {
			writeError(ctx, w, fmt.Errorf("%w: invalid raw payload for entity_key=%s", usecase.ErrInvalidInput, record.EntityKey))
			return
		}

		var sourceUpdatedAt *time.Time
		if strings.TrimSpace(record.SourceUpdatedAt) != "" {
			parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(record.SourceUpdatedAt))
			if err != nil {
				writeError(ctx, w, fmt.Errorf("%w: invalid source_updated_at for entity_key=%s", usecase.ErrInvalidInput, record.EntityKey))
				return
			}
			sourceUpdatedAt = &parsed
		}

		items = append(items, rawdata.Payload{
			EntityType:      record.EntityType,
			EntityKey:       record.EntityKey,
			LeaguePublicID:  record.LeagueID,
			FixturePublicID: record.FixtureID,
			TeamPublicID:    record.TeamID,
			PlayerPublicID:  record.PlayerID,
			PayloadJSON:     payloadJSON,
			SourceUpdatedAt: sourceUpdatedAt,
		})
	}

	if err := h.ingestionService.UpsertRawPayloads(ctx, req.Source, items); err != nil {
		h.logger.WarnContext(ctx, "ingest raw payloads failed", "count", len(items), "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, map[string]any{
		"count":   len(items),
		"updated": true,
	})
}

func (h *Handler) IngestLeagueStandings(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.IngestLeagueStandings")
	defer span.End()

	var req ingestLeagueStandingsRequest
	decoder := sonic.ConfigDefault.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	items := make([]leaguestanding.Standing, 0, len(req.Items))
	for _, item := range req.Items {
		var sourceUpdatedAt *time.Time
		if strings.TrimSpace(item.SourceUpdatedAt) != "" {
			parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(item.SourceUpdatedAt))
			if err != nil {
				writeError(ctx, w, fmt.Errorf("%w: invalid source_updated_at for team_id=%s", usecase.ErrInvalidInput, item.TeamID))
				return
			}
			sourceUpdatedAt = &parsed
		}

		items = append(items, leaguestanding.Standing{
			LeagueID:        req.LeagueID,
			TeamID:          item.TeamID,
			IsLive:          req.IsLive,
			Gameweek:        req.Gameweek,
			Position:        item.Position,
			Played:          item.Played,
			Won:             item.Won,
			Draw:            item.Draw,
			Lost:            item.Lost,
			GoalsFor:        item.GoalsFor,
			GoalsAgainst:    item.GoalsAgainst,
			GoalDifference:  item.GoalDifference,
			Points:          item.Points,
			Form:            item.Form,
			SourceUpdatedAt: sourceUpdatedAt,
		})
	}

	gameweek := req.Gameweek
	if gameweek <= 0 {
		gameweek = inferStandingsGameweekFromIngest(req.Items)
	}

	if err := h.ingestionService.ReplaceLeagueStandings(ctx, req.LeagueID, req.IsLive, gameweek, items); err != nil {
		h.logger.WarnContext(ctx, "ingest league standings failed", "league_id", req.LeagueID, "is_live", req.IsLive, "count", len(items), "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, map[string]any{
		"league_id": req.LeagueID,
		"is_live":   req.IsLive,
		"gameweek":  gameweek,
		"count":     len(items),
		"updated":   true,
	})
}

func inferStandingsGameweekFromIngest(items []ingestLeagueStandingRecord) int {
	maxPlayed := 0
	for _, item := range items {
		if item.Played > maxPlayed {
			maxPlayed = item.Played
		}
	}
	if maxPlayed <= 0 {
		return 1
	}
	return maxPlayed
}

func marshalPayloadJSON(ctx context.Context, payload map[string]any) (string, error) {
	ctx, span := startSpan(ctx, "httpapi.marshalPayloadJSON")
	defer span.End()

	if len(payload) == 0 {
		return "", fmt.Errorf("payload is empty")
	}

	encoded, err := sonic.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	return string(encoded), nil
}
