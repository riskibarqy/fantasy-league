package usecase

import (
	"testing"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
)

func TestResolveDashboardGameweek(t *testing.T) {
	now := time.Date(2026, time.February, 22, 12, 0, 0, 0, time.UTC)

	t.Run("prefers live gameweek", func(t *testing.T) {
		items := []fixture.Fixture{
			{Gameweek: 4, Status: fixture.StatusScheduled, KickoffAt: now.Add(24 * time.Hour)},
			{Gameweek: 3, Status: fixture.StatusLive, KickoffAt: now.Add(-30 * time.Minute)},
		}

		got := resolveDashboardGameweek(items, now)
		if got != 3 {
			t.Fatalf("unexpected gameweek: got=%d want=3", got)
		}
	})

	t.Run("uses nearest upcoming when no live", func(t *testing.T) {
		items := []fixture.Fixture{
			{Gameweek: 2, Status: fixture.StatusFinished, KickoffAt: now.Add(-24 * time.Hour)},
			{Gameweek: 3, Status: fixture.StatusScheduled, KickoffAt: now.Add(2 * time.Hour)},
			{Gameweek: 4, Status: fixture.StatusScheduled, KickoffAt: now.Add(48 * time.Hour)},
		}

		got := resolveDashboardGameweek(items, now)
		if got != 3 {
			t.Fatalf("unexpected gameweek: got=%d want=3", got)
		}
	})

	t.Run("falls back to latest completed when no active or upcoming", func(t *testing.T) {
		items := []fixture.Fixture{
			{Gameweek: 2, Status: fixture.StatusFinished, KickoffAt: now.Add(-72 * time.Hour)},
			{Gameweek: 5, Status: fixture.StatusFinished, KickoffAt: now.Add(-2 * time.Hour)},
		}

		got := resolveDashboardGameweek(items, now)
		if got != 5 {
			t.Fatalf("unexpected gameweek: got=%d want=5", got)
		}
	})
}
