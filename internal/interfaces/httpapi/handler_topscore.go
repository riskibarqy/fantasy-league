package httpapi

import (
	"github.com/riskibarqy/fantasy-league/internal/usecase"
	"net/http"
	"strings"
)

func (h *Handler) ListTopScorerByLeagueAndSeason(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListPlayersByLeague")
	defer span.End()

	leagueID := r.PathValue("leagueID")
	season := r.PathValue("season")
	s := strings.ReplaceAll(season, "-", "/")
	topScorers, err := h.topScoreService.ListTopScorer(ctx, leagueID, s)
	if err != nil {
		h.logger.WarnContext(ctx, "list top score failed", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	res := make(map[string][]TopScorePublicDTO, len(usecase.TopScoreTypeMap))
	for k, p := range topScorers {
		for _, item := range p {
			res[k] = append(res[k], TopScorePublicDTO{
				TypeID:           item.TypeID,
				TypeName:         item.TypeName,
				Rank:             item.Rank,
				Total:            item.Total,
				LeagueID:         item.LeagueID,
				PlayerID:         item.PlayerID,
				Season:           item.Season,
				ParticipantID:    item.ParticipantID,
				PlayerName:       item.PlayerName,
				ImagePlayer:      item.ImagePlayer,
				Nationality:      item.Nationality,
				ImageNationality: item.ImageNationality,
				ParticipantName:  item.ParticipantName,
				ImageParticipant: item.ImageParticipant,
				PositionName:     item.PositionName,
			})
		}
	}

	writeSuccess(ctx, w, http.StatusOK, res)
}
