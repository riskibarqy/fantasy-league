package app

import (
	"strings"
	"testing"
)

func TestNormalizeDBURL(t *testing.T) {
	t.Run("appends flag by default", func(t *testing.T) {
		got := normalizeDBURL("postgres://user:pass@localhost:5432/dbname?sslmode=disable", true)
		want := "disable_prepared_binary_result=yes"
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in url, got %q", want, got)
		}
	})

	t.Run("keeps explicit value", func(t *testing.T) {
		in := "postgres://user:pass@localhost:5432/dbname?sslmode=disable&disable_prepared_binary_result=no"
		got := normalizeDBURL(in, true)
		if got != in {
			t.Fatalf("expected url unchanged, got %q", got)
		}
	})

	t.Run("toggle off keeps url unchanged", func(t *testing.T) {
		in := "postgres://user:pass@localhost:5432/dbname?sslmode=disable"
		got := normalizeDBURL(in, false)
		if got != in {
			t.Fatalf("expected url unchanged, got %q", got)
		}
	})
}
