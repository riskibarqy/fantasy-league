package usecase

import (
	"testing"

	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
)

func TestCalculateLineupPoints_CaptainAndViceCaptain(t *testing.T) {
	item := lineup.Lineup{
		GoalkeeperID:  "gk1",
		DefenderIDs:   []string{"d1", "d2", "d3"},
		MidfielderIDs: []string{"m1", "m2", "m3", "m4"},
		ForwardIDs:    []string{"f1", "f2", "f3"},
		CaptainID:     "m1",
		ViceCaptainID: "f1",
	}

	pointsByPlayer := map[string]int{
		"gk1": 2,
		"d1":  1, "d2": 1, "d3": 1,
		"m1": 5, "m2": 2, "m3": 2, "m4": 1,
		"f1": 3, "f2": 2, "f3": 1,
	}

	got := calculateLineupPoints(item, pointsByPlayer)
	if got != 26 {
		t.Fatalf("unexpected points: got=%d want=26", got)
	}

	pointsByPlayer["m1"] = 0
	got = calculateLineupPoints(item, pointsByPlayer)
	if got != 19 {
		t.Fatalf("unexpected vice-captain fallback points: got=%d want=19", got)
	}
}

func TestDeriveLineupFromSquad(t *testing.T) {
	squad := fantasy.Squad{
		UserID:   "u1",
		LeagueID: "l1",
		Picks: []fantasy.SquadPick{
			{PlayerID: "gk1", Position: player.PositionGoalkeeper},
			{PlayerID: "gk2", Position: player.PositionGoalkeeper},
			{PlayerID: "d1", Position: player.PositionDefender},
			{PlayerID: "d2", Position: player.PositionDefender},
			{PlayerID: "d3", Position: player.PositionDefender},
			{PlayerID: "d4", Position: player.PositionDefender},
			{PlayerID: "d5", Position: player.PositionDefender},
			{PlayerID: "m1", Position: player.PositionMidfielder},
			{PlayerID: "m2", Position: player.PositionMidfielder},
			{PlayerID: "m3", Position: player.PositionMidfielder},
			{PlayerID: "m4", Position: player.PositionMidfielder},
			{PlayerID: "m5", Position: player.PositionMidfielder},
			{PlayerID: "f1", Position: player.PositionForward},
			{PlayerID: "f2", Position: player.PositionForward},
			{PlayerID: "f3", Position: player.PositionForward},
		},
	}

	got, err := deriveLineupFromSquad(squad)
	if err != nil {
		t.Fatalf("derive lineup: %v", err)
	}

	starterCount := 1 + len(got.DefenderIDs) + len(got.MidfielderIDs) + len(got.ForwardIDs)
	if starterCount != 11 {
		t.Fatalf("unexpected starter count: got=%d want=11", starterCount)
	}
	if len(got.SubstituteIDs) != 4 {
		t.Fatalf("unexpected substitute count: got=%d want=4", len(got.SubstituteIDs))
	}
	if got.SubstituteIDs[0] != "gk2" {
		t.Fatalf("expected first substitute to be bench GK, got=%s", got.SubstituteIDs[0])
	}
	if got.CaptainID == "" || got.ViceCaptainID == "" {
		t.Fatalf("captain and vice-captain must be set")
	}
}
