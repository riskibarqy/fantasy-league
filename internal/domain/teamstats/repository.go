package teamstats

import "context"

type Repository interface {
	GetSeasonStatsByLeagueAndTeam(ctx context.Context, leagueID, teamID string) (SeasonStats, error)
	ListMatchHistoryByLeagueAndTeam(ctx context.Context, leagueID, teamID string, limit int) ([]MatchHistory, error)
}
