package postgres

import "time"

type jobDispatchInsertModel struct {
	DispatchID       string     `db:"dispatch_id"`
	JobName          string     `db:"job_name"`
	JobPath          string     `db:"job_path"`
	LeagueID         string     `db:"league_public_id"`
	Payload          string     `db:"payload"`
	Status           string     `db:"status"`
	SentAt           *time.Time `db:"sent_at"`
	CompletedAt      *time.Time `db:"completed_at"`
	FailedAt         *time.Time `db:"failed_at"`
	LastError        *string    `db:"last_error"`
	SentTraceID      *string    `db:"sent_trace_id"`
	SentSpanID       *string    `db:"sent_span_id"`
	CompletedTraceID *string    `db:"completed_trace_id"`
	CompletedSpanID  *string    `db:"completed_span_id"`
	FailedTraceID    *string    `db:"failed_trace_id"`
	FailedSpanID     *string    `db:"failed_span_id"`
}
