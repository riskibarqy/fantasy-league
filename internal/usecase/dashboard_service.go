package usecase

import (
	"context"
	"fmt"

	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
)

type Dashboard struct {
	Gameweek         int
	Budget           float64
	TeamValue        float64
	TotalPoints      int
	Rank             int
	SelectedLeagueID string
}

type DashboardService struct {
	leagueRepo  league.Repository
	playerRepo  player.Repository
	fixtureRepo fixture.Repository
	lineupRepo  lineup.Repository
}

func NewDashboardService(
	leagueRepo league.Repository,
	playerRepo player.Repository,
	fixtureRepo fixture.Repository,
	lineupRepo lineup.Repository,
) *DashboardService {
	return &DashboardService{
		leagueRepo:  leagueRepo,
		playerRepo:  playerRepo,
		fixtureRepo: fixtureRepo,
		lineupRepo:  lineupRepo,
	}
}

func (s *DashboardService) Get(ctx context.Context, userID string) (Dashboard, error) {
	leagues, err := s.leagueRepo.List(ctx)
	if err != nil {
		return Dashboard{}, fmt.Errorf("list leagues: %w", err)
	}
	if len(leagues) == 0 {
		return Dashboard{}, fmt.Errorf("%w: no leagues available", ErrNotFound)
	}

	selected := leagues[0]
	for _, l := range leagues {
		if l.IsDefault {
			selected = l
			break
		}
	}

	fixtures, err := s.fixtureRepo.ListByLeague(ctx, selected.ID)
	if err != nil {
		return Dashboard{}, fmt.Errorf("list fixtures: %w", err)
	}

	gameweek := 1
	if len(fixtures) > 0 {
		gameweek = fixtures[0].Gameweek
		for _, f := range fixtures {
			if f.Gameweek < gameweek {
				gameweek = f.Gameweek
			}
		}
	}

	teamValue := 98.7
	stored, exists, err := s.lineupRepo.GetByUserAndLeague(ctx, userID, selected.ID)
	if err != nil {
		return Dashboard{}, fmt.Errorf("get lineup for dashboard: %w", err)
	}
	if exists {
		allIDs := []string{stored.GoalkeeperID}
		allIDs = append(allIDs, stored.DefenderIDs...)
		allIDs = append(allIDs, stored.MidfielderIDs...)
		allIDs = append(allIDs, stored.ForwardIDs...)
		allIDs = append(allIDs, stored.SubstituteIDs...)

		players, pErr := s.playerRepo.GetByIDs(ctx, selected.ID, allIDs)
		if pErr == nil && len(players) > 0 {
			var totalPrice int64
			for _, p := range players {
				totalPrice += p.Price
			}
			teamValue = float64(totalPrice) / 10.0
		}
	}

	return Dashboard{
		Gameweek:         gameweek,
		Budget:           100.0,
		TeamValue:        teamValue,
		TotalPoints:      0,
		Rank:             0,
		SelectedLeagueID: selected.ID,
	}, nil
}
