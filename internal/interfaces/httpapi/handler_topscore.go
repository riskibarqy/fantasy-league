package httpapi

import "net/http"

func (h *Handler) ListTopScorerByLeagueAndSeason(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListPlayersByLeague")
	defer span.End()

	leagueID := r.PathValue("leagueID")
	season := r.PathValue("season")
	topScorers, err := h.topScoreService.ListTopScorer(ctx, leagueID, season)
	if err != nil {
		h.logger.WarnContext(ctx, "list players failed", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	res := make(map[string][]TopScorePublicDTO)
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
