package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	sonic "github.com/bytedance/sonic"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func (h *Handler) CreateCustomLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.CreateCustomLeague")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req createCustomLeagueRequest
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

	group, err := h.customLeagueService.CreateGroup(ctx, usecase.CreateCustomLeagueInput{
		UserID:   principal.UserID,
		LeagueID: req.LeagueID,
		Name:     req.Name,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "create custom league failed", "user_id", principal.UserID, "league_id", req.LeagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusCreated, customLeagueToDTO(ctx, group))
}

func (h *Handler) ListMyCustomLeagues(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListMyCustomLeagues")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	groups, err := h.customLeagueService.ListMyGroups(ctx, principal.UserID)
	if err != nil {
		h.logger.WarnContext(ctx, "list my custom leagues failed", "user_id", principal.UserID, "error", err)
		writeError(ctx, w, err)
		return
	}

	items := make([]customLeagueListDTO, 0, len(groups))
	for _, group := range groups {
		items = append(items, customLeagueListToDTO(ctx, group))
	}
	writeSuccess(ctx, w, http.StatusOK, items)
}

func (h *Handler) GetCustomLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetCustomLeague")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}
	groupID := strings.TrimSpace(r.PathValue("groupID"))

	group, err := h.customLeagueService.GetGroup(ctx, principal.UserID, groupID)
	if err != nil {
		h.logger.WarnContext(ctx, "get custom league failed", "user_id", principal.UserID, "group_id", groupID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, customLeagueToDTO(ctx, group))
}

func (h *Handler) UpdateCustomLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.UpdateCustomLeague")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}
	groupID := strings.TrimSpace(r.PathValue("groupID"))

	var req updateCustomLeagueRequest
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

	if err := h.customLeagueService.UpdateGroupName(ctx, usecase.UpdateCustomLeagueInput{
		UserID:  principal.UserID,
		GroupID: groupID,
		Name:    req.Name,
	}); err != nil {
		h.logger.WarnContext(ctx, "update custom league failed", "user_id", principal.UserID, "group_id", groupID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, map[string]bool{"updated": true})
}

func (h *Handler) DeleteCustomLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.DeleteCustomLeague")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}
	groupID := strings.TrimSpace(r.PathValue("groupID"))

	if err := h.customLeagueService.DeleteGroup(ctx, principal.UserID, groupID); err != nil {
		h.logger.WarnContext(ctx, "delete custom league failed", "user_id", principal.UserID, "group_id", groupID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *Handler) JoinCustomLeagueByInvite(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.JoinCustomLeagueByInvite")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req joinCustomLeagueByInviteRequest
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

	group, err := h.customLeagueService.JoinByInviteCode(ctx, usecase.JoinCustomLeagueByInviteInput{
		UserID:     principal.UserID,
		InviteCode: req.InviteCode,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "join custom league by invite failed", "user_id", principal.UserID, "invite_code", req.InviteCode, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, customLeagueToDTO(ctx, group))
}

func (h *Handler) ListCustomLeagueStandings(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListCustomLeagueStandings")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}
	groupID := strings.TrimSpace(r.PathValue("groupID"))

	standings, err := h.customLeagueService.GetStandings(ctx, principal.UserID, groupID)
	if err != nil {
		h.logger.WarnContext(ctx, "list custom league standings failed", "user_id", principal.UserID, "group_id", groupID, "error", err)
		writeError(ctx, w, err)
		return
	}

	items := make([]customLeagueStandingDTO, 0, len(standings))
	for _, standing := range standings {
		items = append(items, customLeagueStandingToDTO(ctx, standing))
	}
	writeSuccess(ctx, w, http.StatusOK, items)
}
