package team

import "context"

// Repository describes team persistence needs from use cases.
type Repository interface {
	ListByLeague(ctx context.Context, leagueID string) ([]Team, error)
	GetByID(ctx context.Context, leagueID, teamID string) (Team, bool, error)
}
