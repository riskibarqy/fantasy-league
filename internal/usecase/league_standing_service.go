package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/leaguestanding"
)

type LeagueStandingService struct {
	leagueRepo   league.Repository
	standingRepo leaguestanding.Repository
}

func NewLeagueStandingService(leagueRepo league.Repository, standingRepo leaguestanding.Repository) *LeagueStandingService {
	return &LeagueStandingService{
		leagueRepo:   leagueRepo,
		standingRepo: standingRepo,
	}
}

func (s *LeagueStandingService) ListByLeague(ctx context.Context, leagueID string, live bool) ([]leaguestanding.Standing, error) {
	leagueID = strings.TrimSpace(leagueID)
	if leagueID == "" {
		return nil, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}

	_, exists, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("get league: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("%w: league=%s", ErrNotFound, leagueID)
	}

	items, err := s.standingRepo.ListByLeague(ctx, leagueID, live)
	if err != nil {
		return nil, fmt.Errorf("list league standings: %w", err)
	}

	return items, nil
}
