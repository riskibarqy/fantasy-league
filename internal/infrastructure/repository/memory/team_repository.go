package memory

import (
	"context"
	"strings"
	"sync"

	"github.com/riskibarqy/fantasy-league/internal/domain/team"
)

type TeamRepository struct {
	mu            sync.RWMutex
	teamsByLeague map[string][]team.Team
}

func NewTeamRepository(teams []team.Team) *TeamRepository {
	teamsByLeague := make(map[string][]team.Team)
	for _, item := range teams {
		teamsByLeague[item.LeagueID] = append(teamsByLeague[item.LeagueID], item)
	}

	return &TeamRepository{teamsByLeague: teamsByLeague}
}

func (r *TeamRepository) ListByLeague(_ context.Context, leagueID string) ([]team.Team, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	teams := r.teamsByLeague[leagueID]
	out := make([]team.Team, 0, len(teams))
	out = append(out, teams...)

	return out, nil
}

func (r *TeamRepository) GetByID(_ context.Context, leagueID, teamID string) (team.Team, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	teams := r.teamsByLeague[leagueID]
	for _, item := range teams {
		if item.ID == teamID {
			return item, true, nil
		}
	}

	return team.Team{}, false, nil
}

func (r *TeamRepository) UpsertTeams(_ context.Context, items []team.Team) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, item := range items {
		leagueID := strings.TrimSpace(item.LeagueID)
		teamID := strings.TrimSpace(item.ID)
		if leagueID == "" || teamID == "" {
			continue
		}

		rows := r.teamsByLeague[leagueID]
		updated := false
		for idx := range rows {
			if rows[idx].ID == teamID {
				rows[idx] = item
				updated = true
				break
			}
		}
		if !updated {
			rows = append(rows, item)
		}
		r.teamsByLeague[leagueID] = rows
	}

	return nil
}
