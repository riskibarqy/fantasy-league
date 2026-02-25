package usecase

import (
	"strings"
	"testing"
	"time"
)

func TestDedupKey_UsesQStashSafeFormat(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, time.February, 25, 4, 25, 42, 0, time.UTC)
	got := dedupKey("sync-live", "idn:liga/1 2025", at, 5*time.Minute)

	if strings.Contains(got, ":") {
		t.Fatalf("dedup key must not contain colon, got=%q", got)
	}

	want := "sync-live-idn-liga-1-2025-20260225T042500Z"
	if got != want {
		t.Fatalf("unexpected dedup key: got=%q want=%q", got, want)
	}
}

func TestSanitizeDedupSegment_EmptyFallback(t *testing.T) {
	t.Parallel()

	if got := sanitizeDedupSegment(" \t "); got != "unknown" {
		t.Fatalf("unexpected sanitize fallback: got=%q want=%q", got, "unknown")
	}
}

func TestNextScheduleDelay_WithUpcomingFixture(t *testing.T) {
	t.Parallel()

	svc := &JobOrchestratorService{
		cfg: JobOrchestratorConfig{
			ScheduleInterval: 15 * time.Minute,
			LiveInterval:     5 * time.Minute,
			PreKickoffLead:   15 * time.Minute,
		},
	}

	now := time.Date(2026, time.February, 25, 10, 0, 0, 0, time.UTC)
	upcoming := now.Add(2 * time.Hour) // kickoff at 12:00

	got := svc.nextScheduleDelay(now, false, &upcoming)
	want := time.Hour + 45*time.Minute // sync schedule at 11:45
	if got != want {
		t.Fatalf("unexpected schedule delay: got=%s want=%s", got, want)
	}
}

func TestNextScheduleDelay_NoUpcomingFixture(t *testing.T) {
	t.Parallel()

	svc := &JobOrchestratorService{
		cfg: JobOrchestratorConfig{
			ScheduleInterval: 15 * time.Minute,
			LiveInterval:     5 * time.Minute,
			PreKickoffLead:   15 * time.Minute,
		},
	}

	got := svc.nextScheduleDelay(time.Now().UTC(), false, nil)
	want := 6 * time.Hour
	if got != want {
		t.Fatalf("unexpected schedule delay without upcoming fixture: got=%s want=%s", got, want)
	}
}
