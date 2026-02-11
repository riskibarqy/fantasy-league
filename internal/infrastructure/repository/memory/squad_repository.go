package memory

import (
	"context"
	"sync"

	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
)

type SquadRepository struct {
	mu    sync.RWMutex
	items map[string]fantasy.Squad
}

func NewSquadRepository() *SquadRepository {
	return &SquadRepository{items: make(map[string]fantasy.Squad)}
}

func (r *SquadRepository) GetByUserAndLeague(_ context.Context, userID, leagueID string) (fantasy.Squad, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	squad, ok := r.items[squadKey(userID, leagueID)]
	if !ok {
		return fantasy.Squad{}, false, nil
	}

	return cloneSquad(squad), true, nil
}

func (r *SquadRepository) Upsert(_ context.Context, squad fantasy.Squad) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items[squadKey(squad.UserID, squad.LeagueID)] = cloneSquad(squad)
	return nil
}

func squadKey(userID, leagueID string) string {
	return userID + "::" + leagueID
}

func cloneSquad(s fantasy.Squad) fantasy.Squad {
	copied := s
	copied.Picks = append([]fantasy.SquadPick(nil), s.Picks...)
	return copied
}
