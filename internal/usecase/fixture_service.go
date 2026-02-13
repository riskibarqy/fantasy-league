package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
)

type FixtureService struct {
	leagueRepo  league.Repository
	fixtureRepo fixture.Repository
}

func NewFixtureService(leagueRepo league.Repository, fixtureRepo fixture.Repository) *FixtureService {
	return &FixtureService{
		leagueRepo:  leagueRepo,
		fixtureRepo: fixtureRepo,
	}
}

func (s *FixtureService) ListByLeague(ctx context.Context, leagueID string) ([]fixture.Fixture, error) {
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

	fixtures, err := s.fixtureRepo.ListByLeague(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("list fixtures by league: %w", err)
	}

	return fixtures, nil
}
