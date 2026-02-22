package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func (h *Handler) GetLineupByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetLineupByLeague")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	item, exists, err := h.lineupService.GetByUserAndLeague(ctx, principal.UserID, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "get lineup failed", "league_id", leagueID, "user_id", principal.UserID, "error", err)
		writeError(ctx, w, err)
		return
	}
	if !exists {
		writeSuccess(ctx, w, http.StatusOK, nil)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, lineupToDTO(ctx, item))
}

func (h *Handler) SaveLineupByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.SaveLineupByLeague")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	var req lineupUpsertRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}

	if strings.TrimSpace(req.LeagueID) == "" {
		req.LeagueID = leagueID
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}
	if strings.TrimSpace(req.LeagueID) != leagueID {
		writeError(ctx, w, fmt.Errorf("%w: league id mismatch between path and payload", usecase.ErrInvalidInput))
		return
	}

	item, err := h.lineupService.Save(ctx, usecase.SaveLineupInput{
		UserID:        principal.UserID,
		LeagueID:      req.LeagueID,
		GoalkeeperID:  req.GoalkeeperID,
		DefenderIDs:   req.DefenderIDs,
		MidfielderIDs: req.MidfielderIDs,
		ForwardIDs:    req.ForwardIDs,
		SubstituteIDs: req.SubstituteIDs,
		CaptainID:     req.CaptainID,
		ViceCaptainID: req.ViceCaptainID,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "save lineup failed", "league_id", leagueID, "user_id", principal.UserID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, lineupToDTO(ctx, item))
}
