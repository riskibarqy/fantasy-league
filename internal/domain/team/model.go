package team

import "fmt"

// Team is a real football club inside a league.
type Team struct {
	ID       string
	LeagueID string
	Name     string
	Short    string
}

func (t Team) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("team id is required")
	}
	if t.LeagueID == "" {
		return fmt.Errorf("team league id is required")
	}
	if t.Name == "" {
		return fmt.Errorf("team name is required")
	}

	return nil
}
