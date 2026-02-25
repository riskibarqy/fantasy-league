package jobscheduler

import "time"

type DispatchStatus string

const (
	StatusSent      DispatchStatus = "sent"
	StatusCompleted DispatchStatus = "completed"
	StatusFailed    DispatchStatus = "failed"
)

type DispatchEvent struct {
	DispatchID   string
	JobName      string
	JobPath      string
	LeagueID     string
	Status       DispatchStatus
	Payload      map[string]any
	ErrorMessage string
	OccurredAt   time.Time
	TraceID      string
	SpanID       string
}
