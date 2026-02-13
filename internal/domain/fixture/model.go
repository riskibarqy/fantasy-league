package fixture

import "time"

// Fixture represents one scheduled match.
type Fixture struct {
	ID        string
	LeagueID  string
	Gameweek  int
	HomeTeam  string
	AwayTeam  string
	KickoffAt time.Time
	Venue     string
}
