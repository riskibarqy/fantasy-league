package fixture

import "context"

// Repository exposes fixture read operations.
type Repository interface {
	ListByLeague(ctx context.Context, leagueID string) ([]Fixture, error)
	GetByID(ctx context.Context, leagueID, fixtureID string) (Fixture, bool, error)
}
