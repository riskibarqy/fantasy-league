package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func (h *Handler) GetMySeasonPointsSummary(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetMySeasonPointsSummary")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}
	if h.scoringService == nil {
		writeError(ctx, w, fmt.Errorf("%w: scoring service is not configured", usecase.ErrDependencyUnavailable))
		return
	}

	leagueID := strings.TrimSpace(r.URL.Query().Get("league_id"))
	if err := h.validateRequest(ctx, getSquadRequest{LeagueID: leagueID}); err != nil {
		writeError(ctx, w, err)
		return
	}

	summary, err := h.scoringService.GetUserSeasonPointsSummary(ctx, leagueID, principal.UserID)
	if err != nil {
		h.logger.WarnContext(ctx, "get user season points summary failed", "user_id", principal.UserID, "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, userSeasonPointsSummaryToDTO(ctx, summary))
}

func (h *Handler) ListMyPlayerPointsByGameweek(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListMyPlayerPointsByGameweek")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}
	if h.scoringService == nil {
		writeError(ctx, w, fmt.Errorf("%w: scoring service is not configured", usecase.ErrDependencyUnavailable))
		return
	}

	leagueID := strings.TrimSpace(r.URL.Query().Get("league_id"))
	if err := h.validateRequest(ctx, getSquadRequest{LeagueID: leagueID}); err != nil {
		writeError(ctx, w, err)
		return
	}

	var gameweekFilter *int
	rawGameweek := strings.TrimSpace(r.URL.Query().Get("gameweek"))
	if rawGameweek != "" {
		value, err := strconv.Atoi(rawGameweek)
		if err != nil || value <= 0 {
			writeError(ctx, w, fmt.Errorf("%w: gameweek must be a positive integer", usecase.ErrInvalidInput))
			return
		}
		gameweekFilter = &value
	}

	items, err := h.scoringService.ListUserPlayerPointsByLeague(ctx, leagueID, principal.UserID, gameweekFilter)
	if err != nil {
		h.logger.WarnContext(ctx, "list user player points by gameweek failed", "user_id", principal.UserID, "league_id", leagueID, "gameweek", rawGameweek, "error", err)
		writeError(ctx, w, err)
		return
	}

	playerNameByID := make(map[string]string)
	if h.playerService != nil && len(items) > 0 {
		players, listErr := h.playerService.ListPlayersByLeague(ctx, leagueID)
		if listErr != nil {
			h.logger.WarnContext(ctx, "list players failed while mapping user player points", "league_id", leagueID, "error", listErr)
		} else {
			playerNameByID = make(map[string]string, len(players))
			for _, row := range players {
				playerNameByID[row.ID] = row.Name
			}
		}
	}

	responseItems := make([]userGameweekPlayerPointsDTO, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, userGameweekPlayerPointsToDTO(ctx, item, playerNameByID))
	}

	writeSuccess(ctx, w, http.StatusOK, userPlayerPointsResponseDTO{
		LeagueID:         leagueID,
		UserID:           principal.UserID,
		FilteredGameweek: gameweekFilter,
		Items:            responseItems,
	})
}
