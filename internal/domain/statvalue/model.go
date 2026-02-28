package statvalue

import (
	"fmt"
	"strings"
	"time"
)

type Type struct {
	ExternalTypeID int64
	Name           string
	DeveloperName  string
	Code           string
	ModelType      string
	StatGroup      string
	Metadata       map[string]any
}

func (v Type) Validate() error {
	if v.ExternalTypeID <= 0 {
		return fmt.Errorf("external type id must be greater than zero")
	}
	if strings.TrimSpace(v.Name) == "" && strings.TrimSpace(v.DeveloperName) == "" {
		return fmt.Errorf("name or developer name is required")
	}
	return nil
}

type TeamValue struct {
	LeagueID           string
	SeasonRefID        int64
	TeamID             string
	ExternalTeamID     int64
	FixtureID          string
	ExternalFixtureID  int64
	StatTypeExternalID int64
	StatKey            string
	Scope              string
	ValueNum           *float64
	ValueText          string
	ValueJSON          map[string]any
	SourceUpdatedAt    *time.Time
	Metadata           map[string]any
}

func (v TeamValue) Validate() error {
	if strings.TrimSpace(v.LeagueID) == "" {
		return fmt.Errorf("league id is required")
	}
	if v.SeasonRefID <= 0 {
		return fmt.Errorf("season ref id must be greater than zero")
	}
	if strings.TrimSpace(v.TeamID) == "" && v.ExternalTeamID <= 0 {
		return fmt.Errorf("team identity is required")
	}
	if v.StatTypeExternalID <= 0 {
		return fmt.Errorf("stat type external id must be greater than zero")
	}
	if strings.TrimSpace(v.StatKey) == "" {
		return fmt.Errorf("stat key is required")
	}
	return nil
}

type PlayerValue struct {
	LeagueID           string
	SeasonRefID        int64
	PlayerID           string
	ExternalPlayerID   int64
	TeamID             string
	ExternalTeamID     int64
	FixtureID          string
	ExternalFixtureID  int64
	StatTypeExternalID int64
	StatKey            string
	Scope              string
	ValueNum           *float64
	ValueText          string
	ValueJSON          map[string]any
	SourceUpdatedAt    *time.Time
	Metadata           map[string]any
}

func (v PlayerValue) Validate() error {
	if strings.TrimSpace(v.LeagueID) == "" {
		return fmt.Errorf("league id is required")
	}
	if v.SeasonRefID <= 0 {
		return fmt.Errorf("season ref id must be greater than zero")
	}
	if strings.TrimSpace(v.PlayerID) == "" && v.ExternalPlayerID <= 0 {
		return fmt.Errorf("player identity is required")
	}
	if v.StatTypeExternalID <= 0 {
		return fmt.Errorf("stat type external id must be greater than zero")
	}
	if strings.TrimSpace(v.StatKey) == "" {
		return fmt.Errorf("stat key is required")
	}
	return nil
}
