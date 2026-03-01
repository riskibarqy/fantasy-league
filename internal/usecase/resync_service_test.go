package usecase

import (
	"context"
	"testing"

	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	"github.com/riskibarqy/fantasy-league/internal/domain/rawdata"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
)

func TestSportDataSyncService_Resync_SkippedKinds(t *testing.T) {
	t.Parallel()

	svc := &SportDataSyncService{
		provider:   stubResyncProvider{},
		teamRepo:   stubResyncTeamRepo{},
		playerRepo: stubResyncPlayerRepo{},
		ingestion:  &IngestionService{},
		cfg: SportDataSyncConfig{
			Enabled: true,
			SeasonIDByLeague: map[string]int64{
				"idn-liga-1-2025": 25965,
			},
		},
	}

	result, err := svc.Resync(context.Background(), ResyncInput{
		LeagueID: "idn-liga-1-2025",
		SyncData: []string{"players", "team"},
	})
	if err != nil {
		t.Fatalf("Resync error: %v", err)
	}

	if result.TaskCount != 2 {
		t.Fatalf("expected 2 tasks, got=%d", result.TaskCount)
	}
	if result.SkippedCount != 2 {
		t.Fatalf("expected 2 skipped tasks, got=%d", result.SkippedCount)
	}
	if result.FailedCount != 0 {
		t.Fatalf("expected 0 failed tasks, got=%d", result.FailedCount)
	}
	if result.SuccessCount != 0 {
		t.Fatalf("expected 0 success tasks, got=%d", result.SuccessCount)
	}
}

func TestSportDataSyncService_ResolveResyncTargetsBySeason(t *testing.T) {
	t.Parallel()

	svc := &SportDataSyncService{
		cfg: SportDataSyncConfig{
			SeasonIDByLeague: map[string]int64{
				"idn-liga-1-2025": 25965,
				"global-liga-1":   25965,
				"other":           11111,
			},
		},
	}

	targets, err := svc.resolveResyncTargets("", 25965)
	if err != nil {
		t.Fatalf("resolveResyncTargets error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets for season 25965, got=%d", len(targets))
	}
	if targets[0].leagueID != "global-liga-1" || targets[1].leagueID != "idn-liga-1-2025" {
		t.Fatalf("unexpected target ordering: %+v", targets)
	}
}

type stubResyncProvider struct{}

func (stubResyncProvider) FetchFixtureBundleBySeason(_ context.Context, _ int64) (ExternalFixtureBundle, error) {
	return ExternalFixtureBundle{}, nil
}

func (stubResyncProvider) FetchFixturesBySeason(_ context.Context, _ int64) ([]ExternalFixture, []rawdata.Payload, error) {
	return nil, nil, nil
}

func (stubResyncProvider) FetchStandingsBySeason(_ context.Context, _ int64) ([]ExternalStanding, []rawdata.Payload, error) {
	return nil, nil, nil
}

func (stubResyncProvider) FetchLiveStandingsByLeague(_ context.Context, _ int64) ([]ExternalStanding, []rawdata.Payload, error) {
	return nil, nil, nil
}
func (stubResyncProvider) FetchTopScorersBySeasonID(_ context.Context, _ int, _ int, _ int) ([]ExternalTopScorers, bool, error) {
	return nil, false, nil
}

func (stubResyncProvider) FetchStatisticTypes(_ context.Context) ([]ExternalStatType, []rawdata.Payload, error) {
	return nil, nil, nil
}

func (stubResyncProvider) FetchTeamStatisticsBySeason(_ context.Context, _ int64) ([]ExternalTeamStatValue, []rawdata.Payload, error) {
	return nil, nil, nil
}

func (stubResyncProvider) FetchPlayerStatisticsBySeason(_ context.Context, _ int64) ([]ExternalPlayerStatValue, []rawdata.Payload, error) {
	return nil, nil, nil
}

type stubResyncTeamRepo struct{}

func (stubResyncTeamRepo) ListByLeague(_ context.Context, _ string) ([]team.Team, error) {
	return []team.Team{}, nil
}

func (stubResyncTeamRepo) GetByID(_ context.Context, _, _ string) (team.Team, bool, error) {
	return team.Team{}, false, nil
}

type stubResyncPlayerRepo struct{}

func (stubResyncPlayerRepo) ListByLeague(_ context.Context, _ string) ([]player.Player, error) {
	return []player.Player{}, nil
}

func (stubResyncPlayerRepo) GetByIDs(_ context.Context, _ string, _ []string) ([]player.Player, error) {
	return []player.Player{}, nil
}
