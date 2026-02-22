package teamstats

import "context"

type Repository interface {
	GetSeasonStatsByLeagueAndTeam(ctx context.Context, leagueID, teamID string) (SeasonStats, error)
	ListMatchHistoryByLeagueAndTeam(ctx context.Context, leagueID, teamID string, limit int) ([]MatchHistory, error)
	ListFixtureStatsByLeagueAndFixture(ctx context.Context, leagueID, fixtureID string) ([]FixtureStat, error)
	UpsertFixtureStats(ctx context.Context, fixtureID string, stats []FixtureStat) error
}
