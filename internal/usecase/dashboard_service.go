package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/customleague"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
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
	fixtureRepo fixture.Repository
	squadRepo   fantasy.Repository
	groupRepo   customleague.Repository
	scoringSvc  dashboardScoringProvider
}

type dashboardScoringProvider interface {
	GetUserLeagueSummary(ctx context.Context, leagueID, userID string) (int, int, error)
}

func NewDashboardService(
	leagueRepo league.Repository,
	fixtureRepo fixture.Repository,
	squadRepo fantasy.Repository,
	groupRepo customleague.Repository,
	scoringSvc dashboardScoringProvider,
) *DashboardService {
	return &DashboardService{
		leagueRepo:  leagueRepo,
		fixtureRepo: fixtureRepo,
		squadRepo:   squadRepo,
		groupRepo:   groupRepo,
		scoringSvc:  scoringSvc,
	}
}

func (s *DashboardService) Get(ctx context.Context, userID string) (Dashboard, error) {
	ctx, span := startUsecaseSpan(ctx, "usecase.DashboardService.Get")
	defer span.End()

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

	gameweek := resolveDashboardGameweek(fixtures, time.Now().UTC())

	budgetCap := fantasy.DefaultRules().BudgetCap
	teamValue := 0.0
	budget := float64(budgetCap) / 10.0
	squad, exists, err := s.squadRepo.GetByUserAndLeague(ctx, userID, selected.ID)
	if err != nil {
		return Dashboard{}, fmt.Errorf("get squad for dashboard: %w", err)
	}
	if exists {
		usedBudget := squadCost(squad)
		budgetCap = squad.BudgetCap
		teamValue = float64(usedBudget) / 10.0
		if budgetCap > usedBudget {
			budget = float64(budgetCap-usedBudget) / 10.0
		} else {
			budget = 0
		}
	}

	totalPoints := 0
	rank := 0
	if s.scoringSvc != nil {
		points, userRank, scoreErr := s.scoringSvc.GetUserLeagueSummary(ctx, selected.ID, userID)
		if scoreErr != nil {
			return Dashboard{}, fmt.Errorf("get scoring summary for dashboard: %w", scoreErr)
		}
		totalPoints = points
		rank = userRank
	} else {
		defaultGroups, groupErr := s.groupRepo.ListDefaultGroupsByLeague(ctx, selected.ID)
		if groupErr != nil {
			return Dashboard{}, fmt.Errorf("list default groups for dashboard: %w", groupErr)
		}
		if len(defaultGroups) > 0 {
			standings, standingsErr := s.groupRepo.ListStandingsByGroup(ctx, defaultGroups[0].ID)
			if standingsErr != nil {
				return Dashboard{}, fmt.Errorf("list standings for dashboard: %w", standingsErr)
			}
			for _, standing := range standings {
				if standing.UserID == userID {
					totalPoints = standing.Points
					rank = standing.Rank
					break
				}
			}
		}
	}

	return Dashboard{
		Gameweek:         gameweek,
		Budget:           budget,
		TeamValue:        teamValue,
		TotalPoints:      totalPoints,
		Rank:             rank,
		SelectedLeagueID: selected.ID,
	}, nil
}

func squadCost(squad fantasy.Squad) int64 {
	total := int64(0)
	for _, pick := range squad.Picks {
		total += pick.Price
	}
	return total
}

func resolveDashboardGameweek(fixtures []fixture.Fixture, now time.Time) int {
	if len(fixtures) == 0 {
		return 1
	}

	liveMin := 0
	upcomingMin := 0
	lastKnown := 0

	for _, item := range fixtures {
		if item.Gameweek <= 0 {
			continue
		}
		if item.Gameweek > lastKnown {
			lastKnown = item.Gameweek
		}

		status := fixture.NormalizeStatus(item.Status)
		if fixture.IsLiveStatus(status) {
			if liveMin == 0 || item.Gameweek < liveMin {
				liveMin = item.Gameweek
			}
			continue
		}
		if fixture.IsFinishedStatus(status) || fixture.IsCancelledLikeStatus(status) {
			continue
		}

		if item.KickoffAt.IsZero() || !item.KickoffAt.Before(now) {
			if upcomingMin == 0 || item.Gameweek < upcomingMin {
				upcomingMin = item.Gameweek
			}
			continue
		}

		// Match is in the past and still not finished/cancelled; treat as active gameweek.
		if upcomingMin == 0 || item.Gameweek < upcomingMin {
			upcomingMin = item.Gameweek
		}
	}

	if liveMin > 0 {
		return liveMin
	}
	if upcomingMin > 0 {
		return upcomingMin
	}
	if lastKnown > 0 {
		return lastKnown
	}

	return 1
}
