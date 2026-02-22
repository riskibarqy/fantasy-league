package memory

import (
	"context"
	"sync"

	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
)

type FixtureRepository struct {
	mu               sync.RWMutex
	fixturesByLeague map[string][]fixture.Fixture
}

func NewFixtureRepository(fixtures []fixture.Fixture) *FixtureRepository {
	fixturesByLeague := make(map[string][]fixture.Fixture)
	for _, item := range fixtures {
		fixturesByLeague[item.LeagueID] = append(fixturesByLeague[item.LeagueID], item)
	}

	return &FixtureRepository{fixturesByLeague: fixturesByLeague}
}

func (r *FixtureRepository) ListByLeague(_ context.Context, leagueID string) ([]fixture.Fixture, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.fixturesByLeague[leagueID]
	out := make([]fixture.Fixture, 0, len(items))
	out = append(out, items...)
	return out, nil
}

func (r *FixtureRepository) GetByID(_ context.Context, leagueID, fixtureID string) (fixture.Fixture, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, item := range r.fixturesByLeague[leagueID] {
		if item.ID == fixtureID {
			return item, true, nil
		}
	}

	return fixture.Fixture{}, false, nil
}
