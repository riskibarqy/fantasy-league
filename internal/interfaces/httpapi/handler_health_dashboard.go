package httpapi

import (
	"fmt"
	"net/http"

	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.Healthz")
	defer span.End()

	writeSuccess(ctx, w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetDashboard")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	dashboard, err := h.dashboardService.Get(ctx, principal.UserID)
	if err != nil {
		h.logger.ErrorContext(ctx, "get dashboard failed", "user_id", principal.UserID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, dashboardDTO{
		Gameweek:         dashboard.Gameweek,
		Budget:           dashboard.Budget,
		TeamValue:        dashboard.TeamValue,
		TotalPoints:      dashboard.TotalPoints,
		Rank:             dashboard.Rank,
		SelectedLeagueID: dashboard.SelectedLeagueID,
	})
}
