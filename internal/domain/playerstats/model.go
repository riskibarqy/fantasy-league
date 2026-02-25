package playerstats

import "time"

type SeasonStats struct {
	MinutesPlayed int
	Goals         int
	Assists       int
	CleanSheets   int
	YellowCards   int
	RedCards      int
	Saves         int
	Appearances   int
	TotalPoints   int
}

type MatchHistory struct {
	FixtureID     string
	Gameweek      int
	KickoffAt     time.Time
	HomeTeam      string
	AwayTeam      string
	TeamID        string
	MinutesPlayed int
	Goals         int
	Assists       int
	CleanSheet    bool
	YellowCards   int
	RedCards      int
	Saves         int
	FantasyPoints int
}

type FixtureEvent struct {
	EventID                int64
	FixtureID              string
	FixtureExternalID      int64
	TeamID                 string
	TeamExternalID         int64
	PlayerID               string
	PlayerExternalID       int64
	AssistPlayerID         string
	AssistPlayerExternalID int64
	EventType              string
	Detail                 string
	Minute                 int
	ExtraMinute            int
	Metadata               map[string]any
}

type FixtureStat struct {
	FixtureID         string
	FixtureExternalID int64
	PlayerID          string
	PlayerExternalID  int64
	TeamID            string
	TeamExternalID    int64
	MinutesPlayed     int
	Goals             int
	Assists           int
	CleanSheet        bool
	YellowCards       int
	RedCards          int
	Saves             int
	FantasyPoints     int
	AdvancedStats     map[string]any
}
