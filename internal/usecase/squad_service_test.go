package usecase

import (
	"context"
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

type recordingDefaultLeagueJoiner struct {
	called   bool
	userID   string
	leagueID string
	squadID  string
	err      error
}

func (j *recordingDefaultLeagueJoiner) EnsureDefaultMemberships(_ context.Context, userID, leagueID, squadID string) error {
	j.called = true
	j.userID = userID
	j.leagueID = leagueID
	j.squadID = squadID
	return j.err
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
		"idn-gk-01",
		"idn-gk-02",
		"idn-gk-03",
		"idn-def-01",
		"idn-def-02",
		"idn-def-03",
		"idn-def-04",
		"idn-def-06",
		"idn-mid-01",
		"idn-mid-03",
		"idn-mid-04",
		"idn-mid-05",
		"idn-mid-06",
		"idn-fwd-03",
		"idn-fwd-04",
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

func TestSquadService_PickSquad_DefaultAndReuseName(t *testing.T) {
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

	playerIDs := []string{
		"idn-gk-01",
		"idn-gk-02",
		"idn-gk-03",
		"idn-def-01",
		"idn-def-02",
		"idn-def-03",
		"idn-def-04",
		"idn-def-06",
		"idn-mid-01",
		"idn-mid-03",
		"idn-mid-04",
		"idn-mid-05",
		"idn-mid-06",
		"idn-fwd-03",
		"idn-fwd-04",
	}

	created, err := service.PickSquad(t.Context(), PickSquadInput{
		UserID:    "user-1",
		LeagueID:  memory.LeagueIDLiga1Indonesia,
		PlayerIDs: playerIDs,
	})
	if err != nil {
		t.Fatalf("pick squad create failed: %v", err)
	}
	if created.Name != defaultSquadName {
		t.Fatalf("expected default squad name %q, got %q", defaultSquadName, created.Name)
	}

	updated, err := service.PickSquad(t.Context(), PickSquadInput{
		UserID:    "user-1",
		LeagueID:  memory.LeagueIDLiga1Indonesia,
		PlayerIDs: playerIDs,
	})
	if err != nil {
		t.Fatalf("pick squad update failed: %v", err)
	}
	if updated.Name != defaultSquadName {
		t.Fatalf("expected existing squad name %q to be reused, got %q", defaultSquadName, updated.Name)
	}
}

func TestSquadService_AddPlayerToSquad(t *testing.T) {
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

	squad, err := service.AddPlayerToSquad(t.Context(), AddPlayerToSquadInput{
		UserID:   "user-1",
		LeagueID: memory.LeagueIDLiga1Indonesia,
		PlayerID: "idn-gk-02",
	})
	if err != nil {
		t.Fatalf("add first player failed: %v", err)
	}
	if len(squad.Picks) != 1 {
		t.Fatalf("expected 1 pick, got %d", len(squad.Picks))
	}
	if squad.Name != defaultSquadName {
		t.Fatalf("expected default squad name %q, got %q", defaultSquadName, squad.Name)
	}

	_, err = service.AddPlayerToSquad(t.Context(), AddPlayerToSquadInput{
		UserID:   "user-1",
		LeagueID: memory.LeagueIDLiga1Indonesia,
		PlayerID: "idn-gk-02",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for duplicate add, got %v", err)
	}

	for _, playerID := range []string{"idn-def-02", "idn-mid-06"} {
		_, err = service.AddPlayerToSquad(t.Context(), AddPlayerToSquadInput{
			UserID:   "user-1",
			LeagueID: memory.LeagueIDLiga1Indonesia,
			PlayerID: playerID,
		})
		if err != nil {
			t.Fatalf("unexpected add player failure for %s: %v", playerID, err)
		}
	}

	_, err = service.AddPlayerToSquad(t.Context(), AddPlayerToSquadInput{
		UserID:   "user-1",
		LeagueID: memory.LeagueIDLiga1Indonesia,
		PlayerID: "idn-fwd-02",
	})
	if !errors.Is(err, fantasy.ErrExceededTeamLimit) {
		t.Fatalf("expected ErrExceededTeamLimit, got %v", err)
	}
}

func TestSquadService_UpsertSquad_AutoJoinDefaultLeagues(t *testing.T) {
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

	joiner := &recordingDefaultLeagueJoiner{}
	service.SetDefaultLeagueJoiner(joiner)

	playerIDs := []string{
		"idn-gk-01",
		"idn-gk-02",
		"idn-gk-03",
		"idn-def-01",
		"idn-def-02",
		"idn-def-03",
		"idn-def-04",
		"idn-def-06",
		"idn-mid-01",
		"idn-mid-03",
		"idn-mid-04",
		"idn-mid-05",
		"idn-mid-06",
		"idn-fwd-03",
		"idn-fwd-04",
	}

	squad, err := service.UpsertSquad(t.Context(), UpsertSquadInput{
		UserID:    "user-1",
		LeagueID:  memory.LeagueIDLiga1Indonesia,
		Name:      "Garuda FC",
		PlayerIDs: playerIDs,
	})
	if err != nil {
		t.Fatalf("upsert squad with auto join failed: %v", err)
	}

	if !joiner.called {
		t.Fatalf("expected default league joiner to be called")
	}
	if joiner.userID != "user-1" {
		t.Fatalf("unexpected joiner user_id: %s", joiner.userID)
	}
	if joiner.leagueID != memory.LeagueIDLiga1Indonesia {
		t.Fatalf("unexpected joiner league_id: %s", joiner.leagueID)
	}
	if joiner.squadID != squad.ID {
		t.Fatalf("unexpected joiner squad_id: %s", joiner.squadID)
	}
}
