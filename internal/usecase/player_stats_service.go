package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/domain/playerstats"
)

type PlayerStatsService struct {
	statsRepo playerstats.Repository
}

func NewPlayerStatsService(statsRepo playerstats.Repository) *PlayerStatsService {
	return &PlayerStatsService{statsRepo: statsRepo}
}

func (s *PlayerStatsService) GetSeasonStats(ctx context.Context, leagueID, playerID string) (playerstats.SeasonStats, error) {
	leagueID = strings.TrimSpace(leagueID)
	playerID = strings.TrimSpace(playerID)
	if leagueID == "" {
		return playerstats.SeasonStats{}, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}
	if playerID == "" {
		return playerstats.SeasonStats{}, fmt.Errorf("%w: player id is required", ErrInvalidInput)
	}

	stats, err := s.statsRepo.GetSeasonStatsByLeagueAndPlayer(ctx, leagueID, playerID)
	if err != nil {
		return playerstats.SeasonStats{}, fmt.Errorf("get season stats: %w", err)
	}

	return stats, nil
}

func (s *PlayerStatsService) ListMatchHistory(ctx context.Context, leagueID, playerID string, limit int) ([]playerstats.MatchHistory, error) {
	leagueID = strings.TrimSpace(leagueID)
	playerID = strings.TrimSpace(playerID)
	if leagueID == "" {
		return nil, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}
	if playerID == "" {
		return nil, fmt.Errorf("%w: player id is required", ErrInvalidInput)
	}

	items, err := s.statsRepo.ListMatchHistoryByLeagueAndPlayer(ctx, leagueID, playerID, limit)
	if err != nil {
		return nil, fmt.Errorf("list match history: %w", err)
	}

	return items, nil
}

func (s *PlayerStatsService) ListFixtureEvents(ctx context.Context, leagueID, fixtureID string) ([]playerstats.FixtureEvent, error) {
	leagueID = strings.TrimSpace(leagueID)
	fixtureID = strings.TrimSpace(fixtureID)
	if leagueID == "" {
		return nil, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}
	if fixtureID == "" {
		return nil, fmt.Errorf("%w: fixture id is required", ErrInvalidInput)
	}

	items, err := s.statsRepo.ListFixtureEventsByLeagueAndFixture(ctx, leagueID, fixtureID)
	if err != nil {
		return nil, fmt.Errorf("list fixture events: %w", err)
	}

	return items, nil
}

func (s *PlayerStatsService) ListFixtureStats(ctx context.Context, leagueID, fixtureID string) ([]playerstats.FixtureStat, error) {
	leagueID = strings.TrimSpace(leagueID)
	fixtureID = strings.TrimSpace(fixtureID)
	if leagueID == "" {
		return nil, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}
	if fixtureID == "" {
		return nil, fmt.Errorf("%w: fixture id is required", ErrInvalidInput)
	}

	items, err := s.statsRepo.ListFixtureStatsByLeagueAndFixture(ctx, leagueID, fixtureID)
	if err != nil {
		return nil, fmt.Errorf("list fixture player stats: %w", err)
	}

	return items, nil
}
