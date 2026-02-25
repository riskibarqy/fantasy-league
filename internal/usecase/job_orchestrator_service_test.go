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
