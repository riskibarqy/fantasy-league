CREATE TABLE job_dispatches (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    dispatch_id TEXT NOT NULL,
    job_name TEXT NOT NULL,
    job_path TEXT NOT NULL,
    league_public_id TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL,
    sent_at timestamptz,
    completed_at timestamptz,
    failed_at timestamptz,
    last_error TEXT,
    sent_trace_id TEXT,
    sent_span_id TEXT,
    completed_trace_id TEXT,
    completed_span_id TEXT,
    failed_trace_id TEXT,
    failed_span_id TEXT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT ck_job_dispatches_status CHECK (status IN ('sent', 'completed', 'failed'))
);

CREATE UNIQUE INDEX uq_job_dispatches_dispatch_id_active
    ON job_dispatches (dispatch_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_job_dispatches_status_created_active
    ON job_dispatches (status, created_at DESC, id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_job_dispatches_league_status_active
    ON job_dispatches (league_public_id, status, created_at DESC, id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_job_dispatches_trace_sent_active
    ON job_dispatches (sent_trace_id, sent_span_id, id)
    WHERE deleted_at IS NULL AND sent_trace_id IS NOT NULL;

CREATE INDEX idx_job_dispatches_trace_completed_active
    ON job_dispatches (completed_trace_id, completed_span_id, id)
    WHERE deleted_at IS NULL AND completed_trace_id IS NOT NULL;

CREATE INDEX idx_job_dispatches_trace_failed_active
    ON job_dispatches (failed_trace_id, failed_span_id, id)
    WHERE deleted_at IS NULL AND failed_trace_id IS NOT NULL;

CREATE TRIGGER trg_job_dispatches_touch_updated_at
    BEFORE UPDATE ON job_dispatches
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
