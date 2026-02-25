package memory

import (
	"context"
	"strings"
	"sync"

	"github.com/riskibarqy/fantasy-league/internal/domain/player"
)

type PlayerRepository struct {
	mu              sync.RWMutex
	playersByLeague map[string][]player.Player
	indexByLeague   map[string]map[string]player.Player
}

func NewPlayerRepository(players []player.Player) *PlayerRepository {
	playersByLeague := make(map[string][]player.Player)
	indexByLeague := make(map[string]map[string]player.Player)

	for _, p := range players {
		playersByLeague[p.LeagueID] = append(playersByLeague[p.LeagueID], p)
		if _, ok := indexByLeague[p.LeagueID]; !ok {
			indexByLeague[p.LeagueID] = make(map[string]player.Player)
		}
		indexByLeague[p.LeagueID][p.ID] = p
	}

	return &PlayerRepository{
		playersByLeague: playersByLeague,
		indexByLeague:   indexByLeague,
	}
}

func (r *PlayerRepository) ListByLeague(_ context.Context, leagueID string) ([]player.Player, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players := r.playersByLeague[leagueID]
	out := make([]player.Player, 0, len(players))
	out = append(out, players...)

	return out, nil
}

func (r *PlayerRepository) GetByIDs(_ context.Context, leagueID string, playerIDs []string) ([]player.Player, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	index := r.indexByLeague[leagueID]
	out := make([]player.Player, 0, len(playerIDs))
	for _, id := range playerIDs {
		p, ok := index[id]
		if !ok {
			continue
		}
		out = append(out, p)
	}

	return out, nil
}

func (r *PlayerRepository) UpsertPlayers(_ context.Context, items []player.Player) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, item := range items {
		leagueID := strings.TrimSpace(item.LeagueID)
		playerID := strings.TrimSpace(item.ID)
		if leagueID == "" || playerID == "" {
			continue
		}

		rows := r.playersByLeague[leagueID]
		index := r.indexByLeague[leagueID]
		if index == nil {
			index = make(map[string]player.Player)
		}

		updated := false
		for idx := range rows {
			if rows[idx].ID == playerID {
				rows[idx] = item
				updated = true
				break
			}
		}
		if !updated {
			rows = append(rows, item)
		}
		index[playerID] = item

		r.playersByLeague[leagueID] = rows
		r.indexByLeague[leagueID] = index
	}

	return nil
}
