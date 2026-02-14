package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	idgen "github.com/riskibarqy/fantasy-league/internal/platform/id"
)

// UpsertSquadInput is the incoming payload for create/update squad.
type UpsertSquadInput struct {
	UserID    string
	LeagueID  string
	Name      string
	PlayerIDs []string
}

// PickSquadInput is the incoming payload for squad picking API.
// SquadName is optional:
// - when empty and squad exists, existing name is reused
// - when empty and squad does not exist, a default name is used
type PickSquadInput struct {
	UserID    string
	LeagueID  string
	SquadName string
	PlayerIDs []string
}

type AddPlayerToSquadInput struct {
	UserID    string
	LeagueID  string
	SquadName string
	PlayerID  string
}

const defaultSquadName = "My Squad"

type SquadService struct {
	leagueRepo league.Repository
	playerRepo player.Repository
	squadRepo  fantasy.Repository
	rules      fantasy.Rules
	idGen      idgen.Generator
	logger     *slog.Logger
	now        func() time.Time
}

func NewSquadService(
	leagueRepo league.Repository,
	playerRepo player.Repository,
	squadRepo fantasy.Repository,
	rules fantasy.Rules,
	idGen idgen.Generator,
	logger *slog.Logger,
) *SquadService {
	if logger == nil {
		logger = slog.Default()
	}

	return &SquadService{
		leagueRepo: leagueRepo,
		playerRepo: playerRepo,
		squadRepo:  squadRepo,
		rules:      rules,
		idGen:      idGen,
		logger:     logger,
		now:        time.Now,
	}
}

func (s *SquadService) UpsertSquad(ctx context.Context, input UpsertSquadInput) (fantasy.Squad, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.LeagueID = strings.TrimSpace(input.LeagueID)
	input.Name = strings.TrimSpace(input.Name)

	if input.UserID == "" {
		return fantasy.Squad{}, fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if input.LeagueID == "" {
		return fantasy.Squad{}, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}
	if input.Name == "" {
		return fantasy.Squad{}, fmt.Errorf("%w: squad name is required", ErrInvalidInput)
	}
	if len(input.PlayerIDs) == 0 {
		return fantasy.Squad{}, fmt.Errorf("%w: player ids are required", ErrInvalidInput)
	}

	if err := s.validateLeague(ctx, input.LeagueID); err != nil {
		return fantasy.Squad{}, err
	}

	playerIDs, err := cleanPlayerIDs(input.PlayerIDs)
	if err != nil {
		return fantasy.Squad{}, err
	}

	players, err := s.playerRepo.GetByIDs(ctx, input.LeagueID, playerIDs)
	if err != nil {
		return fantasy.Squad{}, fmt.Errorf("get players by ids: %w", err)
	}
	if len(players) != len(playerIDs) {
		return fantasy.Squad{}, fmt.Errorf("%w: some players are missing from league=%s", ErrInvalidInput, input.LeagueID)
	}

	pickByPlayerID := make(map[string]fantasy.SquadPick, len(players))
	for _, p := range players {
		pickByPlayerID[p.ID] = fantasy.SquadPick{
			PlayerID: p.ID,
			TeamID:   p.TeamID,
			Position: p.Position,
			Price:    p.Price,
		}
	}

	picks := make([]fantasy.SquadPick, 0, len(playerIDs))
	for _, playerID := range playerIDs {
		pick, ok := pickByPlayerID[playerID]
		if !ok {
			return fantasy.Squad{}, fmt.Errorf("%w: player id %s not found", ErrInvalidInput, playerID)
		}
		picks = append(picks, pick)
	}

	if err := fantasy.ValidatePicks(picks, s.rules); err != nil {
		return fantasy.Squad{}, fmt.Errorf("validate squad picks: %w", err)
	}

	now := s.now().UTC()
	existingSquad, exists, err := s.squadRepo.GetByUserAndLeague(ctx, input.UserID, input.LeagueID)
	if err != nil {
		return fantasy.Squad{}, fmt.Errorf("get existing squad: %w", err)
	}

	squadID := existingSquad.ID
	createdAt := existingSquad.CreatedAt
	if !exists {
		squadID, err = s.idGen.NewID()
		if err != nil {
			return fantasy.Squad{}, fmt.Errorf("generate squad id: %w", err)
		}
		createdAt = now
	}

	squad := fantasy.Squad{
		ID:        squadID,
		UserID:    input.UserID,
		LeagueID:  input.LeagueID,
		Name:      input.Name,
		Picks:     picks,
		BudgetCap: s.rules.BudgetCap,
		CreatedAt: createdAt,
		UpdatedAt: now,
	}

	if err := squad.ValidateBasic(); err != nil {
		return fantasy.Squad{}, fmt.Errorf("validate squad: %w", err)
	}

	if err := s.squadRepo.Upsert(ctx, squad); err != nil {
		return fantasy.Squad{}, fmt.Errorf("upsert squad: %w", err)
	}

	s.logger.InfoContext(ctx, "squad upserted",
		"user_id", input.UserID,
		"league_id", input.LeagueID,
		"squad_id", squad.ID,
		"player_count", len(squad.Picks),
	)

	return squad, nil
}

func (s *SquadService) PickSquad(ctx context.Context, input PickSquadInput) (fantasy.Squad, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.LeagueID = strings.TrimSpace(input.LeagueID)
	input.SquadName = strings.TrimSpace(input.SquadName)

	if input.UserID == "" {
		return fantasy.Squad{}, fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if input.LeagueID == "" {
		return fantasy.Squad{}, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}
	if len(input.PlayerIDs) == 0 {
		return fantasy.Squad{}, fmt.Errorf("%w: player ids are required", ErrInvalidInput)
	}

	name := input.SquadName
	if name == "" {
		existing, exists, err := s.squadRepo.GetByUserAndLeague(ctx, input.UserID, input.LeagueID)
		if err != nil {
			return fantasy.Squad{}, fmt.Errorf("get existing squad for pick: %w", err)
		}
		if exists && strings.TrimSpace(existing.Name) != "" {
			name = strings.TrimSpace(existing.Name)
		} else {
			name = defaultSquadName
		}
	}

	return s.UpsertSquad(ctx, UpsertSquadInput{
		UserID:    input.UserID,
		LeagueID:  input.LeagueID,
		Name:      name,
		PlayerIDs: input.PlayerIDs,
	})
}

func (s *SquadService) AddPlayerToSquad(ctx context.Context, input AddPlayerToSquadInput) (fantasy.Squad, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.LeagueID = strings.TrimSpace(input.LeagueID)
	input.SquadName = strings.TrimSpace(input.SquadName)
	input.PlayerID = strings.TrimSpace(input.PlayerID)

	if input.UserID == "" {
		return fantasy.Squad{}, fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if input.LeagueID == "" {
		return fantasy.Squad{}, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}
	if input.PlayerID == "" {
		return fantasy.Squad{}, fmt.Errorf("%w: player id is required", ErrInvalidInput)
	}

	if err := s.validateLeague(ctx, input.LeagueID); err != nil {
		return fantasy.Squad{}, err
	}

	players, err := s.playerRepo.GetByIDs(ctx, input.LeagueID, []string{input.PlayerID})
	if err != nil {
		return fantasy.Squad{}, fmt.Errorf("get player by id: %w", err)
	}
	if len(players) != 1 {
		return fantasy.Squad{}, fmt.Errorf("%w: player=%s league=%s", ErrNotFound, input.PlayerID, input.LeagueID)
	}
	selected := players[0]

	existing, exists, err := s.squadRepo.GetByUserAndLeague(ctx, input.UserID, input.LeagueID)
	if err != nil {
		return fantasy.Squad{}, fmt.Errorf("get existing squad: %w", err)
	}

	picks := make([]fantasy.SquadPick, 0, len(existing.Picks)+1)
	picks = append(picks, existing.Picks...)
	for _, p := range existing.Picks {
		if p.PlayerID == input.PlayerID {
			return fantasy.Squad{}, fmt.Errorf("%w: duplicate player id %s", ErrInvalidInput, input.PlayerID)
		}
	}
	picks = append(picks, fantasy.SquadPick{
		PlayerID: selected.ID,
		TeamID:   selected.TeamID,
		Position: selected.Position,
		Price:    selected.Price,
	})

	if err := fantasy.ValidatePicksPartial(picks, s.rules); err != nil {
		return fantasy.Squad{}, fmt.Errorf("validate squad picks: %w", err)
	}

	name := input.SquadName
	if name == "" {
		if exists && strings.TrimSpace(existing.Name) != "" {
			name = strings.TrimSpace(existing.Name)
		} else {
			name = defaultSquadName
		}
	}

	now := s.now().UTC()
	squadID := existing.ID
	createdAt := existing.CreatedAt
	if !exists {
		squadID, err = s.idGen.NewID()
		if err != nil {
			return fantasy.Squad{}, fmt.Errorf("generate squad id: %w", err)
		}
		createdAt = now
	}

	squad := fantasy.Squad{
		ID:        squadID,
		UserID:    input.UserID,
		LeagueID:  input.LeagueID,
		Name:      name,
		Picks:     picks,
		BudgetCap: s.rules.BudgetCap,
		CreatedAt: createdAt,
		UpdatedAt: now,
	}

	if err := squad.ValidateBasic(); err != nil {
		return fantasy.Squad{}, fmt.Errorf("validate squad: %w", err)
	}
	if err := s.squadRepo.Upsert(ctx, squad); err != nil {
		return fantasy.Squad{}, fmt.Errorf("upsert squad: %w", err)
	}

	s.logger.InfoContext(ctx, "player added to squad",
		"user_id", input.UserID,
		"league_id", input.LeagueID,
		"squad_id", squad.ID,
		"player_id", input.PlayerID,
		"player_count", len(squad.Picks),
	)

	return squad, nil
}

func (s *SquadService) GetUserSquad(ctx context.Context, userID, leagueID string) (fantasy.Squad, error) {
	userID = strings.TrimSpace(userID)
	leagueID = strings.TrimSpace(leagueID)
	if userID == "" || leagueID == "" {
		return fantasy.Squad{}, fmt.Errorf("%w: user_id and league_id are required", ErrInvalidInput)
	}

	squad, exists, err := s.squadRepo.GetByUserAndLeague(ctx, userID, leagueID)
	if err != nil {
		return fantasy.Squad{}, fmt.Errorf("get squad: %w", err)
	}
	if !exists {
		return fantasy.Squad{}, fmt.Errorf("%w: squad not found", ErrNotFound)
	}

	return squad, nil
}

func (s *SquadService) validateLeague(ctx context.Context, leagueID string) error {
	_, exists, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("get league by id: %w", err)
	}
	if !exists {
		return fmt.Errorf("%w: league=%s", ErrNotFound, leagueID)
	}

	return nil
}

func cleanPlayerIDs(playerIDs []string) ([]string, error) {
	cleaned := make([]string, 0, len(playerIDs))
	seen := make(map[string]struct{}, len(playerIDs))
	for _, id := range playerIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, fmt.Errorf("%w: player id cannot be empty", ErrInvalidInput)
		}
		if _, ok := seen[id]; ok {
			return nil, fmt.Errorf("%w: duplicate player id %s", ErrInvalidInput, id)
		}
		seen[id] = struct{}{}
		cleaned = append(cleaned, id)
	}

	return cleaned, nil
}
