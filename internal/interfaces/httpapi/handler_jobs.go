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

	sonic "github.com/bytedance/sonic"
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

func (h *Handler) RunSyncScheduleDirect(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.RunSyncScheduleDirect")
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

	result, err := h.jobOrchestrator.RunScheduleSyncDirect(ctx, usecase.JobSyncInput{
		LeagueID: req.LeagueID,
		Force:    req.Force,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "run direct sync schedule failed", "league_id", req.LeagueID, "force", req.Force, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, result)
}

func (h *Handler) RunResync(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.RunResync")
	defer span.End()

	if h.sportDataSyncService == nil {
		writeError(ctx, w, fmt.Errorf("%w: sport data sync service is not configured", usecase.ErrDependencyUnavailable))
		return
	}

	req, err := decodeResyncRequest(r)
	if err != nil {
		writeError(ctx, w, err)
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}
	if len(req.SyncData) == 0 {
		writeError(ctx, w, fmt.Errorf("%w: sync_data is required", usecase.ErrInvalidInput))
		return
	}
	if strings.TrimSpace(req.LeagueID) == "" && req.SeasonID <= 0 {
		writeError(ctx, w, fmt.Errorf("%w: league_id or season_id is required", usecase.ErrInvalidInput))
		return
	}

	result, err := h.sportDataSyncService.Resync(ctx, usecase.ResyncInput{
		LeagueID:   req.LeagueID,
		SeasonID:   req.SeasonID,
		SyncData:   req.SyncData,
		MaxWorkers: req.MaxWorkers,
		Gameweeks:  req.Gameweeks,
		DryRun:     req.DryRun,
	})
	if err != nil {
		h.logger.WarnContext(ctx,
			"run resync failed",
			"league_id", req.LeagueID,
			"season_id", req.SeasonID,
			"sync_data", req.SyncData,
			"max_workers", req.MaxWorkers,
			"error", err,
		)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, result)
}

// RunSyncMasterData synchronizes season master data (teams, players, stat types).
// Intended to be executed once at new season bootstrap, with optional dry-run.
func (h *Handler) RunSyncMasterData(w http.ResponseWriter, r *http.Request) {
	h.runSyncPreset(w, r, "master-data", []string{"team", "players", "stat_types"})
}

// RunSyncTeamSchedule synchronizes fixture schedule data for teams.
// Useful for refreshing calendar/timeline and fixture-event consistency.
func (h *Handler) RunSyncTeamSchedule(w http.ResponseWriter, r *http.Request) {
	h.runSyncPreset(w, r, "team-schedule", []string{"fixtures"})
}

// RunSyncReconcile repairs data mismatches by re-syncing fixtures, standings,
// and fixture-derived stats. Supports narrowing by gameweeks.
func (h *Handler) RunSyncReconcile(w http.ResponseWriter, r *http.Request) {
	h.runSyncPreset(w, r, "reconcile", []string{"fixtures", "standing", "team_fixtures", "player_fixture_stats"})
}

// GetSyncRun returns execution result payload for a previous sync run id.
func (h *Handler) GetSyncRun(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetSyncRun")
	defer span.End()

	runID := strings.TrimSpace(r.PathValue("runID"))
	if runID == "" {
		writeError(ctx, w, fmt.Errorf("%w: run id is required", usecase.ErrInvalidInput))
		return
	}

	record, ok := h.loadSyncRun(runID)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: sync run=%s", usecase.ErrNotFound, runID))
		return
	}

	writeSuccess(ctx, w, http.StatusOK, record)
}

// runSyncPreset executes a pre-defined sync mode and stores result by run id.
func (h *Handler) runSyncPreset(w http.ResponseWriter, r *http.Request, mode string, defaultSyncData []string) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.runSyncPreset")
	defer span.End()

	if h.sportDataSyncService == nil {
		writeError(ctx, w, fmt.Errorf("%w: sport data sync service is not configured", usecase.ErrDependencyUnavailable))
		return
	}

	req, err := decodeResyncRequest(r)
	if err != nil {
		writeError(ctx, w, err)
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	syncData := defaultSyncData
	if mode == "reconcile" && len(req.SyncData) > 0 {
		syncData = req.SyncData
	}
	if strings.TrimSpace(req.LeagueID) == "" && req.SeasonID <= 0 {
		writeError(ctx, w, fmt.Errorf("%w: league_id or season_id is required", usecase.ErrInvalidInput))
		return
	}

	runID := buildSyncRunID(mode, req.LeagueID, req.SeasonID, time.Now().UTC())
	result, err := h.sportDataSyncService.Resync(ctx, usecase.ResyncInput{
		LeagueID:   req.LeagueID,
		SeasonID:   req.SeasonID,
		SyncData:   syncData,
		MaxWorkers: req.MaxWorkers,
		Gameweeks:  req.Gameweeks,
		DryRun:     req.DryRun,
	})
	if err != nil {
		h.logger.WarnContext(ctx,
			"run sync preset failed",
			"mode", mode,
			"run_id", runID,
			"league_id", req.LeagueID,
			"season_id", req.SeasonID,
			"sync_data", syncData,
			"gameweeks", req.Gameweeks,
			"dry_run", req.DryRun,
			"error", err,
		)
		writeError(ctx, w, err)
		return
	}

	record := syncRunRecord{
		RunID:      runID,
		Mode:       mode,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339Nano),
		LeagueID:   req.LeagueID,
		SeasonID:   req.SeasonID,
		SyncData:   append([]string(nil), syncData...),
		Gameweeks:  append([]int(nil), req.Gameweeks...),
		DryRun:     req.DryRun,
		MaxWorkers: req.MaxWorkers,
		Result:     result,
	}
	h.saveSyncRun(record)

	writeSuccess(ctx, w, http.StatusOK, record)
}

func buildSyncRunID(mode, leagueID string, seasonID int64, now time.Time) string {
	mode = sanitizeDispatchPart(mode)
	leagueID = sanitizeDispatchPart(leagueID)
	if seasonID <= 0 {
		seasonID = 0
	}
	return fmt.Sprintf("sync-%s-%s-%d-%s", mode, leagueID, seasonID, now.UTC().Format("20060102T150405.000000000Z"))
}

func (h *Handler) saveSyncRun(record syncRunRecord) {
	if strings.TrimSpace(record.RunID) == "" {
		return
	}

	now := time.Now().UTC()
	h.syncRunsMu.Lock()
	h.pruneSyncRunsLocked(now)
	h.syncRuns[record.RunID] = record
	if h.syncRunMaxEntries > 0 && len(h.syncRuns) > h.syncRunMaxEntries {
		h.evictOldestSyncRunsLocked(len(h.syncRuns) - h.syncRunMaxEntries)
	}
	h.syncRunsMu.Unlock()
}

func (h *Handler) loadSyncRun(runID string) (syncRunRecord, bool) {
	h.syncRunsMu.Lock()
	h.pruneSyncRunsLocked(time.Now().UTC())
	record, ok := h.syncRuns[runID]
	h.syncRunsMu.Unlock()
	return record, ok
}

func (h *Handler) pruneSyncRunsLocked(now time.Time) {
	if h.syncRunRetention <= 0 {
		return
	}
	cutoff := now.Add(-h.syncRunRetention)
	for runID, record := range h.syncRuns {
		if syncRunCreatedAt(record).Before(cutoff) {
			delete(h.syncRuns, runID)
		}
	}
}

func (h *Handler) evictOldestSyncRunsLocked(count int) {
	for i := 0; i < count && len(h.syncRuns) > 0; i++ {
		oldestRunID := ""
		var oldestCreatedAt time.Time
		for runID, record := range h.syncRuns {
			createdAt := syncRunCreatedAt(record)
			if oldestRunID == "" || createdAt.Before(oldestCreatedAt) {
				oldestRunID = runID
				oldestCreatedAt = createdAt
			}
		}
		if oldestRunID == "" {
			return
		}
		delete(h.syncRuns, oldestRunID)
	}
}

func syncRunCreatedAt(record syncRunRecord) time.Time {
	createdAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(record.CreatedAt))
	if err != nil {
		return time.Unix(0, 0).UTC()
	}
	return createdAt
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
	decoder := sonic.ConfigDefault.NewDecoder(r.Body)
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

func decodeResyncRequest(r *http.Request) (resyncRequest, error) {
	decoder := sonic.ConfigDefault.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req resyncRequest
	if err := decoder.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return resyncRequest{}, fmt.Errorf("%w: request body is required", usecase.ErrInvalidInput)
		}
		return resyncRequest{}, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err)
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

type resyncRequest struct {
	LeagueID string `json:"league_id" validate:"omitempty"`
	SeasonID int64  `json:"season_id" validate:"omitempty,gt=0"`
	// Optional override. For reconcile mode, defaults are applied when omitted.
	SyncData   []string `json:"sync_data" validate:"omitempty,min=1,dive,required"`
	MaxWorkers int      `json:"max_workers" validate:"omitempty,gte=1,lte=2"`
	// Optional fixture scope for fixture-related sync kinds.
	Gameweeks []int `json:"gameweeks" validate:"omitempty,min=1,dive,gt=0"`
	// DryRun validates and computes rows without writing to DB.
	DryRun bool `json:"dry_run"`
}

type syncRunRecord struct {
	RunID      string               `json:"run_id"`
	Mode       string               `json:"mode"`
	CreatedAt  string               `json:"created_at"`
	LeagueID   string               `json:"league_id,omitempty"`
	SeasonID   int64                `json:"season_id,omitempty"`
	SyncData   []string             `json:"sync_data"`
	Gameweeks  []int                `json:"gameweeks,omitempty"`
	DryRun     bool                 `json:"dry_run"`
	MaxWorkers int                  `json:"max_workers"`
	Result     usecase.ResyncResult `json:"result"`
}
