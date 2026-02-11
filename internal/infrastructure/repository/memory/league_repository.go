package memory

import (
	"context"
	"sync"

	"github.com/riskibarqy/fantasy-league/internal/domain/league"
)

type LeagueRepository struct {
	mu     sync.RWMutex
	items  map[string]league.League
	orders []string
}

func NewLeagueRepository(leagues []league.League) *LeagueRepository {
	items := make(map[string]league.League, len(leagues))
	orders := make([]string, 0, len(leagues))

	for _, l := range leagues {
		items[l.ID] = l
		orders = append(orders, l.ID)
	}

	return &LeagueRepository{
		items:  items,
		orders: orders,
	}
}

func (r *LeagueRepository) List(_ context.Context) ([]league.League, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]league.League, 0, len(r.orders))
	for _, id := range r.orders {
		out = append(out, r.items[id])
	}

	return out, nil
}

func (r *LeagueRepository) GetByID(_ context.Context, leagueID string) (league.League, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	l, ok := r.items[leagueID]
	if !ok {
		return league.League{}, false, nil
	}

	return l, true, nil
}
