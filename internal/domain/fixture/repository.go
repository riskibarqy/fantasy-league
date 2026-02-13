package fixture

import "context"

// Repository exposes fixture read operations.
type Repository interface {
	ListByLeague(ctx context.Context, leagueID string) ([]Fixture, error)
}
