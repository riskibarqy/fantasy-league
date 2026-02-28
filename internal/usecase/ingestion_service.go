package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/leaguestanding"
	"github.com/riskibarqy/fantasy-league/internal/domain/playerstats"
	"github.com/riskibarqy/fantasy-league/internal/domain/rawdata"
	"github.com/riskibarqy/fantasy-league/internal/domain/teamstats"
)

type IngestionService struct {
	fixtureWriter   fixtureIngestionWriter
	standingRepo    leaguestanding.Repository
	playerStatsRepo playerstats.Repository
	teamStatsRepo   teamstats.Repository
	rawDataRepo     rawdata.Repository
}

type fixtureIngestionWriter interface {
	UpsertFixtures(ctx context.Context, fixtures []fixture.Fixture) error
}

func NewIngestionService(
	fixtureWriter fixtureIngestionWriter,
	standingRepo leaguestanding.Repository,
	playerStatsRepo playerstats.Repository,
	teamStatsRepo teamstats.Repository,
	rawDataRepo rawdata.Repository,
) *IngestionService {
	return &IngestionService{
		fixtureWriter:   fixtureWriter,
		standingRepo:    standingRepo,
		playerStatsRepo: playerStatsRepo,
		teamStatsRepo:   teamStatsRepo,
		rawDataRepo:     rawDataRepo,
	}
}

func (s *IngestionService) UpsertFixtures(ctx context.Context, fixtures []fixture.Fixture) error {
	ctx, span := startUsecaseSpan(ctx, "usecase.IngestionService.UpsertFixtures")
	defer span.End()

	if len(fixtures) == 0 {
		return fmt.Errorf("%w: fixtures are required", ErrInvalidInput)
	}
	for idx := range fixtures {
		fixtures[idx].ID = strings.TrimSpace(fixtures[idx].ID)
		fixtures[idx].LeagueID = strings.TrimSpace(fixtures[idx].LeagueID)
		fixtures[idx].HomeTeam = strings.TrimSpace(fixtures[idx].HomeTeam)
		fixtures[idx].AwayTeam = strings.TrimSpace(fixtures[idx].AwayTeam)
		fixtures[idx].Venue = strings.TrimSpace(fixtures[idx].Venue)
		fixtures[idx].Status = fixture.NormalizeStatus(fixtures[idx].Status)
		if fixtures[idx].ID == "" || fixtures[idx].LeagueID == "" {
			return fmt.Errorf("%w: fixture id and league id are required", ErrInvalidInput)
		}
		if fixtures[idx].Gameweek <= 0 {
			return fmt.Errorf("%w: fixture gameweek must be greater than zero", ErrInvalidInput)
		}
		if fixtures[idx].KickoffAt.IsZero() {
			return fmt.Errorf("%w: fixture kickoff_at is required", ErrInvalidInput)
		}
	}

	if err := s.fixtureWriter.UpsertFixtures(ctx, fixtures); err != nil {
		return fmt.Errorf("upsert fixtures: %w", err)
	}
	return nil
}

func (s *IngestionService) UpsertPlayerFixtureStats(ctx context.Context, fixtureID string, stats []playerstats.FixtureStat) error {
	ctx, span := startUsecaseSpan(ctx, "usecase.IngestionService.UpsertPlayerFixtureStats")
	defer span.End()

	fixtureID = strings.TrimSpace(fixtureID)
	if fixtureID == "" {
		return fmt.Errorf("%w: fixture_id is required", ErrInvalidInput)
	}

	cleaned := make([]playerstats.FixtureStat, 0, len(stats))
	for _, item := range stats {
		item.FixtureID = fixtureID
		item.PlayerID = strings.TrimSpace(item.PlayerID)
		item.TeamID = strings.TrimSpace(item.TeamID)
		if item.PlayerID == "" && item.PlayerExternalID <= 0 {
			return fmt.Errorf("%w: player identity is required", ErrInvalidInput)
		}
		if item.TeamID == "" && item.TeamExternalID <= 0 {
			return fmt.Errorf("%w: team identity is required", ErrInvalidInput)
		}
		cleaned = append(cleaned, item)
	}

	if err := s.playerStatsRepo.UpsertFixtureStats(ctx, fixtureID, cleaned); err != nil {
		return fmt.Errorf("upsert player fixture stats: %w", err)
	}
	return nil
}

func (s *IngestionService) UpsertTeamFixtureStats(ctx context.Context, fixtureID string, stats []teamstats.FixtureStat) error {
	ctx, span := startUsecaseSpan(ctx, "usecase.IngestionService.UpsertTeamFixtureStats")
	defer span.End()

	fixtureID = strings.TrimSpace(fixtureID)
	if fixtureID == "" {
		return fmt.Errorf("%w: fixture_id is required", ErrInvalidInput)
	}

	cleaned := make([]teamstats.FixtureStat, 0, len(stats))
	for _, item := range stats {
		item.FixtureID = fixtureID
		item.TeamID = strings.TrimSpace(item.TeamID)
		if item.TeamID == "" && item.TeamExternalID <= 0 {
			return fmt.Errorf("%w: team identity is required", ErrInvalidInput)
		}
		cleaned = append(cleaned, item)
	}

	if err := s.teamStatsRepo.UpsertFixtureStats(ctx, fixtureID, cleaned); err != nil {
		return fmt.Errorf("upsert team fixture stats: %w", err)
	}
	return nil
}

func (s *IngestionService) ReplaceFixtureEvents(ctx context.Context, fixtureID string, events []playerstats.FixtureEvent) error {
	ctx, span := startUsecaseSpan(ctx, "usecase.IngestionService.ReplaceFixtureEvents")
	defer span.End()

	fixtureID = strings.TrimSpace(fixtureID)
	if fixtureID == "" {
		return fmt.Errorf("%w: fixture_id is required", ErrInvalidInput)
	}

	cleaned := make([]playerstats.FixtureEvent, 0, len(events))
	for _, item := range events {
		item.FixtureID = fixtureID
		item.EventType = strings.TrimSpace(item.EventType)
		item.TeamID = strings.TrimSpace(item.TeamID)
		item.PlayerID = strings.TrimSpace(item.PlayerID)
		item.AssistPlayerID = strings.TrimSpace(item.AssistPlayerID)
		item.Detail = strings.TrimSpace(item.Detail)
		if item.EventType == "" {
			return fmt.Errorf("%w: event_type is required", ErrInvalidInput)
		}
		cleaned = append(cleaned, item)
	}

	if err := s.playerStatsRepo.ReplaceFixtureEvents(ctx, fixtureID, cleaned); err != nil {
		return fmt.Errorf("replace fixture events: %w", err)
	}
	return nil
}

func (s *IngestionService) UpsertRawPayloads(ctx context.Context, source string, items []rawdata.Payload) error {
	ctx, span := startUsecaseSpan(ctx, "usecase.IngestionService.UpsertRawPayloads")
	defer span.End()

	if s.rawDataRepo == nil {
		return nil
	}

	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		source = "sportmonks"
	}

	cleaned := make([]rawdata.Payload, 0, len(items))
	for _, item := range items {
		item.Source = source
		item.EntityType = strings.ToLower(strings.TrimSpace(item.EntityType))
		item.EntityKey = strings.TrimSpace(item.EntityKey)
		item.LeaguePublicID = strings.TrimSpace(item.LeaguePublicID)
		item.FixturePublicID = strings.TrimSpace(item.FixturePublicID)
		item.TeamPublicID = strings.TrimSpace(item.TeamPublicID)
		item.PlayerPublicID = strings.TrimSpace(item.PlayerPublicID)
		item.PayloadJSON = strings.TrimSpace(item.PayloadJSON)
		if item.EntityType == "" || item.EntityKey == "" || item.PayloadJSON == "" {
			return fmt.Errorf("%w: entity_type, entity_key and payload are required", ErrInvalidInput)
		}

		hash := sha256.Sum256([]byte(item.PayloadJSON))
		item.PayloadHash = hex.EncodeToString(hash[:])
		cleaned = append(cleaned, item)
	}

	if err := s.rawDataRepo.UpsertMany(ctx, cleaned); err != nil {
		return fmt.Errorf("upsert raw payloads: %w", err)
	}

	return nil
}

func (s *IngestionService) ReplaceLeagueStandings(ctx context.Context, leagueID string, live bool, items []leaguestanding.Standing) error {
	ctx, span := startUsecaseSpan(ctx, "usecase.IngestionService.ReplaceLeagueStandings")
	defer span.End()

	leagueID = strings.TrimSpace(leagueID)
	if leagueID == "" {
		return fmt.Errorf("%w: league_id is required", ErrInvalidInput)
	}
	if s.standingRepo == nil {
		return nil
	}

	cleaned := make([]leaguestanding.Standing, 0, len(items))
	for _, item := range items {
		item.LeagueID = leagueID
		item.TeamID = strings.TrimSpace(item.TeamID)
		item.Form = strings.TrimSpace(item.Form)
		item.IsLive = live

		if item.TeamID == "" {
			return fmt.Errorf("%w: team_id is required", ErrInvalidInput)
		}
		if item.Position <= 0 {
			return fmt.Errorf("%w: position must be greater than zero", ErrInvalidInput)
		}
		if item.Played < 0 || item.Won < 0 || item.Draw < 0 || item.Lost < 0 {
			return fmt.Errorf("%w: played/won/draw/lost cannot be negative", ErrInvalidInput)
		}
		if item.GoalsFor < 0 || item.GoalsAgainst < 0 {
			return fmt.Errorf("%w: goals_for/goals_against cannot be negative", ErrInvalidInput)
		}
		if item.Points < 0 {
			return fmt.Errorf("%w: points cannot be negative", ErrInvalidInput)
		}

		cleaned = append(cleaned, item)
	}

	if err := s.standingRepo.ReplaceByLeague(ctx, leagueID, live, cleaned); err != nil {
		return fmt.Errorf("replace league standings: %w", err)
	}
	return nil
}
