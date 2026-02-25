package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/riskibarqy/fantasy-league/internal/domain/jobscheduler"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
	"go.opentelemetry.io/otel/trace"
)

var internalJobDispatchUnsafeRegex = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

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
		h.recordInternalJobDispatch(ctx, req, jobscheduler.DispatchEvent{
			JobName:      "sync-schedule",
			JobPath:      "/v1/internal/jobs/sync-schedule",
			LeagueID:     req.LeagueID,
			Status:       jobscheduler.StatusFailed,
			Payload:      buildInternalJobPayload(req),
			ErrorMessage: err.Error(),
			OccurredAt:   time.Now().UTC(),
		})
		h.logger.WarnContext(ctx, "run sync schedule job failed", "league_id", req.LeagueID, "force", req.Force, "error", err)
		writeError(ctx, w, err)
		return
	}
	h.recordInternalJobDispatch(ctx, req, jobscheduler.DispatchEvent{
		JobName:    "sync-schedule",
		JobPath:    "/v1/internal/jobs/sync-schedule",
		LeagueID:   req.LeagueID,
		Status:     jobscheduler.StatusCompleted,
		Payload:    buildInternalJobPayload(req),
		OccurredAt: time.Now().UTC(),
	})

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
		h.recordInternalJobDispatch(ctx, req, jobscheduler.DispatchEvent{
			JobName:      "bootstrap",
			JobPath:      "/v1/internal/jobs/bootstrap",
			LeagueID:     req.LeagueID,
			Status:       jobscheduler.StatusFailed,
			Payload:      buildInternalJobPayload(req),
			ErrorMessage: err.Error(),
			OccurredAt:   time.Now().UTC(),
		})
		h.logger.WarnContext(ctx, "run bootstrap job failed", "league_id", req.LeagueID, "force", req.Force, "error", err)
		writeError(ctx, w, err)
		return
	}
	h.recordInternalJobDispatch(ctx, req, jobscheduler.DispatchEvent{
		JobName:    "bootstrap",
		JobPath:    "/v1/internal/jobs/bootstrap",
		LeagueID:   req.LeagueID,
		Status:     jobscheduler.StatusCompleted,
		Payload:    buildInternalJobPayload(req),
		OccurredAt: time.Now().UTC(),
	})

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
		h.recordInternalJobDispatch(ctx, req, jobscheduler.DispatchEvent{
			JobName:      "sync-live",
			JobPath:      "/v1/internal/jobs/sync-live",
			LeagueID:     req.LeagueID,
			Status:       jobscheduler.StatusFailed,
			Payload:      buildInternalJobPayload(req),
			ErrorMessage: err.Error(),
			OccurredAt:   time.Now().UTC(),
		})
		h.logger.WarnContext(ctx, "run sync live job failed", "league_id", req.LeagueID, "force", req.Force, "error", err)
		writeError(ctx, w, err)
		return
	}
	h.recordInternalJobDispatch(ctx, req, jobscheduler.DispatchEvent{
		JobName:    "sync-live",
		JobPath:    "/v1/internal/jobs/sync-live",
		LeagueID:   req.LeagueID,
		Status:     jobscheduler.StatusCompleted,
		Payload:    buildInternalJobPayload(req),
		OccurredAt: time.Now().UTC(),
	})

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

func (h *Handler) recordInternalJobDispatch(ctx context.Context, req internalJobSyncRequest, event jobscheduler.DispatchEvent) {
	if h.jobDispatchRepo == nil {
		return
	}

	dispatchID := strings.TrimSpace(req.DispatchID)
	if dispatchID == "" {
		dispatchID = buildManualDispatchID(event.JobName, req.LeagueID, event.OccurredAt)
	}
	event.DispatchID = dispatchID

	traceID, spanID := traceMetaFromContext(ctx)
	event.TraceID = traceID
	event.SpanID = spanID

	if err := h.jobDispatchRepo.UpsertEvent(ctx, event); err != nil {
		h.logger.WarnContext(ctx, "record internal job dispatch failed",
			"dispatch_id", event.DispatchID,
			"job_name", event.JobName,
			"status", event.Status,
			"error", err,
		)
	}
}

func buildInternalJobPayload(req internalJobSyncRequest) map[string]any {
	payload := map[string]any{
		"league_id": req.LeagueID,
		"force":     req.Force,
	}
	if strings.TrimSpace(req.DispatchID) != "" {
		payload["dispatch_id"] = req.DispatchID
	}
	return payload
}

func buildManualDispatchID(jobName, leagueID string, now time.Time) string {
	jobName = sanitizeDispatchPart(jobName)
	leagueID = sanitizeDispatchPart(leagueID)
	ts := now.UTC().Format("20060102T150405.000000000Z")
	return "manual-" + jobName + "-" + leagueID + "-" + ts
}

func sanitizeDispatchPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return internalJobDispatchUnsafeRegex.ReplaceAllString(value, "-")
}

func traceMetaFromContext(ctx context.Context) (string, string) {
	spanContext := trace.SpanFromContext(ctx).SpanContext()
	if !spanContext.IsValid() {
		return "", ""
	}
	return spanContext.TraceID().String(), spanContext.SpanID().String()
}
