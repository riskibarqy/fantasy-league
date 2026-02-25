package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	jsoniter "github.com/json-iterator/go"
	"github.com/riskibarqy/fantasy-league/internal/domain/jobscheduler"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type JobDispatchRepository struct {
	db *sqlx.DB
}

func NewJobDispatchRepository(db *sqlx.DB) *JobDispatchRepository {
	return &JobDispatchRepository{db: db}
}

func (r *JobDispatchRepository) UpsertEvent(ctx context.Context, event jobscheduler.DispatchEvent) error {
	dispatchID := strings.TrimSpace(event.DispatchID)
	if dispatchID == "" {
		return fmt.Errorf("dispatch id is required")
	}

	jobName := strings.TrimSpace(event.JobName)
	if jobName == "" {
		jobName = "unknown"
	}
	jobPath := strings.TrimSpace(event.JobPath)
	if jobPath == "" {
		jobPath = "/unknown"
	}
	leagueID := strings.TrimSpace(event.LeagueID)
	if leagueID == "" {
		leagueID = "unknown"
	}

	occurredAt := event.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	payloadJSON, err := marshalPayload(event.Payload)
	if err != nil {
		return fmt.Errorf("marshal job dispatch payload: %w", err)
	}

	model := jobDispatchInsertModel{
		DispatchID: dispatchID,
		JobName:    jobName,
		JobPath:    jobPath,
		LeagueID:   leagueID,
		Payload:    payloadJSON,
		Status:     string(event.Status),
		LastError:  optionalString(event.ErrorMessage),
	}

	switch event.Status {
	case jobscheduler.StatusSent:
		model.SentAt = &occurredAt
		model.SentTraceID = optionalString(event.TraceID)
		model.SentSpanID = optionalString(event.SpanID)
		model.LastError = nil
	case jobscheduler.StatusCompleted:
		model.CompletedAt = &occurredAt
		model.CompletedTraceID = optionalString(event.TraceID)
		model.CompletedSpanID = optionalString(event.SpanID)
		model.LastError = nil
	case jobscheduler.StatusFailed:
		model.FailedAt = &occurredAt
		model.FailedTraceID = optionalString(event.TraceID)
		model.FailedSpanID = optionalString(event.SpanID)
	}

	query, args, err := qb.InsertModel("job_dispatches", model, `ON CONFLICT (dispatch_id) WHERE deleted_at IS NULL
DO UPDATE SET
    job_name = EXCLUDED.job_name,
    job_path = EXCLUDED.job_path,
    league_public_id = EXCLUDED.league_public_id,
    payload = EXCLUDED.payload,
    status = EXCLUDED.status,
    sent_at = CASE
        WHEN EXCLUDED.status = 'sent' THEN EXCLUDED.sent_at
        ELSE COALESCE(job_dispatches.sent_at, EXCLUDED.sent_at)
    END,
    completed_at = CASE
        WHEN EXCLUDED.status = 'completed' THEN EXCLUDED.completed_at
        ELSE job_dispatches.completed_at
    END,
    failed_at = CASE
        WHEN EXCLUDED.status = 'failed' THEN EXCLUDED.failed_at
        WHEN EXCLUDED.status = 'completed' THEN NULL
        ELSE job_dispatches.failed_at
    END,
    last_error = CASE
        WHEN EXCLUDED.status = 'failed' THEN EXCLUDED.last_error
        ELSE NULL
    END,
    sent_trace_id = CASE
        WHEN EXCLUDED.status = 'sent' THEN EXCLUDED.sent_trace_id
        ELSE job_dispatches.sent_trace_id
    END,
    sent_span_id = CASE
        WHEN EXCLUDED.status = 'sent' THEN EXCLUDED.sent_span_id
        ELSE job_dispatches.sent_span_id
    END,
    completed_trace_id = CASE
        WHEN EXCLUDED.status = 'completed' THEN EXCLUDED.completed_trace_id
        ELSE job_dispatches.completed_trace_id
    END,
    completed_span_id = CASE
        WHEN EXCLUDED.status = 'completed' THEN EXCLUDED.completed_span_id
        ELSE job_dispatches.completed_span_id
    END,
    failed_trace_id = CASE
        WHEN EXCLUDED.status = 'failed' THEN EXCLUDED.failed_trace_id
        ELSE job_dispatches.failed_trace_id
    END,
    failed_span_id = CASE
        WHEN EXCLUDED.status = 'failed' THEN EXCLUDED.failed_span_id
        ELSE job_dispatches.failed_span_id
    END,
    deleted_at = NULL`)
	if err != nil {
		return fmt.Errorf("build upsert job dispatch query: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert job dispatch dispatch_id=%s status=%s: %w", dispatchID, event.Status, err)
	}

	return nil
}

func marshalPayload(payload map[string]any) (string, error) {
	if len(payload) == 0 {
		return "{}", nil
	}
	raw, err := jsoniter.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
