package scoring

import (
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
)

type GameweekLock struct {
	LeagueID   string
	Gameweek   int
	DeadlineAt time.Time
	IsLocked   bool
	LockedAt   *time.Time
}

type UserGameweekPoints struct {
	LeagueID     string
	Gameweek     int
	UserID       string
	Points       int
	CalculatedAt time.Time
}

type SquadSnapshot struct {
	LeagueID   string
	Gameweek   int
	Squad      fantasy.Squad
	CapturedAt time.Time
}

type LineupSnapshot struct {
	LeagueID   string
	Gameweek   int
	Lineup     lineup.Lineup
	CapturedAt time.Time
}
