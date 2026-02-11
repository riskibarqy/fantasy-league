package fantasy

import (
	"fmt"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/player"
)

// SquadPick represents one selected player in a user's fantasy squad.
type SquadPick struct {
	PlayerID string
	TeamID   string
	Position player.Position
	Price    int64
}

// Squad contains user team composition for one league.
type Squad struct {
	ID        string
	UserID    string
	LeagueID  string
	Name      string
	Picks     []SquadPick
	BudgetCap int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s Squad) ValidateBasic() error {
	if s.ID == "" {
		return fmt.Errorf("squad id is required")
	}
	if s.UserID == "" {
		return fmt.Errorf("user id is required")
	}
	if s.LeagueID == "" {
		return fmt.Errorf("league id is required")
	}
	if s.Name == "" {
		return fmt.Errorf("squad name is required")
	}
	if s.BudgetCap <= 0 {
		return fmt.Errorf("budget cap must be greater than zero")
	}
	if len(s.Picks) == 0 {
		return fmt.Errorf("squad picks are required")
	}

	return nil
}
