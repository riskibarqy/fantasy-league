package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	"github.com/riskibarqy/fantasy-league/internal/domain/playerstats"
	"github.com/riskibarqy/fantasy-league/internal/domain/scoring"
)

func TestScoringService_GetUserSeasonPointsSummary(t *testing.T) {
	t.Parallel()

	const (
		leagueID = "idn-liga-1-2025"
		userID   = "user-1"
	)

	scoringRepo := &stubPointsScoringRepository{
		userRows: []scoring.UserGameweekPoints{
			{LeagueID: leagueID, Gameweek: 1, UserID: userID, Points: 42},
			{LeagueID: leagueID, Gameweek: 2, UserID: userID, Points: 58},
			{LeagueID: leagueID, Gameweek: 2, UserID: "user-2", Points: 61},
		},
	}

	service := NewScoringService(nil, nil, nil, &stubPointsPlayerStatsRepository{}, nil, scoringRepo)
	service.markEnsure(leagueID, time.Now().UTC())

	got, err := service.GetUserSeasonPointsSummary(context.Background(), leagueID, userID)
	if err != nil {
		t.Fatalf("GetUserSeasonPointsSummary error: %v", err)
	}

	if got.TotalPoints != 100 {
		t.Fatalf("unexpected total points: got=%d want=100", got.TotalPoints)
	}
	if got.HighestPoints != 58 {
		t.Fatalf("unexpected highest points: got=%d want=58", got.HighestPoints)
	}
	if got.Gameweeks != 2 {
		t.Fatalf("unexpected gameweeks: got=%d want=2", got.Gameweeks)
	}
	if got.AveragePoints != 50 {
		t.Fatalf("unexpected average points: got=%v want=50", got.AveragePoints)
	}
}

func TestScoringService_ListUserPlayerPointsByLeague_FilteredGameweek(t *testing.T) {
	t.Parallel()

	const (
		leagueID = "idn-liga-1-2025"
		userID   = "user-1"
	)

	lineupGw2 := lineup.Lineup{
		UserID:        userID,
		LeagueID:      leagueID,
		GoalkeeperID:  "gk1",
		DefenderIDs:   []string{"def1"},
		MidfielderIDs: []string{"mid1"},
		ForwardIDs:    []string{"fwd1"},
		SubstituteIDs: []string{"sub1", "sub2", "sub3", "sub4"},
		CaptainID:     "mid1",
		ViceCaptainID: "fwd1",
	}

	scoringRepo := &stubPointsScoringRepository{
		userRows: []scoring.UserGameweekPoints{
			{LeagueID: leagueID, Gameweek: 2, UserID: userID, Points: 19},
		},
		lineupsByGameweek: map[int]lineup.Lineup{
			2: lineupGw2,
		},
	}

	playerStatsRepo := &stubPointsPlayerStatsRepository{
		pointsByGameweek: map[int]map[string]int{
			2: {
				"gk1":  2,
				"def1": 3,
				"mid1": 5,
				"fwd1": 4,
				"sub1": 10,
			},
		},
	}

	service := NewScoringService(nil, nil, nil, playerStatsRepo, nil, scoringRepo)
	service.markEnsure(leagueID, time.Now().UTC())

	gameweek := 2
	got, err := service.ListUserPlayerPointsByLeague(context.Background(), leagueID, userID, &gameweek)
	if err != nil {
		t.Fatalf("ListUserPlayerPointsByLeague error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("unexpected rows length: got=%d want=1", len(got))
	}
	if got[0].Gameweek != 2 {
		t.Fatalf("unexpected gameweek: got=%d want=2", got[0].Gameweek)
	}
	if got[0].TotalPoints != 19 {
		t.Fatalf("unexpected total points: got=%d want=19", got[0].TotalPoints)
	}

	var captainRow *UserPlayerPoints
	var benchRow *UserPlayerPoints
	for idx := range got[0].Players {
		row := &got[0].Players[idx]
		if row.PlayerID == "mid1" {
			captainRow = row
		}
		if row.PlayerID == "sub1" {
			benchRow = row
		}
	}
	if captainRow == nil {
		t.Fatalf("captain row not found")
	}
	if captainRow.Multiplier != 2 || captainRow.CountedPoints != 10 {
		t.Fatalf("unexpected captain points: multiplier=%d counted=%d", captainRow.Multiplier, captainRow.CountedPoints)
	}
	if benchRow == nil {
		t.Fatalf("bench row not found")
	}
	if benchRow.CountedPoints != 0 {
		t.Fatalf("bench points must not be counted, got=%d", benchRow.CountedPoints)
	}
}

type stubPointsScoringRepository struct {
	userRows          []scoring.UserGameweekPoints
	lineupsByGameweek map[int]lineup.Lineup
}

func (s *stubPointsScoringRepository) GetGameweekLock(_ context.Context, _ string, _ int) (scoring.GameweekLock, bool, error) {
	return scoring.GameweekLock{}, false, nil
}

func (s *stubPointsScoringRepository) ListGameweekLocksByLeague(_ context.Context, _ string) ([]scoring.GameweekLock, error) {
	return nil, nil
}

func (s *stubPointsScoringRepository) UpsertGameweekLock(_ context.Context, _ scoring.GameweekLock) error {
	return nil
}

func (s *stubPointsScoringRepository) GetSquadSnapshot(_ context.Context, _ string, _ int, _ string) (scoring.SquadSnapshot, bool, error) {
	return scoring.SquadSnapshot{}, false, nil
}

func (s *stubPointsScoringRepository) UpsertSquadSnapshot(_ context.Context, _ scoring.SquadSnapshot) error {
	return nil
}

func (s *stubPointsScoringRepository) GetLineupSnapshot(_ context.Context, leagueID string, gameweek int, userID string) (scoring.LineupSnapshot, bool, error) {
	item, ok := s.lineupsByGameweek[gameweek]
	if !ok || item.UserID != userID {
		return scoring.LineupSnapshot{}, false, nil
	}
	return scoring.LineupSnapshot{
		LeagueID: leagueID,
		Gameweek: gameweek,
		Lineup:   item,
	}, true, nil
}

func (s *stubPointsScoringRepository) UpsertLineupSnapshot(_ context.Context, _ scoring.LineupSnapshot) error {
	return nil
}

func (s *stubPointsScoringRepository) ListLineupSnapshotGameweeksByLeague(_ context.Context, _ string) ([]int, error) {
	return nil, nil
}

func (s *stubPointsScoringRepository) ListLineupSnapshotsByLeagueGameweek(_ context.Context, _ string, _ int) ([]scoring.LineupSnapshot, error) {
	return nil, nil
}

func (s *stubPointsScoringRepository) UpsertUserGameweekPoints(_ context.Context, _ scoring.UserGameweekPoints) error {
	return nil
}

func (s *stubPointsScoringRepository) ListUserGameweekPointsByLeague(_ context.Context, _ string) ([]scoring.UserGameweekPoints, error) {
	out := make([]scoring.UserGameweekPoints, len(s.userRows))
	copy(out, s.userRows)
	return out, nil
}

type stubPointsPlayerStatsRepository struct {
	pointsByGameweek map[int]map[string]int
}

func (s *stubPointsPlayerStatsRepository) GetSeasonStatsByLeagueAndPlayer(_ context.Context, _, _ string) (playerstats.SeasonStats, error) {
	return playerstats.SeasonStats{}, nil
}

func (s *stubPointsPlayerStatsRepository) ListMatchHistoryByLeagueAndPlayer(_ context.Context, _, _ string, _ int) ([]playerstats.MatchHistory, error) {
	return nil, nil
}

func (s *stubPointsPlayerStatsRepository) ListFixtureStatsByLeagueAndFixture(_ context.Context, _, _ string) ([]playerstats.FixtureStat, error) {
	return nil, nil
}

func (s *stubPointsPlayerStatsRepository) ListFixtureEventsByLeagueAndFixture(_ context.Context, _, _ string) ([]playerstats.FixtureEvent, error) {
	return nil, nil
}

func (s *stubPointsPlayerStatsRepository) UpsertFixtureStats(_ context.Context, _ string, _ []playerstats.FixtureStat) error {
	return nil
}

func (s *stubPointsPlayerStatsRepository) ReplaceFixtureEvents(_ context.Context, _ string, _ []playerstats.FixtureEvent) error {
	return nil
}

func (s *stubPointsPlayerStatsRepository) GetFantasyPointsByLeagueAndGameweek(_ context.Context, _ string, gameweek int) (map[string]int, error) {
	if s.pointsByGameweek == nil {
		return map[string]int{}, nil
	}
	out := make(map[string]int)
	for key, value := range s.pointsByGameweek[gameweek] {
		out[key] = value
	}
	return out, nil
}

var _ scoring.Repository = (*stubPointsScoringRepository)(nil)
var _ playerstats.Repository = (*stubPointsPlayerStatsRepository)(nil)
