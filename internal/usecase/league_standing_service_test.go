package usecase

import (
	"context"
	"testing"

	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/leaguestanding"
)

func TestLeagueStandingService_ListByLeague_FallbackFromFixtures(t *testing.T) {
	t.Parallel()

	const leagueID = "idn-liga-1-2025"
	repo := &stubLeagueRepository{
		byID: map[string]league.League{
			leagueID: {ID: leagueID, Name: "Liga 1"},
		},
	}
	standingsRepo := &stubLeagueStandingRepository{
		rows: map[string][]leaguestanding.Standing{
			standingsKey(leagueID, false): {},
		},
	}

	homeA := 2
	awayB := 1
	homeA2 := 1
	awayC := 1
	homeB := 0
	awayC2 := 3
	fixturesRepo := &stubFixtureRepository{
		byLeague: map[string][]fixture.Fixture{
			leagueID: {
				{
					ID:         "f1",
					LeagueID:   leagueID,
					HomeTeamID: "team-a",
					AwayTeamID: "team-b",
					HomeScore:  &homeA,
					AwayScore:  &awayB,
					Status:     fixture.StatusFinished,
				},
				{
					ID:         "f2",
					LeagueID:   leagueID,
					HomeTeamID: "team-a",
					AwayTeamID: "team-c",
					HomeScore:  &homeA2,
					AwayScore:  &awayC,
					Status:     "FT",
				},
				{
					ID:         "f3",
					LeagueID:   leagueID,
					HomeTeamID: "team-b",
					AwayTeamID: "team-c",
					HomeScore:  &homeB,
					AwayScore:  &awayC2,
					Status:     fixture.StatusFinished,
				},
			},
		},
	}

	service := NewLeagueStandingService(repo, standingsRepo, fixturesRepo)

	got, err := service.ListByLeague(context.Background(), leagueID, false)
	if err != nil {
		t.Fatalf("ListByLeague error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(got))
	}

	if got[0].TeamID != "team-c" || got[0].Points != 4 || got[0].Position != 1 {
		t.Fatalf("unexpected rank 1 row: %+v", got[0])
	}
	if got[1].TeamID != "team-a" || got[1].Points != 4 || got[1].Position != 2 {
		t.Fatalf("unexpected rank 2 row: %+v", got[1])
	}
	if got[2].TeamID != "team-b" || got[2].Points != 0 || got[2].Position != 3 {
		t.Fatalf("unexpected rank 3 row: %+v", got[2])
	}
}

func TestLeagueStandingService_ListByLeague_UsesStoredRowsWhenAvailable(t *testing.T) {
	t.Parallel()

	const leagueID = "idn-liga-1-2025"
	repo := &stubLeagueRepository{
		byID: map[string]league.League{
			leagueID: {ID: leagueID, Name: "Liga 1"},
		},
	}
	standingsRepo := &stubLeagueStandingRepository{
		rows: map[string][]leaguestanding.Standing{
			standingsKey(leagueID, false): {
				{
					LeagueID: leagueID,
					TeamID:   "team-a",
					Position: 1,
					Points:   99,
				},
			},
		},
	}
	fixturesRepo := &stubFixtureRepository{
		byLeague: map[string][]fixture.Fixture{
			leagueID: {
				{
					ID:         "f1",
					LeagueID:   leagueID,
					HomeTeamID: "team-b",
					AwayTeamID: "team-c",
					Status:     fixture.StatusFinished,
				},
			},
		},
	}

	service := NewLeagueStandingService(repo, standingsRepo, fixturesRepo)

	got, err := service.ListByLeague(context.Background(), leagueID, false)
	if err != nil {
		t.Fatalf("ListByLeague error: %v", err)
	}
	if len(got) != 1 || got[0].TeamID != "team-a" || got[0].Points != 99 {
		t.Fatalf("expected stored standings to be used, got=%+v", got)
	}
}

type stubLeagueRepository struct {
	byID map[string]league.League
}

func (s *stubLeagueRepository) List(_ context.Context) ([]league.League, error) {
	out := make([]league.League, 0, len(s.byID))
	for _, item := range s.byID {
		out = append(out, item)
	}
	return out, nil
}

func (s *stubLeagueRepository) GetByID(_ context.Context, leagueID string) (league.League, bool, error) {
	item, ok := s.byID[leagueID]
	return item, ok, nil
}

type stubLeagueStandingRepository struct {
	rows map[string][]leaguestanding.Standing
}

func (s *stubLeagueStandingRepository) ListByLeague(_ context.Context, leagueID string, live bool) ([]leaguestanding.Standing, error) {
	items := s.rows[standingsKey(leagueID, live)]
	out := make([]leaguestanding.Standing, len(items))
	copy(out, items)
	return out, nil
}

func (s *stubLeagueStandingRepository) ReplaceByLeague(_ context.Context, leagueID string, live bool, rows []leaguestanding.Standing) error {
	out := make([]leaguestanding.Standing, len(rows))
	copy(out, rows)
	s.rows[standingsKey(leagueID, live)] = out
	return nil
}

type stubFixtureRepository struct {
	byLeague map[string][]fixture.Fixture
}

func (s *stubFixtureRepository) ListByLeague(_ context.Context, leagueID string) ([]fixture.Fixture, error) {
	items := s.byLeague[leagueID]
	out := make([]fixture.Fixture, len(items))
	copy(out, items)
	return out, nil
}

func (s *stubFixtureRepository) GetByID(_ context.Context, leagueID, fixtureID string) (fixture.Fixture, bool, error) {
	for _, item := range s.byLeague[leagueID] {
		if item.ID == fixtureID {
			return item, true, nil
		}
	}
	return fixture.Fixture{}, false, nil
}

func standingsKey(leagueID string, live bool) string {
	if live {
		return leagueID + "|live"
	}
	return leagueID + "|final"
}
