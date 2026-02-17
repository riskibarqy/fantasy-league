package teamstats

import "time"

type SeasonStats struct {
	Appearances          int
	AveragePossessionPct float64
	TotalShots           int
	TotalShotsOnTarget   int
	TotalCorners         int
	TotalFouls           int
	TotalOffsides        int
}

type MatchHistory struct {
	FixtureID      string
	Gameweek       int
	KickoffAt      time.Time
	HomeTeam       string
	AwayTeam       string
	OpponentTeamID string
	IsHome         bool
	PossessionPct  float64
	Shots          int
	ShotsOnTarget  int
	Corners        int
	Fouls          int
	Offsides       int
}
