package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
)

type PlayerService struct {
	leagueRepo league.Repository
	playerRepo player.Repository
}

func NewPlayerService(leagueRepo league.Repository, playerRepo player.Repository) *PlayerService {
	return &PlayerService{
		leagueRepo: leagueRepo,
		playerRepo: playerRepo,
	}
}

func (s *PlayerService) ListPlayersByLeague(ctx context.Context, leagueID string) ([]player.Player, error) {
	ctx, span := startUsecaseSpan(ctx, "usecase.PlayerService.ListPlayersByLeague")
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

	players, err := s.playerRepo.ListByLeague(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("list players by league: %w", err)
	}

	return players, nil
}

func (s *PlayerService) GetPlayerByLeagueAndID(ctx context.Context, leagueID, playerID string) (player.Player, error) {
	ctx, span := startUsecaseSpan(ctx, "usecase.PlayerService.GetPlayerByLeagueAndID")
	defer span.End()

	leagueID = strings.TrimSpace(leagueID)
	playerID = strings.TrimSpace(playerID)
	if leagueID == "" {
		return player.Player{}, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}
	if playerID == "" {
		return player.Player{}, fmt.Errorf("%w: player id is required", ErrInvalidInput)
	}

	_, exists, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil {
		return player.Player{}, fmt.Errorf("get league: %w", err)
	}
	if !exists {
		return player.Player{}, fmt.Errorf("%w: league=%s", ErrNotFound, leagueID)
	}

	items, err := s.playerRepo.GetByIDs(ctx, leagueID, []string{playerID})
	if err != nil {
		return player.Player{}, fmt.Errorf("get player by id: %w", err)
	}
	if len(items) == 0 {
		return player.Player{}, fmt.Errorf("%w: player=%s league=%s", ErrNotFound, playerID, leagueID)
	}

	return items[0], nil
}
