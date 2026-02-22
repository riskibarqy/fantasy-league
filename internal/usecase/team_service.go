package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
	"github.com/riskibarqy/fantasy-league/internal/domain/teamstats"
)

type TeamDetails struct {
	Team       team.Team
	Statistics teamstats.SeasonStats
}

type TeamService struct {
	leagueRepo    league.Repository
	teamRepo      team.Repository
	teamStatsRepo teamstats.Repository
}

func NewTeamService(
	leagueRepo league.Repository,
	teamRepo team.Repository,
	teamStatsRepo teamstats.Repository,
) *TeamService {
	return &TeamService{
		leagueRepo:    leagueRepo,
		teamRepo:      teamRepo,
		teamStatsRepo: teamStatsRepo,
	}
}

func (s *TeamService) GetTeamDetailsByLeague(ctx context.Context, leagueID, teamID string) (TeamDetails, error) {
	teamItem, err := s.getTeam(ctx, leagueID, teamID)
	if err != nil {
		return TeamDetails{}, err
	}

	stats, err := s.teamStatsRepo.GetSeasonStatsByLeagueAndTeam(ctx, leagueID, teamID)
	if err != nil {
		return TeamDetails{}, fmt.Errorf("get team season stats: %w", err)
	}

	return TeamDetails{
		Team:       teamItem,
		Statistics: stats,
	}, nil
}

func (s *TeamService) GetTeamStatsByLeague(ctx context.Context, leagueID, teamID string) (teamstats.SeasonStats, error) {
	if _, err := s.getTeam(ctx, leagueID, teamID); err != nil {
		return teamstats.SeasonStats{}, err
	}

	stats, err := s.teamStatsRepo.GetSeasonStatsByLeagueAndTeam(ctx, leagueID, teamID)
	if err != nil {
		return teamstats.SeasonStats{}, fmt.Errorf("get team season stats: %w", err)
	}

	return stats, nil
}

func (s *TeamService) GetTeamHistoryByLeague(ctx context.Context, leagueID, teamID string, limit int) ([]teamstats.MatchHistory, error) {
	if _, err := s.getTeam(ctx, leagueID, teamID); err != nil {
		return nil, err
	}

	items, err := s.teamStatsRepo.ListMatchHistoryByLeagueAndTeam(ctx, leagueID, teamID, limit)
	if err != nil {
		return nil, fmt.Errorf("list team match history: %w", err)
	}

	return items, nil
}

func (s *TeamService) ListFixtureStatsByLeague(ctx context.Context, leagueID, fixtureID string) ([]teamstats.FixtureStat, error) {
	leagueID = strings.TrimSpace(leagueID)
	fixtureID = strings.TrimSpace(fixtureID)
	if leagueID == "" {
		return nil, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}
	if fixtureID == "" {
		return nil, fmt.Errorf("%w: fixture id is required", ErrInvalidInput)
	}

	_, exists, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("get league: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("%w: league=%s", ErrNotFound, leagueID)
	}

	items, err := s.teamStatsRepo.ListFixtureStatsByLeagueAndFixture(ctx, leagueID, fixtureID)
	if err != nil {
		return nil, fmt.Errorf("list team fixture stats: %w", err)
	}

	return items, nil
}

func (s *TeamService) getTeam(ctx context.Context, leagueID, teamID string) (team.Team, error) {
	leagueID = strings.TrimSpace(leagueID)
	teamID = strings.TrimSpace(teamID)
	if leagueID == "" {
		return team.Team{}, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}
	if teamID == "" {
		return team.Team{}, fmt.Errorf("%w: team id is required", ErrInvalidInput)
	}

	_, exists, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil {
		return team.Team{}, fmt.Errorf("get league: %w", err)
	}
	if !exists {
		return team.Team{}, fmt.Errorf("%w: league=%s", ErrNotFound, leagueID)
	}

	item, exists, err := s.teamRepo.GetByID(ctx, leagueID, teamID)
	if err != nil {
		return team.Team{}, fmt.Errorf("get team by id: %w", err)
	}
	if !exists {
		return team.Team{}, fmt.Errorf("%w: team=%s league=%s", ErrNotFound, teamID, leagueID)
	}

	return item, nil
}
