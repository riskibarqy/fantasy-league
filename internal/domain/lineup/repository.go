package lineup

import "context"

// Repository exposes lineup persistence operations.
type Repository interface {
	GetByUserAndLeague(ctx context.Context, userID, leagueID string) (Lineup, bool, error)
	Upsert(ctx context.Context, lineup Lineup) error
}
