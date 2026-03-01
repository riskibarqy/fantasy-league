package leaguestanding

import "time"

// Standing represents a league table row for one team.
type Standing struct {
	LeagueID        string
	TeamID          string
	IsLive          bool
	Gameweek        int
	Position        int
	Played          int
	Won             int
	Draw            int
	Lost            int
	GoalsFor        int
	GoalsAgainst    int
	GoalDifference  int
	Points          int
	Form            string
	SourceUpdatedAt *time.Time
}
