package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func (h *Handler) ListLeagues(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListLeagues")
	defer span.End()

	leagues, err := h.leagueService.ListLeagues(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list leagues failed", "error", err)
		writeError(ctx, w, err)
		return
	}

	items := make([]leaguePublicDTO, 0, len(leagues))
	for _, l := range leagues {
		items = append(items, leagueToPublicDTO(ctx, l))
	}

	writeSuccess(ctx, w, http.StatusOK, items)
}

func (h *Handler) ListTeamsByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListTeamsByLeague")
	defer span.End()

	leagueID := r.PathValue("leagueID")
	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	items := make([]teamDTO, 0, len(teams))
	for _, t := range teams {
		items = append(items, teamToDTO(ctx, t))
	}

	writeSuccess(ctx, w, http.StatusOK, items)
}

func (h *Handler) GetTeamDetailsByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetTeamDetailsByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	teamID := strings.TrimSpace(r.PathValue("teamID"))

	item, err := h.teamService.GetTeamDetailsByLeague(ctx, leagueID, teamID)
	if err != nil {
		h.logger.WarnContext(ctx, "get team details failed", "league_id", leagueID, "team_id", teamID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, teamDetailToDTO(ctx, item))
}

func (h *Handler) GetTeamHistoryByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetTeamHistoryByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	teamID := strings.TrimSpace(r.PathValue("teamID"))
	limit := 8
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			writeError(ctx, w, fmt.Errorf("%w: limit must be positive integer", usecase.ErrInvalidInput))
			return
		}
		limit = v
	}

	history, err := h.teamService.GetTeamHistoryByLeague(ctx, leagueID, teamID, limit)
	if err != nil {
		h.logger.WarnContext(ctx, "get team history failed", "league_id", leagueID, "team_id", teamID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while mapping team history", "league_id", leagueID, "team_id", teamID, "error", err)
		writeError(ctx, w, err)
		return
	}
	teamNameByID := make(map[string]string, len(teams))
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
	}

	writeSuccess(ctx, w, http.StatusOK, teamHistoryToDTO(ctx, history, teamNameByID))
}

func (h *Handler) GetTeamStatsByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetTeamStatsByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	teamID := strings.TrimSpace(r.PathValue("teamID"))

	stats, err := h.teamService.GetTeamStatsByLeague(ctx, leagueID, teamID)
	if err != nil {
		h.logger.WarnContext(ctx, "get team stats failed", "league_id", leagueID, "team_id", teamID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, teamSeasonStatsToDTO(ctx, stats))
}

func (h *Handler) ListLeagueStandings(w http.ResponseWriter, r *http.Request) {
	h.listLeagueStandings(w, r, false)
}

func (h *Handler) ListLiveLeagueStandings(w http.ResponseWriter, r *http.Request) {
	h.listLeagueStandings(w, r, true)
}

func (h *Handler) listLeagueStandings(w http.ResponseWriter, r *http.Request, live bool) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.listLeagueStandings")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	items, err := h.leagueStandingService.ListByLeague(ctx, leagueID, live)
	if err != nil {
		h.logger.WarnContext(ctx, "list league standings failed", "league_id", leagueID, "live", live, "error", err)
		writeError(ctx, w, err)
		return
	}

	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while mapping league standings", "league_id", leagueID, "live", live, "error", err)
		writeError(ctx, w, err)
		return
	}
	teamNameByID := make(map[string]string, len(teams))
	teamLogoByID := make(map[string]string, len(teams))
	for _, item := range teams {
		teamNameByID[item.ID] = item.Name
		teamLogoByID[item.ID] = teamLogoWithFallback(ctx, item.Name, item.ImageURL)
	}

	out := make([]leagueStandingDTO, 0, len(items))
	for _, item := range items {
		out = append(out, leagueStandingToDTO(ctx, item, teamNameByID[item.TeamID], teamLogoByID[item.TeamID]))
	}

	writeSuccess(ctx, w, http.StatusOK, out)
}
