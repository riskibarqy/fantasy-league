package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	sonic "github.com/bytedance/sonic"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func (h *Handler) UpsertSquad(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.UpsertSquad")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req upsertSquadRequest
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

	squad, err := h.squadService.UpsertSquad(ctx, usecase.UpsertSquadInput{
		UserID:    principal.UserID,
		LeagueID:  req.LeagueID,
		Name:      req.SquadName,
		PlayerIDs: req.PlayerIDs,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "upsert squad failed", "user_id", principal.UserID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, squadToDTO(ctx, squad))
}

func (h *Handler) PickSquad(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.PickSquad")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req pickSquadRequest
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

	squad, err := h.squadService.PickSquad(ctx, usecase.PickSquadInput{
		UserID:    principal.UserID,
		LeagueID:  req.LeagueID,
		SquadName: req.SquadName,
		PlayerIDs: req.PlayerIDs,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "pick squad failed", "user_id", principal.UserID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, squadToDTO(ctx, squad))
}

func (h *Handler) AddPlayerToMySquad(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.AddPlayerToMySquad")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req addPlayerToSquadRequest
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

	squad, err := h.squadService.AddPlayerToSquad(ctx, usecase.AddPlayerToSquadInput{
		UserID:    principal.UserID,
		LeagueID:  req.LeagueID,
		SquadName: req.SquadName,
		PlayerID:  req.PlayerID,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "add player to squad failed", "user_id", principal.UserID, "league_id", req.LeagueID, "player_id", req.PlayerID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, squadToDTO(ctx, squad))
}

func (h *Handler) GetMySquad(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetMySquad")
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

	squad, err := h.squadService.GetUserSquad(ctx, principal.UserID, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "get squad failed", "user_id", principal.UserID, "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, squadToDTO(ctx, squad))
}
