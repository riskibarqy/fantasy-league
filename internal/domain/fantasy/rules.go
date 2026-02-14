package fantasy

import (
	"errors"
	"fmt"

	"github.com/riskibarqy/fantasy-league/internal/domain/player"
)

var (
	ErrInvalidSquadSize       = errors.New("invalid squad size")
	ErrExceededBudget         = errors.New("budget cap exceeded")
	ErrExceededTeamLimit      = errors.New("max players from same team exceeded")
	ErrInsufficientFormation  = errors.New("minimum formation requirement not met")
	ErrUnknownPlayerPosition  = errors.New("unknown player position")
	ErrDuplicatePlayerInSquad = errors.New("duplicate player in squad")
)

// Rules stores fantasy roster validation parameters.
type Rules struct {
	SquadSize         int
	BudgetCap         int64
	MaxPlayersPerTeam int
	MinByPosition     map[player.Position]int
}

func DefaultRules() Rules {
	return Rules{
		SquadSize:         11,
		BudgetCap:         1000,
		MaxPlayersPerTeam: 3,
		MinByPosition: map[player.Position]int{
			player.PositionGoalkeeper: 1,
			player.PositionDefender:   3,
			player.PositionMidfielder: 3,
			player.PositionForward:    1,
		},
	}
}

func ValidatePicks(picks []SquadPick, rules Rules) error {
	if len(picks) != rules.SquadSize {
		return fmt.Errorf("%w: expected %d, got %d", ErrInvalidSquadSize, rules.SquadSize, len(picks))
	}

	teamCounter := make(map[string]int)
	positionCounter := make(map[player.Position]int)
	playerSet := make(map[string]struct{})
	var totalCost int64

	for _, pick := range picks {
		if pick.PlayerID == "" {
			return fmt.Errorf("player id is required")
		}
		if _, exists := playerSet[pick.PlayerID]; exists {
			return fmt.Errorf("%w: %s", ErrDuplicatePlayerInSquad, pick.PlayerID)
		}
		playerSet[pick.PlayerID] = struct{}{}

		if _, ok := player.AllPositions[pick.Position]; !ok {
			return fmt.Errorf("%w: %s", ErrUnknownPlayerPosition, pick.Position)
		}
		if pick.TeamID == "" {
			return fmt.Errorf("team id is required for player %s", pick.PlayerID)
		}
		if pick.Price <= 0 {
			return fmt.Errorf("player price must be greater than zero: %s", pick.PlayerID)
		}

		teamCounter[pick.TeamID]++
		if teamCounter[pick.TeamID] > rules.MaxPlayersPerTeam {
			return fmt.Errorf("%w: team=%s max=%d", ErrExceededTeamLimit, pick.TeamID, rules.MaxPlayersPerTeam)
		}

		positionCounter[pick.Position]++
		totalCost += pick.Price
	}

	if totalCost > rules.BudgetCap {
		return fmt.Errorf("%w: cap=%d used=%d", ErrExceededBudget, rules.BudgetCap, totalCost)
	}

	for pos, minRequired := range rules.MinByPosition {
		if positionCounter[pos] < minRequired {
			return fmt.Errorf("%w: pos=%s min=%d current=%d", ErrInsufficientFormation, pos, minRequired, positionCounter[pos])
		}
	}

	return nil
}

// ValidatePicksPartial validates draft picks while user builds squad incrementally.
// It does not require exact squad size or minimum formation constraints yet.
func ValidatePicksPartial(picks []SquadPick, rules Rules) error {
	if len(picks) == 0 {
		return fmt.Errorf("%w: expected at least 1, got 0", ErrInvalidSquadSize)
	}
	if len(picks) > rules.SquadSize {
		return fmt.Errorf("%w: expected at most %d, got %d", ErrInvalidSquadSize, rules.SquadSize, len(picks))
	}

	teamCounter := make(map[string]int)
	playerSet := make(map[string]struct{})
	var totalCost int64

	for _, pick := range picks {
		if pick.PlayerID == "" {
			return fmt.Errorf("player id is required")
		}
		if _, exists := playerSet[pick.PlayerID]; exists {
			return fmt.Errorf("%w: %s", ErrDuplicatePlayerInSquad, pick.PlayerID)
		}
		playerSet[pick.PlayerID] = struct{}{}

		if _, ok := player.AllPositions[pick.Position]; !ok {
			return fmt.Errorf("%w: %s", ErrUnknownPlayerPosition, pick.Position)
		}
		if pick.TeamID == "" {
			return fmt.Errorf("team id is required for player %s", pick.PlayerID)
		}
		if pick.Price <= 0 {
			return fmt.Errorf("player price must be greater than zero: %s", pick.PlayerID)
		}

		teamCounter[pick.TeamID]++
		if teamCounter[pick.TeamID] > rules.MaxPlayersPerTeam {
			return fmt.Errorf("%w: team=%s max=%d", ErrExceededTeamLimit, pick.TeamID, rules.MaxPlayersPerTeam)
		}

		totalCost += pick.Price
	}

	if totalCost > rules.BudgetCap {
		return fmt.Errorf("%w: cap=%d used=%d", ErrExceededBudget, rules.BudgetCap, totalCost)
	}

	return nil
}
