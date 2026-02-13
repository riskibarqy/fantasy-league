package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
	leaguemock "github.com/riskibarqy/fantasy-league/internal/mocks/domain/league"
	teammock "github.com/riskibarqy/fantasy-league/internal/mocks/domain/team"
	"github.com/stretchr/testify/mock"
)

func TestLeagueService_ListTeamsByLeague_SuccessUsingMockery(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "trace_id", "trace-456")
	leagueRepo := leaguemock.NewRepository(t)
	teamRepo := teammock.NewRepository(t)

	service := NewLeagueService(leagueRepo, teamRepo)
	leagueID := "idn-liga-1-2025"
	expectedTeams := []team.Team{
		{ID: "idn-persija", LeagueID: leagueID, Name: "Persija Jakarta", Short: "PSJ"},
		{ID: "idn-persib", LeagueID: leagueID, Name: "Persib Bandung", Short: "PSB"},
	}

	leagueRepo.
		On("GetByID", mock.MatchedBy(func(v context.Context) bool { return v == ctx }), leagueID).
		Return(league.League{ID: leagueID}, true, nil).
		Once()
	teamRepo.
		On("ListByLeague", mock.MatchedBy(func(v context.Context) bool { return v == ctx }), leagueID).
		Return(expectedTeams, nil).
		Once()

	got, err := service.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		t.Fatalf("list teams by league: %v", err)
	}
	if len(got) != len(expectedTeams) {
		t.Fatalf("unexpected team count: got=%d want=%d", len(got), len(expectedTeams))
	}
	if got[0].ID != expectedTeams[0].ID {
		t.Fatalf("unexpected team id: got=%s want=%s", got[0].ID, expectedTeams[0].ID)
	}
}

func TestLeagueService_ListTeamsByLeague_LeagueNotFoundUsingMockery(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	leagueRepo := leaguemock.NewRepository(t)
	teamRepo := teammock.NewRepository(t)

	service := NewLeagueService(leagueRepo, teamRepo)
	leagueID := "missing-league"

	leagueRepo.
		On("GetByID", mock.MatchedBy(func(v context.Context) bool { return v == ctx }), leagueID).
		Return(league.League{}, false, nil).
		Once()

	_, err := service.ListTeamsByLeague(ctx, leagueID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
