package memory

import (
	"context"
	"sync"

	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
)

type LineupRepository struct {
	mu    sync.RWMutex
	items map[string]lineup.Lineup
}

func NewLineupRepository() *LineupRepository {
	return &LineupRepository{items: make(map[string]lineup.Lineup)}
}

func (r *LineupRepository) GetByUserAndLeague(_ context.Context, userID, leagueID string) (lineup.Lineup, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.items[lineupKey(userID, leagueID)]
	if !ok {
		return lineup.Lineup{}, false, nil
	}

	return cloneLineup(item), true, nil
}

func (r *LineupRepository) Upsert(_ context.Context, item lineup.Lineup) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items[lineupKey(item.UserID, item.LeagueID)] = cloneLineup(item)
	return nil
}

func lineupKey(userID, leagueID string) string {
	return userID + "::" + leagueID
}

func cloneLineup(item lineup.Lineup) lineup.Lineup {
	copied := item
	copied.DefenderIDs = append([]string(nil), item.DefenderIDs...)
	copied.MidfielderIDs = append([]string(nil), item.MidfielderIDs...)
	copied.ForwardIDs = append([]string(nil), item.ForwardIDs...)
	copied.SubstituteIDs = append([]string(nil), item.SubstituteIDs...)
	return copied
}
