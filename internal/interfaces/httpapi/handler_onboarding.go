package httpapi

import (
	"fmt"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func (h *Handler) SaveOnboardingFavoriteClub(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.SaveOnboardingFavoriteClub")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req onboardingFavoriteClubRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	profile, err := h.onboardingService.SaveFavoriteClub(ctx, usecase.SaveFavoriteClubInput{
		UserID:      principal.UserID,
		LeagueID:    req.LeagueID,
		TeamID:      req.TeamID,
		CountryCode: resolveCountryCode(ctx, r),
		IPAddress:   resolveClientIP(ctx, r),
	})
	if err != nil {
		h.logger.WarnContext(ctx, "save onboarding favorite club failed", "user_id", principal.UserID, "league_id", req.LeagueID, "team_id", req.TeamID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, onboardingProfileToDTO(ctx, profile))
}

func (h *Handler) CompleteOnboardingPickSquad(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.CompleteOnboardingPickSquad")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req onboardingPickSquadRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	profile, squad, savedLineup, err := h.onboardingService.Complete(ctx, usecase.CompleteOnboardingInput{
		UserID:        principal.UserID,
		LeagueID:      req.LeagueID,
		SquadName:     req.SquadName,
		PlayerIDs:     req.PlayerIDs,
		GoalkeeperID:  req.GoalkeeperID,
		DefenderIDs:   req.DefenderIDs,
		MidfielderIDs: req.MidfielderIDs,
		ForwardIDs:    req.ForwardIDs,
		SubstituteIDs: req.SubstituteIDs,
		CaptainID:     req.CaptainID,
		ViceCaptainID: req.ViceCaptainID,
		CountryCode:   resolveCountryCode(ctx, r),
		IPAddress:     resolveClientIP(ctx, r),
	})
	if err != nil {
		h.logger.WarnContext(ctx, "complete onboarding squad pick failed", "user_id", principal.UserID, "league_id", req.LeagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, onboardingCompleteResponseDTO{
		Profile: onboardingProfileToDTO(ctx, profile),
		Squad:   squadToDTO(ctx, squad),
		Lineup:  lineupToDTO(ctx, savedLineup),
	})
}
