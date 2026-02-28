package statvalue

import "context"

type Repository interface {
	UpsertTypes(ctx context.Context, items []Type) error
	UpsertTeamValues(ctx context.Context, items []TeamValue) error
	UpsertPlayerValues(ctx context.Context, items []PlayerValue) error
}
