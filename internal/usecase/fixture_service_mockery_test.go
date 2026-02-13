package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	fixturemock "github.com/riskibarqy/fantasy-league/internal/mocks/domain/fixture"
	leaguemock "github.com/riskibarqy/fantasy-league/internal/mocks/domain/league"
	"github.com/stretchr/testify/mock"
)

func TestFixtureService_ListByLeague_SuccessUsingMockery(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "trace_id", "trace-123")
	leagueRepo := leaguemock.NewRepository(t)
	fixtureRepo := fixturemock.NewRepository(t)

	service := NewFixtureService(leagueRepo, fixtureRepo)
	leagueID := "idn-liga-1-2025"
	expectedFixtures := []fixture.Fixture{
		{
			ID:        "fx-001",
			LeagueID:  leagueID,
			Gameweek:  1,
			HomeTeam:  "Persija Jakarta",
			AwayTeam:  "Persib Bandung",
			KickoffAt: time.Date(2026, 2, 14, 19, 0, 0, 0, time.UTC),
			Venue:     "Jakarta International Stadium",
		},
	}

	leagueRepo.
		On("GetByID", mock.MatchedBy(func(v context.Context) bool { return v == ctx }), leagueID).
		Return(league.League{ID: leagueID}, true, nil).
		Once()
	fixtureRepo.
		On("ListByLeague", mock.MatchedBy(func(v context.Context) bool { return v == ctx }), leagueID).
		Return(expectedFixtures, nil).
		Once()

	got, err := service.ListByLeague(ctx, leagueID)
	if err != nil {
		t.Fatalf("list fixtures by league: %v", err)
	}
	if len(got) != len(expectedFixtures) {
		t.Fatalf("unexpected fixture count: got=%d want=%d", len(got), len(expectedFixtures))
	}
	if got[0].ID != expectedFixtures[0].ID {
		t.Fatalf("unexpected fixture id: got=%s want=%s", got[0].ID, expectedFixtures[0].ID)
	}
}

func TestFixtureService_ListByLeague_LeagueNotFoundUsingMockery(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	leagueRepo := leaguemock.NewRepository(t)
	fixtureRepo := fixturemock.NewRepository(t)

	service := NewFixtureService(leagueRepo, fixtureRepo)
	leagueID := "missing-league"

	leagueRepo.
		On("GetByID", mock.MatchedBy(func(v context.Context) bool { return v == ctx }), leagueID).
		Return(league.League{}, false, nil).
		Once()

	_, err := service.ListByLeague(ctx, leagueID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
