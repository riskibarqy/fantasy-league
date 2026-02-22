package playerstats

import "context"

type Repository interface {
	GetSeasonStatsByLeagueAndPlayer(ctx context.Context, leagueID, playerID string) (SeasonStats, error)
	ListMatchHistoryByLeagueAndPlayer(ctx context.Context, leagueID, playerID string, limit int) ([]MatchHistory, error)
	ListFixtureEventsByLeagueAndFixture(ctx context.Context, leagueID, fixtureID string) ([]FixtureEvent, error)
	UpsertFixtureStats(ctx context.Context, fixtureID string, stats []FixtureStat) error
	ReplaceFixtureEvents(ctx context.Context, fixtureID string, events []FixtureEvent) error
	GetFantasyPointsByLeagueAndGameweek(ctx context.Context, leagueID string, gameweek int) (map[string]int, error)
}
