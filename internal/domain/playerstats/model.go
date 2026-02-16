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
	EventID        int64
	FixtureID      string
	TeamID         string
	PlayerID       string
	AssistPlayerID string
	EventType      string
	Detail         string
	Minute         int
	ExtraMinute    int
}
