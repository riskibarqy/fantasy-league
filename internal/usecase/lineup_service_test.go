package usecase

import (
	"errors"
	"testing"

	"github.com/riskibarqy/fantasy-league/internal/infrastructure/repository/memory"
)

func TestLineupService_Save_ValidLineup(t *testing.T) {
	leagueRepo := memory.NewLeagueRepository(memory.SeedLeagues())
	playerRepo := memory.NewPlayerRepository(memory.SeedPlayers())
	lineupRepo := memory.NewLineupRepository()

	svc := NewLineupService(leagueRepo, playerRepo, lineupRepo)

	lineup, err := svc.Save(t.Context(), SaveLineupInput{
		UserID:        "user-1",
		LeagueID:      memory.LeagueIDLiga1Indonesia,
		GoalkeeperID:  "idn-gk-01",
		DefenderIDs:   []string{"idn-def-01", "idn-def-02", "idn-def-03", "idn-def-04"},
		MidfielderIDs: []string{"idn-mid-01", "idn-mid-02", "idn-mid-03"},
		ForwardIDs:    []string{"idn-fwd-01", "idn-fwd-02", "idn-fwd-03"},
		SubstituteIDs: []string{"idn-gk-02", "idn-def-05", "idn-mid-04", "idn-mid-05", "idn-mid-06"},
		CaptainID:     "idn-mid-01",
		ViceCaptainID: "idn-fwd-01",
	})
	if err != nil {
		t.Fatalf("save lineup failed: %v", err)
	}

	if lineup.LeagueID != memory.LeagueIDLiga1Indonesia {
		t.Fatalf("unexpected league id: %s", lineup.LeagueID)
	}
	if len(lineup.SubstituteIDs) != 5 {
		t.Fatalf("unexpected substitute count: %d", len(lineup.SubstituteIDs))
	}
}

func TestLineupService_Save_InvalidFormation(t *testing.T) {
	leagueRepo := memory.NewLeagueRepository(memory.SeedLeagues())
	playerRepo := memory.NewPlayerRepository(memory.SeedPlayers())
	lineupRepo := memory.NewLineupRepository()

	svc := NewLineupService(leagueRepo, playerRepo, lineupRepo)

	_, err := svc.Save(t.Context(), SaveLineupInput{
		UserID:        "user-1",
		LeagueID:      memory.LeagueIDLiga1Indonesia,
		GoalkeeperID:  "idn-gk-01",
		DefenderIDs:   []string{"idn-def-01"},
		MidfielderIDs: []string{"idn-mid-01", "idn-mid-02", "idn-mid-03", "idn-mid-04", "idn-mid-05", "idn-mid-06"},
		ForwardIDs:    []string{"idn-fwd-01", "idn-fwd-02", "idn-fwd-03"},
		SubstituteIDs: []string{"idn-gk-02", "idn-def-02", "idn-def-03", "idn-def-04", "idn-def-05"},
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

	svc := NewLineupService(leagueRepo, playerRepo, lineupRepo)

	_, err := svc.Save(t.Context(), SaveLineupInput{
		UserID:        "user-1",
		LeagueID:      memory.LeagueIDLiga1Indonesia,
		GoalkeeperID:  "idn-gk-01",
		DefenderIDs:   []string{"idn-def-01", "idn-def-02", "idn-def-03", "idn-def-04"},
		MidfielderIDs: []string{"idn-mid-01", "idn-mid-02", "idn-mid-03"},
		ForwardIDs:    []string{"idn-fwd-01", "idn-fwd-02", "idn-fwd-03"},
		SubstituteIDs: []string{"idn-gk-02", "idn-def-05", "idn-mid-04", "idn-mid-05", "idn-def-01"},
		CaptainID:     "idn-mid-01",
		ViceCaptainID: "idn-fwd-01",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
