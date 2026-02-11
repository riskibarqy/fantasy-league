package fantasy

import (
	"errors"
	"testing"

	"github.com/riskibarqy/fantasy-league/internal/domain/player"
)

func TestValidatePicks(t *testing.T) {
	rules := DefaultRules()
	validPicks := []SquadPick{
		{PlayerID: "p1", TeamID: "t1", Position: player.PositionGoalkeeper, Price: 80},
		{PlayerID: "p2", TeamID: "t1", Position: player.PositionDefender, Price: 80},
		{PlayerID: "p3", TeamID: "t2", Position: player.PositionDefender, Price: 80},
		{PlayerID: "p4", TeamID: "t3", Position: player.PositionDefender, Price: 80},
		{PlayerID: "p5", TeamID: "t1", Position: player.PositionMidfielder, Price: 90},
		{PlayerID: "p6", TeamID: "t2", Position: player.PositionMidfielder, Price: 90},
		{PlayerID: "p7", TeamID: "t3", Position: player.PositionMidfielder, Price: 90},
		{PlayerID: "p8", TeamID: "t4", Position: player.PositionMidfielder, Price: 70},
		{PlayerID: "p9", TeamID: "t2", Position: player.PositionForward, Price: 100},
		{PlayerID: "p10", TeamID: "t3", Position: player.PositionForward, Price: 100},
		{PlayerID: "p11", TeamID: "t4", Position: player.PositionForward, Price: 100},
	}

	tests := []struct {
		name      string
		mutate    func([]SquadPick, *Rules)
		targetErr error
	}{
		{
			name: "valid picks",
			mutate: func(_ []SquadPick, _ *Rules) {
			},
			targetErr: nil,
		},
		{
			name: "invalid size",
			mutate: func(picks []SquadPick, cfg *Rules) {
				_ = picks
				cfg.SquadSize = 10
			},
			targetErr: ErrInvalidSquadSize,
		},
		{
			name: "budget exceeded",
			mutate: func(picks []SquadPick, _ *Rules) {
				picks[0].Price = 500
				picks[1].Price = 500
				picks[2].Price = 500
			},
			targetErr: ErrExceededBudget,
		},
		{
			name: "team limit exceeded",
			mutate: func(picks []SquadPick, _ *Rules) {
				picks[3].TeamID = "t1"
			},
			targetErr: ErrExceededTeamLimit,
		},
		{
			name: "formation insufficient",
			mutate: func(picks []SquadPick, _ *Rules) {
				picks[0].Position = player.PositionForward
			},
			targetErr: ErrInsufficientFormation,
		},
		{
			name: "duplicate player",
			mutate: func(picks []SquadPick, _ *Rules) {
				picks[1].PlayerID = "p1"
			},
			targetErr: ErrDuplicatePlayerInSquad,
		},
		{
			name: "unknown position",
			mutate: func(picks []SquadPick, _ *Rules) {
				picks[0].Position = player.Position("UNK")
			},
			targetErr: ErrUnknownPlayerPosition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			picks := append([]SquadPick(nil), validPicks...)
			cfg := rules
			tt.mutate(picks, &cfg)

			err := ValidatePicks(picks, cfg)
			if tt.targetErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}

			if !errors.Is(err, tt.targetErr) {
				t.Fatalf("expected error %v, got %v", tt.targetErr, err)
			}
		})
	}
}
