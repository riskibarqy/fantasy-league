package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func (h *Handler) ListPlayersByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListPlayersByLeague")
	defer span.End()

	leagueID := r.PathValue("leagueID")
	players, err := h.playerService.ListPlayersByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list players failed", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while mapping players", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teamNameByID := make(map[string]string, len(teams))
	teamLogoByID := make(map[string]string, len(teams))
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
		teamLogoByID[t.ID] = teamLogoWithFallback(ctx, t.Name, t.ImageURL)
	}

	items := make([]playerPublicDTO, 0, len(players))
	for _, p := range players {
		teamName := teamNameByID[p.TeamID]
		items = append(items, playerToPublicDTO(ctx, p, teamName, p.ImageURL, teamLogoByID[p.TeamID]))
	}

	writeSuccess(ctx, w, http.StatusOK, items)
}

func (h *Handler) GetPlayerDetailsByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetPlayerDetailsByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	playerID := strings.TrimSpace(r.PathValue("playerID"))

	item, err := h.playerService.GetPlayerByLeagueAndID(ctx, leagueID, playerID)
	if err != nil {
		h.logger.WarnContext(ctx, "get player details failed", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while getting player details", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}
	teamNameByID := make(map[string]string, len(teams))
	teamLogoByID := make(map[string]string, len(teams))
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
		teamLogoByID[t.ID] = teamLogoWithFallback(ctx, t.Name, t.ImageURL)
	}

	historyItems, err := h.playerStatsService.ListMatchHistory(ctx, leagueID, playerID, 8)
	if err != nil {
		h.logger.WarnContext(ctx, "list player history failed while getting player details", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}

	stats, err := h.playerStatsService.GetSeasonStats(ctx, leagueID, playerID)
	if err != nil {
		h.logger.WarnContext(ctx, "get player stats failed while getting player details", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teamName := teamNameByID[item.TeamID]
	teamLogo := teamLogoByID[item.TeamID]
	history := historyToDTO(ctx, item.TeamID, historyItems, teamNameByID)

	writeSuccess(ctx, w, http.StatusOK, playerDetailDTO{
		Player:     playerToPublicDTO(ctx, item, teamName, item.ImageURL, teamLogo),
		Statistics: seasonStatsToDTO(ctx, stats),
		History:    history,
	})
}

func (h *Handler) GetPlayerHistoryByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetPlayerHistoryByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	playerID := strings.TrimSpace(r.PathValue("playerID"))

	item, err := h.playerService.GetPlayerByLeagueAndID(ctx, leagueID, playerID)
	if err != nil {
		h.logger.WarnContext(ctx, "get player for history failed", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while getting player history", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}
	teamNameByID := make(map[string]string, len(teams))
	teamLogoByID := make(map[string]string, len(teams))
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
		teamLogoByID[t.ID] = teamLogoWithFallback(ctx, t.Name, t.ImageURL)
	}

	historyItems, err := h.playerStatsService.ListMatchHistory(ctx, leagueID, playerID, 8)
	if err != nil {
		h.logger.WarnContext(ctx, "list player history failed", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, historyToDTO(ctx, item.TeamID, historyItems, teamNameByID))
}

func (h *Handler) ListMySquadPlayers(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListMySquadPlayers")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	leagueID := strings.TrimSpace(r.URL.Query().Get("league_id"))
	if err := h.validateRequest(ctx, getSquadRequest{LeagueID: leagueID}); err != nil {
		writeError(ctx, w, err)
		return
	}

	players, err := h.playerService.ListPlayersByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list players for squad failed", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}
	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while mapping squad players", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teamNameByID := make(map[string]string, len(teams))
	teamLogoByID := make(map[string]string, len(teams))
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
		teamLogoByID[t.ID] = teamLogoWithFallback(ctx, t.Name, t.ImageURL)
	}

	squadPlayerSet := make(map[string]struct{})
	existingSquad, err := h.squadService.GetUserSquad(ctx, principal.UserID, leagueID)
	if err != nil {
		if !errors.Is(err, usecase.ErrNotFound) {
			h.logger.WarnContext(ctx, "get squad failed while listing squad players", "user_id", principal.UserID, "league_id", leagueID, "error", err)
			writeError(ctx, w, err)
			return
		}
	} else {
		for _, pick := range existingSquad.Picks {
			squadPlayerSet[pick.PlayerID] = struct{}{}
		}
	}

	items := make([]squadPlayerDTO, 0, len(players))
	for _, p := range players {
		_, inSquad := squadPlayerSet[p.ID]
		club := teamNameByID[p.TeamID]
		if strings.TrimSpace(club) == "" {
			club = p.TeamID
		}
		items = append(items, squadPlayerDTO{
			ID:              p.ID,
			LeagueID:        p.LeagueID,
			Name:            p.Name,
			Club:            club,
			Position:        string(p.Position),
			Price:           float64(p.Price) / 10.0,
			Form:            derivedForm(ctx, p.ID),
			ProjectedPoints: derivedProjectedPoints(ctx, p.ID),
			IsInjured:       isInjured(ctx, p.ID),
			InSquad:         inSquad,
			ImageURL:        playerImageWithFallback(ctx, p.ID, p.Name, p.ImageURL),
			TeamLogoURL:     teamLogoByID[p.TeamID],
		})
	}

	writeSuccess(ctx, w, http.StatusOK, items)
}
