package jobscheduler

import "context"

type Repository interface {
	UpsertEvent(ctx context.Context, event DispatchEvent) error
}
