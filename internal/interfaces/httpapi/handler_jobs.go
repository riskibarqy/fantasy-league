package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func (h *Handler) RunSyncScheduleJob(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.RunSyncScheduleJob")
	defer span.End()

	if h.jobOrchestrator == nil {
		writeError(ctx, w, fmt.Errorf("%w: job orchestrator is not configured", usecase.ErrDependencyUnavailable))
		return
	}

	req, err := decodeInternalJobSyncRequest(r)
	if err != nil {
		writeError(ctx, w, err)
		return
	}

	result, err := h.jobOrchestrator.RunScheduleSync(ctx, usecase.JobSyncInput{
		LeagueID: req.LeagueID,
		Force:    req.Force,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "run sync schedule job failed", "league_id", req.LeagueID, "force", req.Force, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, result)
}

func (h *Handler) RunBootstrapJob(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.RunBootstrapJob")
	defer span.End()

	if h.jobOrchestrator == nil {
		writeError(ctx, w, fmt.Errorf("%w: job orchestrator is not configured", usecase.ErrDependencyUnavailable))
		return
	}

	req, err := decodeInternalJobSyncRequest(r)
	if err != nil {
		writeError(ctx, w, err)
		return
	}

	result, err := h.jobOrchestrator.Bootstrap(ctx, usecase.JobSyncInput{
		LeagueID: req.LeagueID,
		Force:    req.Force,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "run bootstrap job failed", "league_id", req.LeagueID, "force", req.Force, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, result)
}

func (h *Handler) RunSyncLiveJob(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.RunSyncLiveJob")
	defer span.End()

	if h.jobOrchestrator == nil {
		writeError(ctx, w, fmt.Errorf("%w: job orchestrator is not configured", usecase.ErrDependencyUnavailable))
		return
	}

	req, err := decodeInternalJobSyncRequest(r)
	if err != nil {
		writeError(ctx, w, err)
		return
	}

	result, err := h.jobOrchestrator.RunLiveSync(ctx, usecase.JobSyncInput{
		LeagueID: req.LeagueID,
		Force:    req.Force,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "run sync live job failed", "league_id", req.LeagueID, "force", req.Force, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, result)
}

func decodeInternalJobSyncRequest(r *http.Request) (internalJobSyncRequest, error) {
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req internalJobSyncRequest
	if err := decoder.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return internalJobSyncRequest{}, nil
		}
		return internalJobSyncRequest{}, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err)
	}

	return req, nil
}
