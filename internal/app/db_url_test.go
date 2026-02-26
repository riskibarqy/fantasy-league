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

func TestDBNameFromURL(t *testing.T) {
	t.Run("url style", func(t *testing.T) {
		got := dbNameFromURL("postgres://user:pass@localhost:5432/fantasy_league?sslmode=disable")
		if got != "fantasy_league" {
			t.Fatalf("unexpected db name: %q", got)
		}
	})

	t.Run("dsn style", func(t *testing.T) {
		got := dbNameFromURL("host=localhost user=postgres dbname=fantasy_league sslmode=disable")
		if got != "fantasy_league" {
			t.Fatalf("unexpected db name: %q", got)
		}
	})
}

func TestFormatDBQueryForTrace(t *testing.T) {
	got := formatDBQueryForTrace(" SELECT   *\nFROM fixtures \t WHERE league_public_id = $1 ")
	want := "SELECT * FROM fixtures WHERE league_public_id = $1"
	if got != want {
		t.Fatalf("unexpected formatted query: %q", got)
	}
}
