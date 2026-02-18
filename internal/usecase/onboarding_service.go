package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	"github.com/riskibarqy/fantasy-league/internal/domain/onboarding"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
)

type SaveFavoriteClubInput struct {
	UserID      string
	LeagueID    string
	TeamID      string
	CountryCode string
	IPAddress   string
}

type CompleteOnboardingInput struct {
	UserID        string
	LeagueID      string
	SquadName     string
	PlayerIDs     []string
	GoalkeeperID  string
	DefenderIDs   []string
	MidfielderIDs []string
	ForwardIDs    []string
	SubstituteIDs []string
	CaptainID     string
	ViceCaptainID string
	CountryCode   string
	IPAddress     string
}

type OnboardingService struct {
	teamRepo           team.Repository
	profileRepo        onboarding.Repository
	squadService       *SquadService
	lineupService      *LineupService
	customLeagueJoiner onboardingLeagueJoiner
	now                func() time.Time
}

type onboardingLeagueJoiner interface {
	EnsureDefaultMemberships(ctx context.Context, userID, leagueID, squadID, countryCode string) error
}

type noopOnboardingLeagueJoiner struct{}

func (noopOnboardingLeagueJoiner) EnsureDefaultMemberships(_ context.Context, _, _, _, _ string) error {
	return nil
}

func NewOnboardingService(
	teamRepo team.Repository,
	profileRepo onboarding.Repository,
	squadService *SquadService,
	lineupService *LineupService,
	customLeagueService *CustomLeagueService,
) *OnboardingService {
	joiner := onboardingLeagueJoiner(noopOnboardingLeagueJoiner{})
	if customLeagueService != nil {
		joiner = customLeagueService
	}

	return &OnboardingService{
		teamRepo:           teamRepo,
		profileRepo:        profileRepo,
		squadService:       squadService,
		lineupService:      lineupService,
		customLeagueJoiner: joiner,
		now:                time.Now,
	}
}

func (s *OnboardingService) SaveFavoriteClub(ctx context.Context, input SaveFavoriteClubInput) (onboarding.Profile, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.LeagueID = strings.TrimSpace(input.LeagueID)
	input.TeamID = strings.TrimSpace(input.TeamID)

	if input.UserID == "" {
		return onboarding.Profile{}, fmt.Errorf("%w: user_id is required", ErrInvalidInput)
	}
	if input.LeagueID == "" {
		return onboarding.Profile{}, fmt.Errorf("%w: league_id is required", ErrInvalidInput)
	}
	if input.TeamID == "" {
		return onboarding.Profile{}, fmt.Errorf("%w: team_id is required", ErrInvalidInput)
	}

	if _, exists, err := s.teamRepo.GetByID(ctx, input.LeagueID, input.TeamID); err != nil {
		return onboarding.Profile{}, fmt.Errorf("get team by id: %w", err)
	} else if !exists {
		return onboarding.Profile{}, fmt.Errorf("%w: favorite team not found in selected league", ErrNotFound)
	}

	profile, err := s.mergeProfile(ctx, profileWriteInput{
		UserID:           input.UserID,
		FavoriteLeagueID: input.LeagueID,
		FavoriteTeamID:   input.TeamID,
		CountryCode:      input.CountryCode,
		IPAddress:        input.IPAddress,
	})
	if err != nil {
		return onboarding.Profile{}, err
	}

	return profile, nil
}

func (s *OnboardingService) Complete(ctx context.Context, input CompleteOnboardingInput) (onboarding.Profile, fantasy.Squad, lineup.Lineup, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.LeagueID = strings.TrimSpace(input.LeagueID)
	input.SquadName = strings.TrimSpace(input.SquadName)
	input.GoalkeeperID = strings.TrimSpace(input.GoalkeeperID)
	input.CaptainID = strings.TrimSpace(input.CaptainID)
	input.ViceCaptainID = strings.TrimSpace(input.ViceCaptainID)

	if input.UserID == "" {
		return onboarding.Profile{}, fantasy.Squad{}, lineup.Lineup{}, fmt.Errorf("%w: user_id is required", ErrInvalidInput)
	}
	if input.LeagueID == "" {
		return onboarding.Profile{}, fantasy.Squad{}, lineup.Lineup{}, fmt.Errorf("%w: league_id is required", ErrInvalidInput)
	}

	squad, err := s.squadService.PickSquad(ctx, PickSquadInput{
		UserID:    input.UserID,
		LeagueID:  input.LeagueID,
		SquadName: input.SquadName,
		PlayerIDs: input.PlayerIDs,
	})
	if err != nil {
		return onboarding.Profile{}, fantasy.Squad{}, lineup.Lineup{}, fmt.Errorf("pick squad: %w", err)
	}

	savedLineup, err := s.lineupService.Save(ctx, SaveLineupInput{
		UserID:        input.UserID,
		LeagueID:      input.LeagueID,
		GoalkeeperID:  input.GoalkeeperID,
		DefenderIDs:   input.DefenderIDs,
		MidfielderIDs: input.MidfielderIDs,
		ForwardIDs:    input.ForwardIDs,
		SubstituteIDs: input.SubstituteIDs,
		CaptainID:     input.CaptainID,
		ViceCaptainID: input.ViceCaptainID,
	})
	if err != nil {
		return onboarding.Profile{}, fantasy.Squad{}, lineup.Lineup{}, fmt.Errorf("save lineup: %w", err)
	}

	profile, err := s.mergeProfile(ctx, profileWriteInput{
		UserID:              input.UserID,
		FavoriteLeagueID:    input.LeagueID,
		CountryCode:         input.CountryCode,
		IPAddress:           input.IPAddress,
		OnboardingCompleted: true,
	})
	if err != nil {
		return onboarding.Profile{}, fantasy.Squad{}, lineup.Lineup{}, err
	}

	if err := s.customLeagueJoiner.EnsureDefaultMemberships(ctx, input.UserID, input.LeagueID, squad.ID, profile.CountryCode); err != nil {
		return onboarding.Profile{}, fantasy.Squad{}, lineup.Lineup{}, fmt.Errorf("auto join default custom leagues: %w", err)
	}

	return profile, squad, savedLineup, nil
}

type profileWriteInput struct {
	UserID              string
	FavoriteLeagueID    string
	FavoriteTeamID      string
	CountryCode         string
	IPAddress           string
	OnboardingCompleted bool
}

func (s *OnboardingService) mergeProfile(ctx context.Context, input profileWriteInput) (onboarding.Profile, error) {
	existing, exists, err := s.profileRepo.GetByUserID(ctx, input.UserID)
	if err != nil {
		return onboarding.Profile{}, fmt.Errorf("get onboarding profile: %w", err)
	}

	now := s.now().UTC()
	out := existing
	out.UserID = input.UserID

	if strings.TrimSpace(input.FavoriteLeagueID) != "" {
		out.FavoriteLeagueID = strings.TrimSpace(input.FavoriteLeagueID)
	}
	if strings.TrimSpace(input.FavoriteTeamID) != "" {
		out.FavoriteTeamID = strings.TrimSpace(input.FavoriteTeamID)
	}

	if country := normalizeCountryCodeForOnboarding(ctx, input.CountryCode); country != "" {
		out.CountryCode = country
	}
	if ip := normalizeIPForOnboarding(ctx, input.IPAddress); ip != "" {
		out.IPAddress = ip
	}

	if input.OnboardingCompleted {
		out.OnboardingCompleted = true
	}
	if !exists {
		out.CreatedAt = now
	}
	out.UpdatedAt = now

	if err := s.profileRepo.Upsert(ctx, out); err != nil {
		return onboarding.Profile{}, fmt.Errorf("upsert onboarding profile: %w", err)
	}

	latest, latestExists, err := s.profileRepo.GetByUserID(ctx, input.UserID)
	if err != nil {
		return onboarding.Profile{}, fmt.Errorf("re-fetch onboarding profile: %w", err)
	}
	if latestExists {
		return latest, nil
	}
	return out, nil
}

func normalizeCountryCodeForOnboarding(ctx context.Context, value string) string {
	_ = ctx
	code := strings.ToUpper(strings.TrimSpace(value))
	if len(code) != 2 {
		return ""
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return code
}

func normalizeIPForOnboarding(ctx context.Context, value string) string {
	_ = ctx
	return strings.TrimSpace(value)
}
