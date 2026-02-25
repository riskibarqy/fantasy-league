package league

import "fmt"

// League is a football league supported by the fantasy platform.
type League struct {
	ID          string
	Name        string
	CountryCode string
	Season      string
	IsDefault   bool
	LeagueRefID int64
}

func (l League) Validate() error {
	if l.ID == "" {
		return fmt.Errorf("league id is required")
	}
	if l.Name == "" {
		return fmt.Errorf("league name is required")
	}
	if l.CountryCode == "" {
		return fmt.Errorf("league country code is required")
	}
	if l.Season == "" {
		return fmt.Errorf("league season is required")
	}

	return nil
}
