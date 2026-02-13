package anubis

import (
	"testing"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/user"
)

func TestInMemoryPrincipalCache_SetGet(t *testing.T) {
	t.Parallel()

	cache := newInMemoryPrincipalCache(200*time.Millisecond, 10)
	cache.Set("k1", user.Principal{UserID: "u-1"})

	principal, ok := cache.Get("k1")
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if principal.UserID != "u-1" {
		t.Fatalf("unexpected user id: %s", principal.UserID)
	}
}

func TestInMemoryPrincipalCache_Expired(t *testing.T) {
	t.Parallel()

	cache := newInMemoryPrincipalCache(20*time.Millisecond, 10)
	cache.Set("k1", user.Principal{UserID: "u-1"})
	time.Sleep(40 * time.Millisecond)

	if _, ok := cache.Get("k1"); ok {
		t.Fatalf("expected cache miss after expiry")
	}
}
