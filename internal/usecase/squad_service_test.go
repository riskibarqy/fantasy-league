package usecase

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/infrastructure/repository/memory"
)

type staticIDGenerator struct {
	id string
}

func (g staticIDGenerator) NewID() (string, error) {
	return g.id, nil
}

func TestSquadService_UpsertSquad_CreateThenUpdate(t *testing.T) {
	leagueRepo := memory.NewLeagueRepository(memory.SeedLeagues())
	playerRepo := memory.NewPlayerRepository(memory.SeedPlayers())
	squadRepo := memory.NewSquadRepository()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := NewSquadService(
		leagueRepo,
		playerRepo,
		squadRepo,
		fantasy.DefaultRules(),
		staticIDGenerator{id: "squad-001"},
		logger,
	)

	firstNow := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return firstNow }

	playerIDs := []string{
		"idn-gk-02",
		"idn-def-04",
		"idn-def-05",
		"idn-def-01",
		"idn-mid-06",
		"idn-mid-05",
		"idn-mid-03",
		"idn-mid-04",
		"idn-fwd-03",
		"idn-fwd-01",
		"idn-fwd-02",
	}

	created, err := service.UpsertSquad(t.Context(), UpsertSquadInput{
		UserID:    "user-1",
		LeagueID:  memory.LeagueIDLiga1Indonesia,
		Name:      "Garuda FC",
		PlayerIDs: playerIDs,
	})
	if err != nil {
		t.Fatalf("upsert squad create failed: %v", err)
	}

	if created.ID != "squad-001" {
		t.Fatalf("expected squad id squad-001, got %s", created.ID)
	}
	if !created.CreatedAt.Equal(firstNow) || !created.UpdatedAt.Equal(firstNow) {
		t.Fatalf("expected created/updated at %v, got created=%v updated=%v", firstNow, created.CreatedAt, created.UpdatedAt)
	}

	secondNow := firstNow.Add(5 * time.Minute)
	service.now = func() time.Time { return secondNow }

	updated, err := service.UpsertSquad(t.Context(), UpsertSquadInput{
		UserID:    "user-1",
		LeagueID:  memory.LeagueIDLiga1Indonesia,
		Name:      "Garuda FC Reborn",
		PlayerIDs: playerIDs,
	})
	if err != nil {
		t.Fatalf("upsert squad update failed: %v", err)
	}

	if updated.ID != created.ID {
		t.Fatalf("expected same squad id on update, got %s vs %s", updated.ID, created.ID)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("expected created_at unchanged, got %v vs %v", updated.CreatedAt, created.CreatedAt)
	}
	if !updated.UpdatedAt.Equal(secondNow) {
		t.Fatalf("expected updated_at %v, got %v", secondNow, updated.UpdatedAt)
	}
}

func TestSquadService_UpsertSquad_InvalidInput(t *testing.T) {
	leagueRepo := memory.NewLeagueRepository(memory.SeedLeagues())
	playerRepo := memory.NewPlayerRepository(memory.SeedPlayers())
	squadRepo := memory.NewSquadRepository()

	service := NewSquadService(
		leagueRepo,
		playerRepo,
		squadRepo,
		fantasy.DefaultRules(),
		staticIDGenerator{id: "squad-001"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	_, err := service.UpsertSquad(t.Context(), UpsertSquadInput{
		UserID:    "user-1",
		LeagueID:  memory.LeagueIDLiga1Indonesia,
		Name:      "Bad Squad",
		PlayerIDs: []string{"idn-gk-02", "idn-gk-02"},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	_, err = service.UpsertSquad(t.Context(), UpsertSquadInput{
		UserID:    "user-1",
		LeagueID:  "unknown-league",
		Name:      "Bad League",
		PlayerIDs: []string{"idn-gk-02"},
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
