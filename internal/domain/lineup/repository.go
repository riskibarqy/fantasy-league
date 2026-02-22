package lineup

import "context"

// Repository exposes lineup persistence operations.
type Repository interface {
	GetByUserAndLeague(ctx context.Context, userID, leagueID string) (Lineup, bool, error)
	ListByLeague(ctx context.Context, leagueID string) ([]Lineup, error)
	Upsert(ctx context.Context, lineup Lineup) error
}
