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
	ctx, span := startUsecaseSpan(ctx, "usecase.FixtureService.ListByLeague")
	defer span.End()

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

func (s *FixtureService) GetByLeagueAndID(ctx context.Context, leagueID, fixtureID string) (fixture.Fixture, error) {
	ctx, span := startUsecaseSpan(ctx, "usecase.FixtureService.GetByLeagueAndID")
	defer span.End()

	leagueID = strings.TrimSpace(leagueID)
	fixtureID = strings.TrimSpace(fixtureID)
	if leagueID == "" {
		return fixture.Fixture{}, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}
	if fixtureID == "" {
		return fixture.Fixture{}, fmt.Errorf("%w: fixture id is required", ErrInvalidInput)
	}

	_, exists, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil {
		return fixture.Fixture{}, fmt.Errorf("get league: %w", err)
	}
	if !exists {
		return fixture.Fixture{}, fmt.Errorf("%w: league=%s", ErrNotFound, leagueID)
	}

	item, exists, err := s.fixtureRepo.GetByID(ctx, leagueID, fixtureID)
	if err != nil {
		return fixture.Fixture{}, fmt.Errorf("get fixture by id: %w", err)
	}
	if !exists {
		return fixture.Fixture{}, fmt.Errorf("%w: fixture=%s league=%s", ErrNotFound, fixtureID, leagueID)
	}

	return item, nil
}
