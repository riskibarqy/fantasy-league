package usecase

import (
	"errors"
	"testing"

	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	"github.com/riskibarqy/fantasy-league/internal/infrastructure/repository/memory"
)

func TestLineupService_Save_ValidLineup(t *testing.T) {
	leagueRepo := memory.NewLeagueRepository(memory.SeedLeagues())
	playerRepo := memory.NewPlayerRepository(memory.SeedPlayers())
	lineupRepo := memory.NewLineupRepository()
	squadRepo := memory.NewSquadRepository()

	svc := NewLineupService(leagueRepo, playerRepo, lineupRepo, squadRepo)
	seedValidSquad(t, squadRepo, "user-1")

	lineup, err := svc.Save(t.Context(), SaveLineupInput{
		UserID:        "user-1",
		LeagueID:      memory.LeagueIDLiga1Indonesia,
		GoalkeeperID:  "idn-gk-01",
		DefenderIDs:   []string{"idn-def-01", "idn-def-02", "idn-def-03", "idn-def-04"},
		MidfielderIDs: []string{"idn-mid-01", "idn-mid-02", "idn-mid-03"},
		ForwardIDs:    []string{"idn-fwd-01", "idn-fwd-02", "idn-fwd-03"},
		SubstituteIDs: []string{"idn-gk-02", "idn-def-05", "idn-mid-04", "idn-mid-05"},
		CaptainID:     "idn-mid-01",
		ViceCaptainID: "idn-fwd-01",
	})
	if err != nil {
		t.Fatalf("save lineup failed: %v", err)
	}

	if lineup.LeagueID != memory.LeagueIDLiga1Indonesia {
		t.Fatalf("unexpected league id: %s", lineup.LeagueID)
	}
	if len(lineup.SubstituteIDs) != 4 {
		t.Fatalf("unexpected substitute count: %d", len(lineup.SubstituteIDs))
	}
}

func TestLineupService_Save_InvalidFormation(t *testing.T) {
	leagueRepo := memory.NewLeagueRepository(memory.SeedLeagues())
	playerRepo := memory.NewPlayerRepository(memory.SeedPlayers())
	lineupRepo := memory.NewLineupRepository()
	squadRepo := memory.NewSquadRepository()

	svc := NewLineupService(leagueRepo, playerRepo, lineupRepo, squadRepo)
	seedValidSquad(t, squadRepo, "user-1")

	_, err := svc.Save(t.Context(), SaveLineupInput{
		UserID:        "user-1",
		LeagueID:      memory.LeagueIDLiga1Indonesia,
		GoalkeeperID:  "idn-gk-01",
		DefenderIDs:   []string{"idn-def-01"},
		MidfielderIDs: []string{"idn-mid-01", "idn-mid-02", "idn-mid-03", "idn-mid-04", "idn-mid-05", "idn-mid-06"},
		ForwardIDs:    []string{"idn-fwd-01", "idn-fwd-02", "idn-fwd-03"},
		SubstituteIDs: []string{"idn-gk-02", "idn-def-02", "idn-def-03", "idn-def-04"},
		CaptainID:     "idn-mid-01",
		ViceCaptainID: "idn-fwd-01",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestLineupService_Save_RejectOverlappingSubstitute(t *testing.T) {
	leagueRepo := memory.NewLeagueRepository(memory.SeedLeagues())
	playerRepo := memory.NewPlayerRepository(memory.SeedPlayers())
	lineupRepo := memory.NewLineupRepository()
	squadRepo := memory.NewSquadRepository()

	svc := NewLineupService(leagueRepo, playerRepo, lineupRepo, squadRepo)
	seedValidSquad(t, squadRepo, "user-1")

	_, err := svc.Save(t.Context(), SaveLineupInput{
		UserID:        "user-1",
		LeagueID:      memory.LeagueIDLiga1Indonesia,
		GoalkeeperID:  "idn-gk-01",
		DefenderIDs:   []string{"idn-def-01", "idn-def-02", "idn-def-03", "idn-def-04"},
		MidfielderIDs: []string{"idn-mid-01", "idn-mid-02", "idn-mid-03"},
		ForwardIDs:    []string{"idn-fwd-01", "idn-fwd-02", "idn-fwd-03"},
		SubstituteIDs: []string{"idn-gk-02", "idn-def-05", "idn-mid-04", "idn-def-01"},
		CaptainID:     "idn-mid-01",
		ViceCaptainID: "idn-fwd-01",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestLineupService_Save_RejectPlayerOutsideSquad(t *testing.T) {
	leagueRepo := memory.NewLeagueRepository(memory.SeedLeagues())
	playerRepo := memory.NewPlayerRepository(memory.SeedPlayers())
	lineupRepo := memory.NewLineupRepository()
	squadRepo := memory.NewSquadRepository()

	svc := NewLineupService(leagueRepo, playerRepo, lineupRepo, squadRepo)
	seedValidSquad(t, squadRepo, "user-1")

	_, err := svc.Save(t.Context(), SaveLineupInput{
		UserID:        "user-1",
		LeagueID:      memory.LeagueIDLiga1Indonesia,
		GoalkeeperID:  "idn-gk-03",
		DefenderIDs:   []string{"idn-def-01", "idn-def-02", "idn-def-03", "idn-def-04"},
		MidfielderIDs: []string{"idn-mid-01", "idn-mid-02", "idn-mid-03"},
		ForwardIDs:    []string{"idn-fwd-01", "idn-fwd-02", "idn-fwd-03"},
		SubstituteIDs: []string{"idn-gk-02", "idn-def-05", "idn-mid-04", "idn-mid-05"},
		CaptainID:     "idn-mid-01",
		ViceCaptainID: "idn-fwd-01",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func seedValidSquad(t *testing.T, squadRepo *memory.SquadRepository, userID string) {
	t.Helper()

	squad := fantasy.Squad{
		ID:        "squad-" + userID,
		UserID:    userID,
		LeagueID:  memory.LeagueIDLiga1Indonesia,
		Name:      "Test Squad",
		BudgetCap: 1000,
		Picks: []fantasy.SquadPick{
			{PlayerID: "idn-gk-01", TeamID: "idn-persija", Position: player.PositionGoalkeeper, Price: 90},
			{PlayerID: "idn-gk-02", TeamID: "idn-persib", Position: player.PositionGoalkeeper, Price: 85},
			{PlayerID: "idn-def-01", TeamID: "idn-persija", Position: player.PositionDefender, Price: 88},
			{PlayerID: "idn-def-02", TeamID: "idn-persib", Position: player.PositionDefender, Price: 92},
			{PlayerID: "idn-def-03", TeamID: "idn-persebaya", Position: player.PositionDefender, Price: 84},
			{PlayerID: "idn-def-04", TeamID: "idn-baliutd", Position: player.PositionDefender, Price: 80},
			{PlayerID: "idn-def-05", TeamID: "idn-persebaya", Position: player.PositionDefender, Price: 72},
			{PlayerID: "idn-mid-01", TeamID: "idn-persija", Position: player.PositionMidfielder, Price: 98},
			{PlayerID: "idn-mid-02", TeamID: "idn-persib", Position: player.PositionMidfielder, Price: 99},
			{PlayerID: "idn-mid-03", TeamID: "idn-persebaya", Position: player.PositionMidfielder, Price: 95},
			{PlayerID: "idn-mid-04", TeamID: "idn-baliutd", Position: player.PositionMidfielder, Price: 97},
			{PlayerID: "idn-mid-05", TeamID: "idn-baliutd", Position: player.PositionMidfielder, Price: 90},
			{PlayerID: "idn-fwd-01", TeamID: "idn-persija", Position: player.PositionForward, Price: 105},
			{PlayerID: "idn-fwd-02", TeamID: "idn-persib", Position: player.PositionForward, Price: 108},
			{PlayerID: "idn-fwd-03", TeamID: "idn-persebaya", Position: player.PositionForward, Price: 100},
		},
	}

	if err := squadRepo.Upsert(t.Context(), squad); err != nil {
		t.Fatalf("seed squad: %v", err)
	}
}
