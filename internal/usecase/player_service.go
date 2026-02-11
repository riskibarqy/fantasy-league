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
