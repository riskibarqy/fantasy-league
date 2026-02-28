package sportmonks

import (
	"testing"

	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func TestEstimateFantasyPoints_FPLStyleGoalAndCleanSheetByPosition(t *testing.T) {
	t.Parallel()

	gk := usecase.ExternalPlayerFixtureStat{
		Position:       "GK",
		MinutesPlayed:  90,
		Goals:          1,
		Assists:        0,
		CleanSheet:     true,
		Saves:          6,
		GoalsConceded:  0,
		YellowCards:    0,
		RedCards:       0,
		BonusPoints:    3,
		PenaltiesSaved: 1,
	}
	// GK: goal 6 + clean sheet 4 + saves 2 + minutes 2 + pen saved 5 + bonus 3 = 22
	if got := estimateFantasyPoints(gk); got != 22 {
		t.Fatalf("gk fantasy points mismatch: got=%d want=22", got)
	}

	mid := usecase.ExternalPlayerFixtureStat{
		Position:      "MID",
		MinutesPlayed: 90,
		Goals:         1,
		Assists:       1,
		CleanSheet:    true,
		YellowCards:   1,
		BonusPoints:   2,
	}
	// MID: goal 5 + assist 3 + clean sheet 1 + minutes 2 - yellow 1 + bonus 2 = 12
	if got := estimateFantasyPoints(mid); got != 12 {
		t.Fatalf("mid fantasy points mismatch: got=%d want=12", got)
	}
}

func TestApplyFPLBonusByBPS_TieRules(t *testing.T) {
	t.Parallel()

	stats := map[string]usecase.ExternalPlayerFixtureStat{
		"10:1": {FixtureExternalID: 10, PlayerExternalID: 1, BPS: 30},
		"10:2": {FixtureExternalID: 10, PlayerExternalID: 2, BPS: 30},
		"10:3": {FixtureExternalID: 10, PlayerExternalID: 3, BPS: 28},
		"10:4": {FixtureExternalID: 10, PlayerExternalID: 4, BPS: 25},
	}

	applyFPLBonusByBPS(10, stats)

	if got := stats["10:1"].BonusPoints; got != 3 {
		t.Fatalf("player 1 bonus mismatch: got=%d want=3", got)
	}
	if got := stats["10:2"].BonusPoints; got != 3 {
		t.Fatalf("player 2 bonus mismatch: got=%d want=3", got)
	}
	// two-way tie for first means next rank is third => 1 point
	if got := stats["10:3"].BonusPoints; got != 1 {
		t.Fatalf("player 3 bonus mismatch: got=%d want=1", got)
	}
	if got := stats["10:4"].BonusPoints; got != 0 {
		t.Fatalf("player 4 bonus mismatch: got=%d want=0", got)
	}
}
