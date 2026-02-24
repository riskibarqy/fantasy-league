package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
)

const (
	lineupStarterSize    = 11
	lineupSubstituteSize = 4
	lineupDefenderMin    = 3
	lineupDefenderMax    = 5
	lineupMidfielderMin  = 3
	lineupMidfielderMax  = 5
	lineupForwardMin     = 1
	lineupForwardMax     = 3
	lineupSquadGKTotal   = 2
	lineupSquadDEFSTotal = 5
	lineupSquadMIDTotal  = 5
	lineupSquadFWDTotal  = 3
)

type SaveLineupInput struct {
	UserID        string
	LeagueID      string
	GoalkeeperID  string
	DefenderIDs   []string
	MidfielderIDs []string
	ForwardIDs    []string
	SubstituteIDs []string
	CaptainID     string
	ViceCaptainID string
}

type LineupService struct {
	leagueRepo league.Repository
	playerRepo player.Repository
	lineupRepo lineup.Repository
	squadRepo  fantasy.Repository
	scorer     leagueScoringUpdater
	now        func() time.Time
}

func NewLineupService(
	leagueRepo league.Repository,
	playerRepo player.Repository,
	lineupRepo lineup.Repository,
	squadRepo fantasy.Repository,
) *LineupService {
	return &LineupService{
		leagueRepo: leagueRepo,
		playerRepo: playerRepo,
		lineupRepo: lineupRepo,
		squadRepo:  squadRepo,
		now:        time.Now,
	}
}

func (s *LineupService) SetScoringUpdater(scorer leagueScoringUpdater) {
	s.scorer = scorer
}

func (s *LineupService) GetByUserAndLeague(ctx context.Context, userID, leagueID string) (lineup.Lineup, bool, error) {
	userID = strings.TrimSpace(userID)
	leagueID = strings.TrimSpace(leagueID)
	if userID == "" || leagueID == "" {
		return lineup.Lineup{}, false, fmt.Errorf("%w: user_id and league_id are required", ErrInvalidInput)
	}

	item, exists, err := s.lineupRepo.GetByUserAndLeague(ctx, userID, leagueID)
	if err != nil {
		return lineup.Lineup{}, false, fmt.Errorf("get lineup by user and league: %w", err)
	}

	return item, exists, nil
}

func (s *LineupService) Save(ctx context.Context, input SaveLineupInput) (lineup.Lineup, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.LeagueID = strings.TrimSpace(input.LeagueID)
	input.GoalkeeperID = strings.TrimSpace(input.GoalkeeperID)
	input.CaptainID = strings.TrimSpace(input.CaptainID)
	input.ViceCaptainID = strings.TrimSpace(input.ViceCaptainID)

	if input.UserID == "" {
		return lineup.Lineup{}, fmt.Errorf("%w: user_id is required", ErrInvalidInput)
	}
	if input.LeagueID == "" {
		return lineup.Lineup{}, fmt.Errorf("%w: league_id is required", ErrInvalidInput)
	}

	if err := s.validateLeague(ctx, input.LeagueID); err != nil {
		return lineup.Lineup{}, err
	}
	if s.scorer != nil {
		if err := s.scorer.EnsureLeagueUpToDate(ctx, input.LeagueID); err != nil {
			return lineup.Lineup{}, fmt.Errorf("ensure league scoring before lineup save: %w", err)
		}
	}

	defenderIDs, err := normalizeIDs(input.DefenderIDs)
	if err != nil {
		return lineup.Lineup{}, err
	}
	midfielderIDs, err := normalizeIDs(input.MidfielderIDs)
	if err != nil {
		return lineup.Lineup{}, err
	}
	forwardIDs, err := normalizeIDs(input.ForwardIDs)
	if err != nil {
		return lineup.Lineup{}, err
	}
	substituteIDs, err := normalizeIDs(input.SubstituteIDs)
	if err != nil {
		return lineup.Lineup{}, err
	}

	if len(defenderIDs) < lineupDefenderMin || len(defenderIDs) > lineupDefenderMax {
		return lineup.Lineup{}, fmt.Errorf("%w: defender count must be between %d and %d", ErrInvalidInput, lineupDefenderMin, lineupDefenderMax)
	}
	if len(midfielderIDs) < lineupMidfielderMin || len(midfielderIDs) > lineupMidfielderMax {
		return lineup.Lineup{}, fmt.Errorf("%w: midfielder count must be between %d and %d", ErrInvalidInput, lineupMidfielderMin, lineupMidfielderMax)
	}
	if len(forwardIDs) < lineupForwardMin || len(forwardIDs) > lineupForwardMax {
		return lineup.Lineup{}, fmt.Errorf("%w: forward count must be between %d and %d", ErrInvalidInput, lineupForwardMin, lineupForwardMax)
	}
	if len(substituteIDs) != lineupSubstituteSize {
		return lineup.Lineup{}, fmt.Errorf("%w: substitute bench must contain exactly 4 players", ErrInvalidInput)
	}

	starters := append([]string{input.GoalkeeperID}, defenderIDs...)
	starters = append(starters, midfielderIDs...)
	starters = append(starters, forwardIDs...)

	if len(starters) != lineupStarterSize {
		return lineup.Lineup{}, fmt.Errorf("%w: starting lineup must contain 11 players", ErrInvalidInput)
	}

	starterSet := make(map[string]struct{}, len(starters))
	for _, id := range starters {
		id = strings.TrimSpace(id)
		if id == "" {
			return lineup.Lineup{}, fmt.Errorf("%w: starter player id cannot be empty", ErrInvalidInput)
		}
		if _, exists := starterSet[id]; exists {
			return lineup.Lineup{}, fmt.Errorf("%w: duplicate starter player id %s", ErrInvalidInput, id)
		}
		starterSet[id] = struct{}{}
	}

	for _, id := range substituteIDs {
		if _, exists := starterSet[id]; exists {
			return lineup.Lineup{}, fmt.Errorf("%w: substitutes must be different from starters", ErrInvalidInput)
		}
	}

	allIDs := append(append([]string(nil), starters...), substituteIDs...)
	allSet := make(map[string]struct{}, len(allIDs))
	for _, id := range allIDs {
		if _, exists := allSet[id]; exists {
			return lineup.Lineup{}, fmt.Errorf("%w: duplicate player id in squad %s", ErrInvalidInput, id)
		}
		allSet[id] = struct{}{}
	}

	if len(allIDs) != lineupStarterSize+lineupSubstituteSize {
		return lineup.Lineup{}, fmt.Errorf("%w: lineup must contain 15 players", ErrInvalidInput)
	}

	if _, ok := starterSet[input.CaptainID]; !ok {
		return lineup.Lineup{}, fmt.Errorf("%w: captain must be in starters", ErrInvalidInput)
	}
	if _, ok := starterSet[input.ViceCaptainID]; !ok {
		return lineup.Lineup{}, fmt.Errorf("%w: vice captain must be in starters", ErrInvalidInput)
	}
	if input.CaptainID == input.ViceCaptainID {
		return lineup.Lineup{}, fmt.Errorf("%w: captain and vice captain must be different", ErrInvalidInput)
	}

	players, err := s.playerRepo.GetByIDs(ctx, input.LeagueID, allIDs)
	if err != nil {
		return lineup.Lineup{}, fmt.Errorf("get players by ids: %w", err)
	}
	if len(players) != len(allIDs) {
		return lineup.Lineup{}, fmt.Errorf("%w: some players are not available in league", ErrInvalidInput)
	}

	playersByID := make(map[string]player.Player, len(players))
	for _, p := range players {
		playersByID[p.ID] = p
	}

	if playersByID[input.GoalkeeperID].Position != player.PositionGoalkeeper {
		return lineup.Lineup{}, fmt.Errorf("%w: goalkeeper slot must contain a GK", ErrInvalidInput)
	}

	if err := validatePositionIDs(defenderIDs, player.PositionDefender, playersByID); err != nil {
		return lineup.Lineup{}, err
	}
	if err := validatePositionIDs(midfielderIDs, player.PositionMidfielder, playersByID); err != nil {
		return lineup.Lineup{}, err
	}
	if err := validatePositionIDs(forwardIDs, player.PositionForward, playersByID); err != nil {
		return lineup.Lineup{}, err
	}
	benchGoalkeeperCount := 0
	benchDefenderCount := 0
	benchMidfielderCount := 0
	benchForwardCount := 0
	for _, id := range substituteIDs {
		p, ok := playersByID[id]
		if !ok {
			return lineup.Lineup{}, fmt.Errorf("%w: unknown player id %s", ErrInvalidInput, id)
		}
		switch p.Position {
		case player.PositionGoalkeeper:
			benchGoalkeeperCount++
		case player.PositionDefender:
			benchDefenderCount++
		case player.PositionMidfielder:
			benchMidfielderCount++
		case player.PositionForward:
			benchForwardCount++
		}
	}

	expectedBenchGoalkeeperCount := lineupSquadGKTotal - 1
	expectedBenchDefenderCount := lineupSquadDEFSTotal - len(defenderIDs)
	expectedBenchMidfielderCount := lineupSquadMIDTotal - len(midfielderIDs)
	expectedBenchForwardCount := lineupSquadFWDTotal - len(forwardIDs)
	if expectedBenchDefenderCount < 0 || expectedBenchMidfielderCount < 0 || expectedBenchForwardCount < 0 {
		return lineup.Lineup{}, fmt.Errorf("%w: invalid formation for configured squad composition", ErrInvalidInput)
	}
	if expectedBenchGoalkeeperCount+expectedBenchDefenderCount+expectedBenchMidfielderCount+expectedBenchForwardCount != lineupSubstituteSize {
		return lineup.Lineup{}, fmt.Errorf("%w: invalid bench size for selected formation", ErrInvalidInput)
	}
	if benchGoalkeeperCount != expectedBenchGoalkeeperCount ||
		benchDefenderCount != expectedBenchDefenderCount ||
		benchMidfielderCount != expectedBenchMidfielderCount ||
		benchForwardCount != expectedBenchForwardCount {
		return lineup.Lineup{}, fmt.Errorf(
			"%w: bench composition must be GK=%d DEF=%d MID=%d FWD=%d for this formation",
			ErrInvalidInput,
			expectedBenchGoalkeeperCount,
			expectedBenchDefenderCount,
			expectedBenchMidfielderCount,
			expectedBenchForwardCount,
		)
	}

	existingSquad, exists, err := s.squadRepo.GetByUserAndLeague(ctx, input.UserID, input.LeagueID)
	if err != nil {
		return lineup.Lineup{}, fmt.Errorf("get user squad before save lineup: %w", err)
	}
	if !exists {
		return lineup.Lineup{}, fmt.Errorf("%w: user must pick fantasy squad before saving lineup", ErrInvalidInput)
	}

	if len(existingSquad.Picks) != lineupStarterSize+lineupSubstituteSize {
		return lineup.Lineup{}, fmt.Errorf("%w: user squad must contain exactly 15 players", ErrInvalidInput)
	}

	squadPlayerSet := make(map[string]struct{}, len(existingSquad.Picks))
	for _, pick := range existingSquad.Picks {
		squadPlayerSet[pick.PlayerID] = struct{}{}
	}

	for _, playerID := range allIDs {
		if _, ok := squadPlayerSet[playerID]; !ok {
			return lineup.Lineup{}, fmt.Errorf("%w: player %s is not part of user fantasy squad", ErrInvalidInput, playerID)
		}
	}

	item := lineup.Lineup{
		UserID:        input.UserID,
		LeagueID:      input.LeagueID,
		GoalkeeperID:  input.GoalkeeperID,
		DefenderIDs:   defenderIDs,
		MidfielderIDs: midfielderIDs,
		ForwardIDs:    forwardIDs,
		SubstituteIDs: substituteIDs,
		CaptainID:     input.CaptainID,
		ViceCaptainID: input.ViceCaptainID,
		UpdatedAt:     s.now().UTC(),
	}

	if err := s.lineupRepo.Upsert(ctx, item); err != nil {
		return lineup.Lineup{}, fmt.Errorf("save lineup: %w", err)
	}

	return item, nil
}

func (s *LineupService) validateLeague(ctx context.Context, leagueID string) error {
	_, exists, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("get league by id: %w", err)
	}
	if !exists {
		return fmt.Errorf("%w: league=%s", ErrNotFound, leagueID)
	}

	return nil
}

func normalizeIDs(ids []string) ([]string, error) {
	cleaned := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, fmt.Errorf("%w: player id cannot be empty", ErrInvalidInput)
		}
		cleaned = append(cleaned, id)
	}
	return cleaned, nil
}

func validatePositionIDs(ids []string, expected player.Position, playersByID map[string]player.Player) error {
	for _, id := range ids {
		p, ok := playersByID[id]
		if !ok {
			return fmt.Errorf("%w: unknown player id %s", ErrInvalidInput, id)
		}
		if p.Position != expected {
			return fmt.Errorf("%w: player %s must have position %s", ErrInvalidInput, id, expected)
		}
	}

	return nil
}
