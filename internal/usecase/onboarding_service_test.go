package usecase

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/onboarding"
	"github.com/riskibarqy/fantasy-league/internal/infrastructure/repository/memory"
)

type inMemoryOnboardingProfileRepo struct {
	profiles map[string]onboarding.Profile
}

func newInMemoryOnboardingProfileRepo() *inMemoryOnboardingProfileRepo {
	return &inMemoryOnboardingProfileRepo{
		profiles: make(map[string]onboarding.Profile),
	}
}

func (r *inMemoryOnboardingProfileRepo) GetByUserID(_ context.Context, userID string) (onboarding.Profile, bool, error) {
	item, ok := r.profiles[userID]
	return item, ok, nil
}

func (r *inMemoryOnboardingProfileRepo) Upsert(_ context.Context, profile onboarding.Profile) error {
	r.profiles[profile.UserID] = profile
	return nil
}

type recordingOnboardingJoiner struct {
	userID      string
	leagueID    string
	squadID     string
	countryCode string
	called      bool
}

func (j *recordingOnboardingJoiner) EnsureDefaultMemberships(_ context.Context, userID, leagueID, squadID, countryCode string) error {
	j.called = true
	j.userID = userID
	j.leagueID = leagueID
	j.squadID = squadID
	j.countryCode = countryCode
	return nil
}

func TestOnboardingService_SaveFavoriteClub(t *testing.T) {
	teamRepo := memory.NewTeamRepository(memory.SeedTeams())
	profileRepo := newInMemoryOnboardingProfileRepo()
	service := NewOnboardingService(teamRepo, profileRepo, nil, nil, nil)

	profile, err := service.SaveFavoriteClub(t.Context(), SaveFavoriteClubInput{
		UserID:      "user-1",
		LeagueID:    memory.LeagueIDLiga1Indonesia,
		TeamID:      "idn-persib",
		CountryCode: "id",
		IPAddress:   "1.2.3.4",
	})
	if err != nil {
		t.Fatalf("save favorite club failed: %v", err)
	}

	if profile.FavoriteLeagueID != memory.LeagueIDLiga1Indonesia {
		t.Fatalf("unexpected favorite league id: %s", profile.FavoriteLeagueID)
	}
	if profile.FavoriteTeamID != "idn-persib" {
		t.Fatalf("unexpected favorite team id: %s", profile.FavoriteTeamID)
	}
	if profile.CountryCode != "ID" {
		t.Fatalf("expected normalized country code ID, got %s", profile.CountryCode)
	}
	if profile.IPAddress != "1.2.3.4" {
		t.Fatalf("unexpected ip address: %s", profile.IPAddress)
	}
	if profile.OnboardingCompleted {
		t.Fatalf("expected onboarding_completed=false after favorite club step")
	}
}

func TestOnboardingService_Complete(t *testing.T) {
	leagueRepo := memory.NewLeagueRepository(memory.SeedLeagues())
	teamRepo := memory.NewTeamRepository(memory.SeedTeams())
	playerRepo := memory.NewPlayerRepository(memory.SeedPlayers())
	squadRepo := memory.NewSquadRepository()
	lineupRepo := memory.NewLineupRepository()
	profileRepo := newInMemoryOnboardingProfileRepo()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	squadSvc := NewSquadService(
		leagueRepo,
		playerRepo,
		squadRepo,
		fantasy.DefaultRules(),
		staticIDGenerator{id: "squad-123"},
		logger,
	)
	lineupSvc := NewLineupService(leagueRepo, playerRepo, lineupRepo, squadRepo)

	service := NewOnboardingService(teamRepo, profileRepo, squadSvc, lineupSvc, nil)
	joiner := &recordingOnboardingJoiner{}
	service.customLeagueJoiner = joiner

	profile, squad, savedLineup, err := service.Complete(t.Context(), CompleteOnboardingInput{
		UserID:        "user-1",
		LeagueID:      memory.LeagueIDLiga1Indonesia,
		SquadName:     "Garuda XI",
		PlayerIDs:     validOnboardingPlayerIDs(),
		GoalkeeperID:  "idn-gk-01",
		DefenderIDs:   []string{"idn-def-01", "idn-def-02", "idn-def-03", "idn-def-04"},
		MidfielderIDs: []string{"idn-mid-01", "idn-mid-03", "idn-mid-04", "idn-mid-05"},
		ForwardIDs:    []string{"idn-fwd-02", "idn-fwd-03"},
		SubstituteIDs: []string{"idn-gk-02", "idn-def-06", "idn-mid-07", "idn-fwd-04"},
		CaptainID:     "idn-mid-01",
		ViceCaptainID: "idn-def-01",
		CountryCode:   "id",
		IPAddress:     "2.2.2.2",
	})
	if err != nil {
		t.Fatalf("complete onboarding failed: %v", err)
	}

	if !profile.OnboardingCompleted {
		t.Fatalf("expected onboarding_completed=true")
	}
	if profile.CountryCode != "ID" {
		t.Fatalf("expected normalized country code ID, got %s", profile.CountryCode)
	}
	if len(squad.Picks) != 15 {
		t.Fatalf("expected 15 squad picks, got %d", len(squad.Picks))
	}
	total := sumSquadCost(squad.Picks)
	if total > squad.BudgetCap {
		t.Fatalf("expected total squad cost <= budget cap, got total=%d budget=%d", total, squad.BudgetCap)
	}
	if savedLineup.CaptainID != "idn-mid-01" || savedLineup.ViceCaptainID != "idn-def-01" {
		t.Fatalf("unexpected captain/vice saved: captain=%s vice=%s", savedLineup.CaptainID, savedLineup.ViceCaptainID)
	}
	if !joiner.called {
		t.Fatalf("expected default custom league joiner to be called")
	}
	if joiner.countryCode != "ID" {
		t.Fatalf("expected joiner country code ID, got %s", joiner.countryCode)
	}
}

func TestOnboardingService_Complete_DoesNotOverrideCountryWithZZ(t *testing.T) {
	leagueRepo := memory.NewLeagueRepository(memory.SeedLeagues())
	teamRepo := memory.NewTeamRepository(memory.SeedTeams())
	playerRepo := memory.NewPlayerRepository(memory.SeedPlayers())
	squadRepo := memory.NewSquadRepository()
	lineupRepo := memory.NewLineupRepository()
	profileRepo := newInMemoryOnboardingProfileRepo()
	profileRepo.profiles["user-1"] = onboarding.Profile{
		UserID:      "user-1",
		CountryCode: "ID",
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	squadSvc := NewSquadService(
		leagueRepo,
		playerRepo,
		squadRepo,
		fantasy.DefaultRules(),
		staticIDGenerator{id: "squad-123"},
		logger,
	)
	lineupSvc := NewLineupService(leagueRepo, playerRepo, lineupRepo, squadRepo)

	service := NewOnboardingService(teamRepo, profileRepo, squadSvc, lineupSvc, nil)
	joiner := &recordingOnboardingJoiner{}
	service.customLeagueJoiner = joiner

	profile, _, _, err := service.Complete(t.Context(), CompleteOnboardingInput{
		UserID:        "user-1",
		LeagueID:      memory.LeagueIDLiga1Indonesia,
		SquadName:     "Garuda XI",
		PlayerIDs:     validOnboardingPlayerIDs(),
		GoalkeeperID:  "idn-gk-01",
		DefenderIDs:   []string{"idn-def-01", "idn-def-02", "idn-def-03", "idn-def-04"},
		MidfielderIDs: []string{"idn-mid-01", "idn-mid-03", "idn-mid-04", "idn-mid-05"},
		ForwardIDs:    []string{"idn-fwd-02", "idn-fwd-03"},
		SubstituteIDs: []string{"idn-gk-02", "idn-def-06", "idn-mid-07", "idn-fwd-04"},
		CaptainID:     "idn-mid-01",
		ViceCaptainID: "idn-def-01",
		CountryCode:   "ZZ",
		IPAddress:     "2.2.2.2",
	})
	if err != nil {
		t.Fatalf("complete onboarding failed: %v", err)
	}

	if profile.CountryCode != "ID" {
		t.Fatalf("expected existing country code ID to be preserved, got %s", profile.CountryCode)
	}
	if !joiner.called {
		t.Fatalf("expected default custom league joiner to be called")
	}
	if joiner.countryCode != "ID" {
		t.Fatalf("expected joiner country code ID, got %s", joiner.countryCode)
	}
}

func validOnboardingPlayerIDs() []string {
	return []string{
		"idn-gk-01",
		"idn-gk-02",
		"idn-def-01",
		"idn-def-02",
		"idn-def-03",
		"idn-def-04",
		"idn-def-06",
		"idn-mid-01",
		"idn-mid-03",
		"idn-mid-04",
		"idn-mid-05",
		"idn-mid-07",
		"idn-fwd-02",
		"idn-fwd-03",
		"idn-fwd-04",
	}
}

func sumSquadCost(picks []fantasy.SquadPick) int64 {
	var total int64
	for _, pick := range picks {
		total += pick.Price
	}
	return total
}
