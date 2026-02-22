package rawdata

import "context"

type Repository interface {
	UpsertMany(ctx context.Context, items []Payload) error
}
