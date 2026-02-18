package onboarding

import "context"

type Repository interface {
	GetByUserID(ctx context.Context, userID string) (Profile, bool, error)
	Upsert(ctx context.Context, profile Profile) error
}
