package fantasy

import "context"

// Repository describes squad persistence needs from use cases.
type Repository interface {
	GetByUserAndLeague(ctx context.Context, userID, leagueID string) (Squad, bool, error)
	Upsert(ctx context.Context, squad Squad) error
}
