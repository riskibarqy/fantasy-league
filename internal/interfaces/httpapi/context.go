package httpapi

import (
	"context"

	"github.com/riskibarqy/fantasy-league/internal/domain/user"
)

type contextKey string

const principalContextKey contextKey = "auth_principal"

func withPrincipal(ctx context.Context, p user.Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, p)
}

func principalFromContext(ctx context.Context) (user.Principal, bool) {
	p, ok := ctx.Value(principalContextKey).(user.Principal)
	return p, ok
}
