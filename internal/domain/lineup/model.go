package lineup

import "time"

// Lineup stores one user's lineup for a league.
type Lineup struct {
	UserID        string
	LeagueID      string
	GoalkeeperID  string
	DefenderIDs   []string
	MidfielderIDs []string
	ForwardIDs    []string
	SubstituteIDs []string
	CaptainID     string
	ViceCaptainID string
	UpdatedAt     time.Time
}
