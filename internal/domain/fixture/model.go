package fixture

import (
	"strings"
	"time"
)

const (
	StatusScheduled = "SCHEDULED"
	StatusLive      = "LIVE"
	StatusFinished  = "FINISHED"
	StatusCancelled = "CANCELLED"
	StatusPostponed = "POSTPONED"
)

// Fixture represents one scheduled match.
type Fixture struct {
	ID           string
	LeagueID     string
	Gameweek     int
	HomeTeam     string
	AwayTeam     string
	HomeTeamID   string
	AwayTeamID   string
	FixtureRefID int64
	KickoffAt    time.Time
	Venue        string
	HomeScore    *int
	AwayScore    *int
	Status       string
	WinnerTeamID string
	FinishedAt   *time.Time
}

func NormalizeStatus(value string) string {
	status := strings.ToUpper(strings.TrimSpace(value))
	if status == "" {
		return StatusScheduled
	}
	return status
}

func IsLiveStatus(status string) bool {
	switch NormalizeStatus(status) {
	case StatusLive, "IN_PLAY", "HT", "1H", "2H", "ET":
		return true
	default:
		return false
	}
}

func IsFinishedStatus(status string) bool {
	switch NormalizeStatus(status) {
	case StatusFinished, "FT", "AET", "PEN":
		return true
	default:
		return false
	}
}

func IsCancelledLikeStatus(status string) bool {
	switch NormalizeStatus(status) {
	case StatusCancelled, StatusPostponed, "ABANDONED":
		return true
	default:
		return false
	}
}
