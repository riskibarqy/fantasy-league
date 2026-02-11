package league

import "context"

// Repository describes league persistence needs from use cases.
type Repository interface {
	List(ctx context.Context) ([]League, error)
	GetByID(ctx context.Context, leagueID string) (League, bool, error)
}
