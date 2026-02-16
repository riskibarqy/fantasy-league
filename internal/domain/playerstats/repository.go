package playerstats

import "context"

type Repository interface {
	GetSeasonStatsByLeagueAndPlayer(ctx context.Context, leagueID, playerID string) (SeasonStats, error)
	ListMatchHistoryByLeagueAndPlayer(ctx context.Context, leagueID, playerID string, limit int) ([]MatchHistory, error)
	ListFixtureEventsByLeagueAndFixture(ctx context.Context, leagueID, fixtureID string) ([]FixtureEvent, error)
}
