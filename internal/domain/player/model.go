package player

import "fmt"

// Position represents football position categories used in fantasy rules.
type Position string

const (
	PositionGoalkeeper Position = "GK"
	PositionDefender   Position = "DEF"
	PositionMidfielder Position = "MID"
	PositionForward    Position = "FWD"
)

var AllPositions = map[Position]struct{}{
	PositionGoalkeeper: {},
	PositionDefender:   {},
	PositionMidfielder: {},
	PositionForward:    {},
}

// Player is a selectable athlete in a fantasy league pool.
type Player struct {
	ID          string
	LeagueID    string
	TeamID      string
	Name        string
	Position    Position
	Price       int64
	ImageURL    string
	PlayerRefID int64
}

func (p Player) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("player id is required")
	}
	if p.LeagueID == "" {
		return fmt.Errorf("player league id is required")
	}
	if p.TeamID == "" {
		return fmt.Errorf("player team id is required")
	}
	if p.Name == "" {
		return fmt.Errorf("player name is required")
	}
	if _, ok := AllPositions[p.Position]; !ok {
		return fmt.Errorf("invalid player position: %s", p.Position)
	}
	if p.Price <= 0 {
		return fmt.Errorf("player price must be greater than zero")
	}

	return nil
}
