package rawdata

import "time"

type Payload struct {
	Source          string
	EntityType      string
	EntityKey       string
	LeaguePublicID  string
	FixturePublicID string
	TeamPublicID    string
	PlayerPublicID  string
	PayloadJSON     string
	PayloadHash     string
	SourceUpdatedAt *time.Time
}
