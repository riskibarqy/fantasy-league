package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
)

type LeagueService struct {
	leagueRepo league.Repository
	teamRepo   team.Repository
}

func NewLeagueService(leagueRepo league.Repository, teamRepo team.Repository) *LeagueService {
	return &LeagueService{
		leagueRepo: leagueRepo,
		teamRepo:   teamRepo,
	}
}

func (s *LeagueService) ListLeagues(ctx context.Context) ([]league.League, error) {
	leagues, err := s.leagueRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list leagues: %w", err)
	}

	return leagues, nil
}

func (s *LeagueService) ListTeamsByLeague(ctx context.Context, leagueID string) ([]team.Team, error) {
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

	teams, err := s.teamRepo.ListByLeague(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("list teams by league: %w", err)
	}

	return teams, nil
}
