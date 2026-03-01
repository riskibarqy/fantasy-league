package usecase

import (
	"context"
	"fmt"
	"github.com/riskibarqy/fantasy-league/internal/domain/topscorers"
	"github.com/riskibarqy/fantasy-league/internal/platform/logging"
	"sort"
	"strings"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/leaguestanding"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	"github.com/riskibarqy/fantasy-league/internal/domain/playerstats"
	"github.com/riskibarqy/fantasy-league/internal/domain/rawdata"
	"github.com/riskibarqy/fantasy-league/internal/domain/statvalue"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
	"github.com/riskibarqy/fantasy-league/internal/domain/teamstats"
)

type SportDataSyncProvider interface {
	FetchFixtureBundleBySeason(ctx context.Context, seasonID int64) (ExternalFixtureBundle, error)
	FetchFixturesBySeason(ctx context.Context, seasonID int64) ([]ExternalFixture, []rawdata.Payload, error)
	FetchStandingsBySeason(ctx context.Context, seasonID int64) ([]ExternalStanding, []rawdata.Payload, error)
	FetchLiveStandingsByLeague(ctx context.Context, leagueRefID int64) ([]ExternalStanding, []rawdata.Payload, error)
	FetchStatisticTypes(ctx context.Context) ([]ExternalStatType, []rawdata.Payload, error)
	FetchTeamStatisticsBySeason(ctx context.Context, seasonID int64) ([]ExternalTeamStatValue, []rawdata.Payload, error)
	FetchPlayerStatisticsBySeason(ctx context.Context, seasonID int64) ([]ExternalPlayerStatValue, []rawdata.Payload, error)
	FetchTopScorersBySeasonID(ctx context.Context, seasonID, page, typeId int) ([]ExternalTopScorers, bool, error)
}

type ExternalFixtureBundle struct {
	Fixtures    []ExternalFixture
	Teams       []ExternalTeam
	Players     []ExternalPlayer
	TeamStats   []ExternalTeamFixtureStat
	PlayerStats []ExternalPlayerFixtureStat
	Events      []ExternalFixtureEvent
	RawPayloads []rawdata.Payload
}
type ExternalTopScorers struct {
	TypeID           int64
	TypeName         string
	Rank             int
	Total            int
	LeagueID         int64
	PlayerID         int64
	Season           string
	ParticipantID    int64
	PlayerName       string
	ImagePlayer      string
	Nationality      string
	ImageNationality string
	ParticipantName  string
	ImageParticipant string
	PositionName     string //attacker
}
type ExternalFixture struct {
	ExternalID           int64
	Gameweek             int
	HomeTeamName         string
	AwayTeamName         string
	HomeTeamExternalID   int64
	AwayTeamExternalID   int64
	KickoffAt            time.Time
	Venue                string
	Status               string
	HomeScore            *int
	AwayScore            *int
	WinnerTeamExternalID int64
	FinishedAt           *time.Time
}

type ExternalStanding struct {
	TeamExternalID  int64
	TeamName        string
	Position        int
	Played          int
	Won             int
	Draw            int
	Lost            int
	GoalsFor        int
	GoalsAgainst    int
	GoalDifference  int
	Points          int
	Form            string
	SourceUpdatedAt *time.Time
}

type ExternalTeam struct {
	ExternalID int64
	Name       string
	Short      string
	ImageURL   string
}

type ExternalPlayer struct {
	ExternalID     int64
	TeamExternalID int64
	Name           string
	Position       string
	ImageURL       string
	Price          int64
}

type ExternalTeamFixtureStat struct {
	FixtureExternalID int64
	TeamExternalID    int64
	TeamName          string
	PossessionPct     float64
	Shots             int
	ShotsOnTarget     int
	Corners           int
	Fouls             int
	Offsides          int
	AdvancedStats     map[string]any
}

type ExternalPlayerFixtureStat struct {
	FixtureExternalID int64
	PlayerExternalID  int64
	TeamExternalID    int64
	Position          string
	MinutesPlayed     int
	Goals             int
	Assists           int
	CleanSheet        bool
	GoalsConceded     int
	OwnGoals          int
	PenaltiesSaved    int
	PenaltiesMissed   int
	YellowCards       int
	RedCards          int
	Saves             int
	BPS               int
	BonusPoints       int
	FantasyPoints     int
	AdvancedStats     map[string]any
}

type ExternalFixtureEvent struct {
	EventExternalID        int64
	FixtureExternalID      int64
	TeamExternalID         int64
	PlayerExternalID       int64
	AssistPlayerExternalID int64
	EventType              string
	Detail                 string
	Minute                 int
	ExtraMinute            int
	Metadata               map[string]any
}

type ExternalStatType struct {
	ExternalTypeID int64
	Name           string
	DeveloperName  string
	Code           string
	ModelType      string
	StatGroup      string
	Metadata       map[string]any
}

type ExternalTeamStatValue struct {
	SeasonRefID        int64
	TeamExternalID     int64
	FixtureExternalID  int64
	StatTypeExternalID int64
	StatTypeName       string
	StatKey            string
	Scope              string
	ValueNum           *float64
	ValueText          string
	ValueJSON          map[string]any
	SourceUpdatedAt    *time.Time
	Metadata           map[string]any
}

type ExternalPlayerStatValue struct {
	SeasonRefID        int64
	PlayerExternalID   int64
	TeamExternalID     int64
	FixtureExternalID  int64
	StatTypeExternalID int64
	StatTypeName       string
	StatKey            string
	Scope              string
	ValueNum           *float64
	ValueText          string
	ValueJSON          map[string]any
	SourceUpdatedAt    *time.Time
	Metadata           map[string]any
}

type SportDataSyncConfig struct {
	Enabled          bool
	SeasonIDByLeague map[string]int64
	LeagueIDByLeague map[string]int64
}

type SportDataSyncService struct {
	topScore   topscorers.Repository
	provider   SportDataSyncProvider
	teamRepo   team.Repository
	playerRepo player.Repository
	statRepo   statvalue.Repository
	ingestion  *IngestionService
	cfg        SportDataSyncConfig
	logger     *logging.Logger
}

func NewSportDataSyncService(
	provider SportDataSyncProvider,
	teamRepo team.Repository,
	playerRepo player.Repository,
	topScoreRepo topscorers.Repository,
	ingestion *IngestionService,
	cfg SportDataSyncConfig,
	logger *logging.Logger,
) *SportDataSyncService {
	if logger == nil {
		logger = logging.Default()
	}

	return &SportDataSyncService{
		provider:   provider,
		teamRepo:   teamRepo,
		playerRepo: playerRepo,
		topScore:   topScoreRepo,
		ingestion:  ingestion,
		cfg:        cfg,
		logger:     logger,
	}
}

func (s *SportDataSyncService) SetStatValueRepository(repo statvalue.Repository) {
	s.statRepo = repo
}

func (s *SportDataSyncService) SyncSchedule(ctx context.Context, lg league.League) error {
	ctx, span := startUsecaseSpan(ctx, "usecase.SportDataSyncService.SyncSchedule")
	defer span.End()

	if !s.cfg.Enabled {
		s.logger.WarnContext(ctx, "skip schedule sync: sport data sync is disabled", "league_id", lg.ID)
		return fmt.Errorf("%w: sport data sync is disabled (SPORTMONKS_ENABLED=false)", ErrDependencyUnavailable)
	}
	if s.provider == nil || s.ingestion == nil || s.teamRepo == nil || s.playerRepo == nil {
		s.logger.WarnContext(ctx,
			"skip schedule sync: sport data provider is not fully configured",
			"league_id", lg.ID,
			"provider_nil", s.provider == nil,
			"ingestion_nil", s.ingestion == nil,
			"team_repo_nil", s.teamRepo == nil,
			"player_repo_nil", s.playerRepo == nil,
		)
		return fmt.Errorf("%w: sport data provider is not fully configured", ErrDependencyUnavailable)
	}

	seasonID, ok := s.cfg.SeasonIDByLeague[strings.TrimSpace(lg.ID)]
	if !ok || seasonID <= 0 {
		return fmt.Errorf("%w: SPORTMONKS_SEASON_ID_MAP missing league=%s", ErrDependencyUnavailable, lg.ID)
	}

	teamMappings, err := s.loadTeamMappings(ctx, lg.ID)
	if err != nil {
		return err
	}

	if err := s.syncFixtureBundle(ctx, lg.ID, seasonID, teamMappings, fixtureBundleSyncOptions{
		onlyLiveWindow:   true,
		upsertRawPayload: false,
	}); err != nil {
		return err
	}

	standings, standingsPayloads, err := s.provider.FetchStandingsBySeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("fetch standings from sport data provider season_id=%d league=%s: %w", seasonID, lg.ID, err)
	}
	mappedStandings := mapExternalStandingsToDomain(lg.ID, standings, teamMappings)
	standingsGameweek := resolveStandingsSnapshotGameweek(standings)
	if len(standings) > 0 && len(mappedStandings) == 0 {
		s.logger.WarnContext(ctx,
			"season standings fetched but no rows mapped to internal teams",
			"league_id", lg.ID,
			"provider_count", len(standings),
		)
	}
	if len(mappedStandings) > 0 {
		if err := s.ingestion.ReplaceLeagueStandings(ctx, lg.ID, false, standingsGameweek, mappedStandings); err != nil {
			return fmt.Errorf("replace season standings league=%s gameweek=%d: %w", lg.ID, standingsGameweek, err)
		}
	}
	if len(standingsPayloads) > 0 {
		payloads := applyLeagueToPayloads(lg.ID, standingsPayloads)
		if err := s.ingestion.UpsertRawPayloads(ctx, "sportmonks", payloads); err != nil {
			return fmt.Errorf("upsert season standings raw payloads league=%s: %w", lg.ID, err)
		}
	}

	if err := s.SyncTopScorers(ctx, lg); err != nil {
		s.logger.WarnContext(ctx,
			"sync top scorers failed during schedule sync; continue without blocking schedule flow",
			"league_id", lg.ID,
			"error", err,
		)
	}

	return nil
}

func (s *SportDataSyncService) SyncLive(ctx context.Context, lg league.League) error {
	ctx, span := startUsecaseSpan(ctx, "usecase.SportDataSyncService.SyncLive")
	defer span.End()

	if !s.cfg.Enabled {
		s.logger.WarnContext(ctx, "skip live sync: sport data sync is disabled", "league_id", lg.ID)
		return fmt.Errorf("%w: sport data sync is disabled (SPORTMONKS_ENABLED=false)", ErrDependencyUnavailable)
	}
	if s.provider == nil || s.ingestion == nil || s.teamRepo == nil || s.playerRepo == nil {
		s.logger.WarnContext(ctx,
			"skip live sync: sport data provider is not fully configured",
			"league_id", lg.ID,
			"provider_nil", s.provider == nil,
			"ingestion_nil", s.ingestion == nil,
			"team_repo_nil", s.teamRepo == nil,
			"player_repo_nil", s.playerRepo == nil,
		)
		return fmt.Errorf("%w: sport data provider is not fully configured", ErrDependencyUnavailable)
	}

	teamMappings, err := s.loadTeamMappings(ctx, lg.ID)
	if err != nil {
		return err
	}

	leagueRefID := lg.LeagueRefID
	if leagueRefID <= 0 {
		leagueRefID = s.cfg.LeagueIDByLeague[strings.TrimSpace(lg.ID)]
	}
	if leagueRefID <= 0 {
		return fmt.Errorf("%w: league reference id is missing for league=%s", ErrDependencyUnavailable, lg.ID)
	}

	liveStandings, livePayloads, err := s.provider.FetchLiveStandingsByLeague(ctx, leagueRefID)
	if err != nil {
		return fmt.Errorf("fetch live standings from sport data provider league_ref_id=%d league=%s: %w", leagueRefID, lg.ID, err)
	}
	mappedLiveStandings := mapExternalStandingsToDomain(lg.ID, liveStandings, teamMappings)
	liveStandingsGameweek := resolveStandingsSnapshotGameweek(liveStandings)
	if len(liveStandings) > 0 && len(mappedLiveStandings) == 0 {
		s.logger.WarnContext(ctx,
			"live standings fetched but no rows mapped to internal teams",
			"league_id", lg.ID,
			"league_ref_id", leagueRefID,
			"provider_count", len(liveStandings),
		)
	}
	if len(mappedLiveStandings) > 0 {
		if err := s.ingestion.ReplaceLeagueStandings(ctx, lg.ID, true, liveStandingsGameweek, mappedLiveStandings); err != nil {
			return fmt.Errorf("replace live standings league=%s gameweek=%d: %w", lg.ID, liveStandingsGameweek, err)
		}
	}
	if len(livePayloads) > 0 {
		payloads := applyLeagueToPayloads(lg.ID, livePayloads)
		if err := s.ingestion.UpsertRawPayloads(ctx, "sportmonks", payloads); err != nil {
			return fmt.Errorf("upsert live standings raw payloads league=%s: %w", lg.ID, err)
		}
	}

	seasonID, ok := s.cfg.SeasonIDByLeague[strings.TrimSpace(lg.ID)]
	if !ok || seasonID <= 0 {
		return nil
	}

	return s.syncFixtureBundle(ctx, lg.ID, seasonID, teamMappings, fixtureBundleSyncOptions{
		onlyLiveWindow:   true,
		upsertRawPayload: false,
	})
}

type fixtureBundleSyncOptions struct {
	onlyLiveWindow   bool
	upsertRawPayload bool
}

func (s *SportDataSyncService) syncFixtureBundle(
	ctx context.Context,
	leagueID string,
	seasonID int64,
	teamMappings teamMappings,
	opts fixtureBundleSyncOptions,
) error {
	bundle, err := s.provider.FetchFixtureBundleBySeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("fetch fixture bundle from sport data provider season_id=%d league=%s: %w", seasonID, leagueID, err)
	}

	if opts.onlyLiveWindow {
		originalBundle := bundle
		gameweekFilter := selectLiveWindowGameweeks(bundle.Fixtures, time.Now().UTC())
		if len(gameweekFilter) > 0 {
			filtered := filterFixtureBundleByGameweeks(bundle, gameweekFilter)
			if len(filtered.Fixtures) == 0 && len(originalBundle.Fixtures) > 0 {
				s.logger.WarnContext(ctx,
					"gameweek window filtering returned zero fixtures; fallback to unfiltered season fixtures",
					"league_id", leagueID,
					"season_id", seasonID,
					"filter_size", len(gameweekFilter),
					"original_fixtures", len(originalBundle.Fixtures),
				)
			} else {
				bundle = filtered
			}
		}
	}

	playerMappings, err := s.loadPlayerMappings(ctx, leagueID)
	if err != nil {
		return err
	}

	mappedFixtures := mapExternalFixturesToDomain(leagueID, bundle.Fixtures, teamMappings)
	if len(mappedFixtures) > 0 {
		if err := s.ingestion.UpsertFixtures(ctx, mappedFixtures); err != nil {
			return fmt.Errorf("upsert fixtures from sport data provider league=%s: %w", leagueID, err)
		}
	}
	if err := s.syncFixtureDerivedData(ctx, leagueID, bundle, teamMappings, playerMappings); err != nil {
		return err
	}

	if opts.upsertRawPayload && len(bundle.RawPayloads) > 0 {
		payloads := applyLeagueToPayloads(leagueID, bundle.RawPayloads)
		if err := s.ingestion.UpsertRawPayloads(ctx, "sportmonks", payloads); err != nil {
			return fmt.Errorf("upsert fixture raw payloads league=%s: %w", leagueID, err)
		}
	}

	return nil
}

func (s *SportDataSyncService) SyncTopScorers(ctx context.Context, lg league.League) error {
	ctx, span := startUsecaseSpan(ctx, "usecase.SportDataSyncService.SyncTopScorers")
	defer span.End()

	leagueID := strings.TrimSpace(lg.ID)
	if leagueID == "" {
		return fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}
	if !s.cfg.Enabled {
		return nil
	}
	if s.provider == nil || s.topScore == nil {
		return fmt.Errorf("%w: top scorers sync dependencies are not configured", ErrDependencyUnavailable)
	}

	seasonID, ok := s.cfg.SeasonIDByLeague[leagueID]
	if !ok || seasonID <= 0 {
		return nil
	}

	typeIDs := make([]int, 0, len(TopScoreTypeMap))
	for _, typeID := range TopScoreTypeMap {
		typeIDs = append(typeIDs, typeID)
	}
	sort.Ints(typeIDs)

	const maxPagesPerType = 20
	var syncedTypeCount int
	failedTypeIDs := make([]int, 0)

	for _, typeID := range typeIDs {
		page := 1
		allRows := make([]ExternalTopScorers, 0)
		typeFailed := false

		for {
			if page > maxPagesPerType {
				s.logger.WarnContext(ctx,
					"top scorers pagination limit reached; stop this type to protect sync runtime",
					"league_id", leagueID,
					"season_id", seasonID,
					"type_id", typeID,
					"max_pages", maxPagesPerType,
				)
				break
			}

			dataTopScorers, hasMore, err := s.provider.FetchTopScorersBySeasonID(ctx, int(seasonID), page, typeID)
			if err != nil {
				typeFailed = true
				s.logger.WarnContext(ctx,
					"fetch top scorers failed; skip remaining pages for this type",
					"league_id", leagueID,
					"season_id", seasonID,
					"type_id", typeID,
					"page", page,
					"error", err,
				)
				break
			}
			allRows = append(allRows, dataTopScorers...)
			if !hasMore {
				break
			}
			page++
		}

		mapped := mappingToModel(allRows, leagueID)
		if len(mapped) == 0 {
			if typeFailed {
				failedTypeIDs = append(failedTypeIDs, typeID)
			}
			continue
		}

		if err := s.topScore.UpsertTopScorers(ctx, mapped); err != nil {
			typeFailed = true
			s.logger.WarnContext(ctx,
				"upsert top scorers failed",
				"league_id", leagueID,
				"season_id", seasonID,
				"type_id", typeID,
				"row_count", len(mapped),
				"error", err,
			)
		}

		if typeFailed {
			failedTypeIDs = append(failedTypeIDs, typeID)
			continue
		}
		syncedTypeCount++
	}

	if syncedTypeCount == 0 && len(failedTypeIDs) > 0 {
		return fmt.Errorf("top scorers sync failed for all types league=%s season_id=%d failed_types=%v", leagueID, seasonID, failedTypeIDs)
	}

	return nil
}

func mappingToModel(a []ExternalTopScorers, leagueID string) (scorers []topscorers.TopScorers) {
	for _, item := range a {
		scorers = append(scorers, topscorers.TopScorers{
			TypeID:           item.TypeID,
			TypeName:         item.TypeName,
			Rank:             item.Rank,
			Total:            item.Total,
			LeagueID:         leagueID,
			PlayerID:         item.PlayerID,
			Season:           item.Season,
			ParticipantID:    item.ParticipantID,
			PlayerName:       item.PlayerName,
			ImagePlayer:      item.ImagePlayer,
			Nationality:      item.Nationality,
			ImageNationality: item.ImageNationality,
			ParticipantName:  item.ParticipantName,
			ImageParticipant: item.ImageParticipant,
			PositionName:     item.PositionName,
		})
	}

	return scorers
}

type teamMappings struct {
	byRefID map[int64]string
	byName  map[string]string
}

type playerMappings struct {
	byRefID map[int64]string
	byRef   map[int64]player.Player
}

func (s *SportDataSyncService) loadTeamMappings(ctx context.Context, leagueID string) (teamMappings, error) {
	teams, err := s.teamRepo.ListByLeague(ctx, leagueID)
	if err != nil {
		return teamMappings{}, fmt.Errorf("list teams for sport data sync league=%s: %w", leagueID, err)
	}

	out := teamMappings{
		byRefID: make(map[int64]string, len(teams)),
		byName:  make(map[string]string, len(teams)*2),
	}
	for _, item := range teams {
		if item.TeamRefID > 0 {
			out.byRefID[item.TeamRefID] = item.ID
		}
		normalizedName := normalizeTeamName(item.Name)
		if normalizedName != "" {
			out.byName[normalizedName] = item.ID
		}
		normalizedShort := normalizeTeamName(item.Short)
		if normalizedShort != "" {
			out.byName[normalizedShort] = item.ID
		}
	}

	return out, nil
}

func (s *SportDataSyncService) loadPlayerMappings(ctx context.Context, leagueID string) (playerMappings, error) {
	items, err := s.playerRepo.ListByLeague(ctx, leagueID)
	if err != nil {
		return playerMappings{}, fmt.Errorf("list players for sport data sync league=%s: %w", leagueID, err)
	}

	out := playerMappings{
		byRefID: make(map[int64]string, len(items)),
		byRef:   make(map[int64]player.Player, len(items)),
	}
	for _, item := range items {
		if item.PlayerRefID > 0 {
			out.byRefID[item.PlayerRefID] = item.ID
			out.byRef[item.PlayerRefID] = item
		}
	}

	return out, nil
}

func (s *SportDataSyncService) syncFixtureDerivedData(
	ctx context.Context,
	leagueID string,
	bundle ExternalFixtureBundle,
	teamMappings teamMappings,
	playerMappings playerMappings,
) error {
	teamStatsByFixture := mapExternalTeamStatsByFixture(leagueID, bundle.TeamStats, teamMappings)
	playerStatsByFixture := mapExternalPlayerStatsByFixture(leagueID, bundle.PlayerStats, teamMappings, playerMappings)
	eventsByFixture := mapExternalFixtureEventsByFixture(leagueID, bundle.Events, teamMappings, playerMappings)

	mappedTeamStatsCount := countTeamStatsMap(teamStatsByFixture)
	if mappedTeamStatsCount < len(bundle.TeamStats) {
		s.logger.WarnContext(ctx, "some team stats could not be mapped", "league_id", leagueID, "provider_count", len(bundle.TeamStats), "mapped_count", mappedTeamStatsCount)
	}
	mappedPlayerStatsCount := countPlayerStatsMap(playerStatsByFixture)
	if mappedPlayerStatsCount < len(bundle.PlayerStats) {
		s.logger.WarnContext(ctx, "some player stats could not be mapped", "league_id", leagueID, "provider_count", len(bundle.PlayerStats), "mapped_count", mappedPlayerStatsCount)
	}
	mappedEventsCount := countFixtureEventsMap(eventsByFixture)
	if mappedEventsCount < len(bundle.Events) {
		s.logger.WarnContext(ctx, "some fixture events could not be mapped", "league_id", leagueID, "provider_count", len(bundle.Events), "mapped_count", mappedEventsCount)
	}

	fixtureIDs := fixtureIDsForDerivedData(teamStatsByFixture, playerStatsByFixture, eventsByFixture)
	for _, fixtureID := range fixtureIDs {
		stats, hasTeamStats := teamStatsByFixture[fixtureID]
		if hasTeamStats {
			if err := s.ingestion.UpsertTeamFixtureStats(ctx, fixtureID, stats); err != nil {
				return fmt.Errorf("upsert team fixture stats fixture=%s league=%s: %w", fixtureID, leagueID, err)
			}
		}

		statsPlayers, hasPlayerStats := playerStatsByFixture[fixtureID]
		if hasPlayerStats {
			if err := s.ingestion.UpsertPlayerFixtureStats(ctx, fixtureID, statsPlayers); err != nil {
				return fmt.Errorf("upsert player fixture stats fixture=%s league=%s: %w", fixtureID, leagueID, err)
			}
		}

		events, hasEvents := eventsByFixture[fixtureID]
		if hasEvents || hasTeamStats || hasPlayerStats {
			if err := s.ingestion.ReplaceFixtureEvents(ctx, fixtureID, events); err != nil {
				return fmt.Errorf("replace fixture events fixture=%s league=%s: %w", fixtureID, leagueID, err)
			}
		}
	}

	return nil
}

func mapExternalFixturesToDomain(leagueID string, items []ExternalFixture, mappings teamMappings) []fixture.Fixture {
	if len(items) == 0 {
		return nil
	}

	out := make([]fixture.Fixture, 0, len(items))
	for _, item := range items {
		if item.ExternalID <= 0 || item.KickoffAt.IsZero() {
			continue
		}

		homeTeamID := resolveTeamPublicID(mappings, item.HomeTeamExternalID, item.HomeTeamName)
		awayTeamID := resolveTeamPublicID(mappings, item.AwayTeamExternalID, item.AwayTeamName)
		winnerTeamID := resolveTeamPublicID(mappings, item.WinnerTeamExternalID, "")

		gameweek := item.Gameweek
		if gameweek <= 0 {
			gameweek = 1
		}

		out = append(out, fixture.Fixture{
			ID:           buildFixturePublicID(leagueID, item.ExternalID),
			LeagueID:     leagueID,
			Gameweek:     gameweek,
			HomeTeam:     strings.TrimSpace(item.HomeTeamName),
			AwayTeam:     strings.TrimSpace(item.AwayTeamName),
			HomeTeamID:   homeTeamID,
			AwayTeamID:   awayTeamID,
			FixtureRefID: item.ExternalID,
			KickoffAt:    item.KickoffAt.UTC(),
			Venue:        strings.TrimSpace(item.Venue),
			HomeScore:    cloneIntPtr(item.HomeScore),
			AwayScore:    cloneIntPtr(item.AwayScore),
			Status:       fixture.NormalizeStatus(item.Status),
			WinnerTeamID: winnerTeamID,
			FinishedAt:   cloneTimePtr(item.FinishedAt),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Gameweek != out[j].Gameweek {
			return out[i].Gameweek < out[j].Gameweek
		}
		if !out[i].KickoffAt.Equal(out[j].KickoffAt) {
			return out[i].KickoffAt.Before(out[j].KickoffAt)
		}
		return out[i].ID < out[j].ID
	})

	return out
}

func mapExternalStandingsToDomain(leagueID string, items []ExternalStanding, mappings teamMappings) []leaguestanding.Standing {
	if len(items) == 0 {
		return []leaguestanding.Standing{}
	}

	out := make([]leaguestanding.Standing, 0, len(items))
	for _, item := range items {
		teamID := resolveTeamPublicID(mappings, item.TeamExternalID, item.TeamName)
		if teamID == "" || item.Position <= 0 {
			continue
		}

		goalDifference := item.GoalDifference
		if goalDifference == 0 && (item.GoalsFor != 0 || item.GoalsAgainst != 0) {
			goalDifference = item.GoalsFor - item.GoalsAgainst
		}

		out = append(out, leaguestanding.Standing{
			LeagueID:        leagueID,
			TeamID:          teamID,
			Position:        item.Position,
			Played:          maxInt(item.Played, 0),
			Won:             maxInt(item.Won, 0),
			Draw:            maxInt(item.Draw, 0),
			Lost:            maxInt(item.Lost, 0),
			GoalsFor:        maxInt(item.GoalsFor, 0),
			GoalsAgainst:    maxInt(item.GoalsAgainst, 0),
			GoalDifference:  goalDifference,
			Points:          maxInt(item.Points, 0),
			Form:            strings.TrimSpace(item.Form),
			SourceUpdatedAt: cloneTimePtr(item.SourceUpdatedAt),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Position != out[j].Position {
			return out[i].Position < out[j].Position
		}
		if out[i].Points != out[j].Points {
			return out[i].Points > out[j].Points
		}
		return out[i].TeamID < out[j].TeamID
	})

	return out
}

func mapExternalTeamsToDomain(leagueID string, items []ExternalTeam, mappings teamMappings) []team.Team {
	if len(items) == 0 {
		return []team.Team{}
	}

	out := make([]team.Team, 0, len(items))
	for _, item := range items {
		if item.ExternalID <= 0 {
			continue
		}

		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}

		teamID := strings.TrimSpace(mappings.byRefID[item.ExternalID])
		if teamID == "" {
			teamID = buildTeamPublicID(leagueID, item.ExternalID)
		}
		short := normalizeTeamShort(item.Short, name)

		out = append(out, team.Team{
			ID:        teamID,
			LeagueID:  leagueID,
			Name:      name,
			Short:     short,
			ImageURL:  strings.TrimSpace(item.ImageURL),
			TeamRefID: item.ExternalID,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ID < out[j].ID
	})

	return out
}

func mapExternalPlayersToDomain(
	leagueID string,
	items []ExternalPlayer,
	teamMappings teamMappings,
	playerMappings playerMappings,
) []player.Player {
	if len(items) == 0 {
		return []player.Player{}
	}

	out := make([]player.Player, 0, len(items))
	seen := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item.ExternalID <= 0 {
			continue
		}
		if _, exists := seen[item.ExternalID]; exists {
			continue
		}
		seen[item.ExternalID] = struct{}{}

		existing := playerMappings.byRef[item.ExternalID]
		playerID := strings.TrimSpace(playerMappings.byRefID[item.ExternalID])
		if playerID == "" {
			playerID = buildPlayerPublicID(leagueID, item.ExternalID)
		}

		teamID := resolveTeamPublicID(teamMappings, item.TeamExternalID, "")
		if teamID == "" {
			teamID = strings.TrimSpace(existing.TeamID)
		}
		if teamID == "" {
			continue
		}

		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = strings.TrimSpace(existing.Name)
		}
		if name == "" {
			name = fmt.Sprintf("Player %d", item.ExternalID)
		}

		pos := normalizeExternalPlayerPosition(item.Position)
		if pos == "" {
			pos = player.Position(strings.TrimSpace(string(existing.Position)))
		}
		if pos == "" {
			pos = player.PositionMidfielder
		}

		price := item.Price
		if price <= 0 {
			price = existing.Price
		}
		if price <= 0 {
			price = 50
		}

		imageURL := strings.TrimSpace(item.ImageURL)
		if imageURL == "" {
			imageURL = strings.TrimSpace(existing.ImageURL)
		}

		out = append(out, player.Player{
			ID:          playerID,
			LeagueID:    leagueID,
			TeamID:      teamID,
			Name:        name,
			Position:    pos,
			Price:       price,
			ImageURL:    imageURL,
			PlayerRefID: item.ExternalID,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ID < out[j].ID
	})

	return out
}

func mapExternalTeamStatsByFixture(
	leagueID string,
	items []ExternalTeamFixtureStat,
	teamMappings teamMappings,
) map[string][]teamstats.FixtureStat {
	out := make(map[string][]teamstats.FixtureStat)
	for _, item := range items {
		if item.FixtureExternalID <= 0 {
			continue
		}
		fixtureID := buildFixturePublicID(leagueID, item.FixtureExternalID)
		teamID := resolveTeamPublicID(teamMappings, item.TeamExternalID, item.TeamName)
		if fixtureID == "" {
			continue
		}
		if teamID == "" && item.TeamExternalID <= 0 {
			continue
		}
		out[fixtureID] = append(out[fixtureID], teamstats.FixtureStat{
			FixtureID:         fixtureID,
			FixtureExternalID: item.FixtureExternalID,
			TeamID:            teamID,
			TeamExternalID:    item.TeamExternalID,
			PossessionPct:     item.PossessionPct,
			Shots:             maxInt(item.Shots, 0),
			ShotsOnTarget:     maxInt(item.ShotsOnTarget, 0),
			Corners:           maxInt(item.Corners, 0),
			Fouls:             maxInt(item.Fouls, 0),
			Offsides:          maxInt(item.Offsides, 0),
			AdvancedStats:     copyMap(item.AdvancedStats),
		})
	}
	return out
}

func mapExternalPlayerStatsByFixture(
	leagueID string,
	items []ExternalPlayerFixtureStat,
	teamMappings teamMappings,
	playerMappings playerMappings,
) map[string][]playerstats.FixtureStat {
	out := make(map[string][]playerstats.FixtureStat)
	for _, item := range items {
		if item.FixtureExternalID <= 0 || item.PlayerExternalID <= 0 {
			continue
		}

		fixtureID := buildFixturePublicID(leagueID, item.FixtureExternalID)
		playerID := strings.TrimSpace(playerMappings.byRefID[item.PlayerExternalID])
		teamID := resolveTeamPublicID(teamMappings, item.TeamExternalID, "")
		if fixtureID == "" {
			continue
		}
		if playerID == "" && item.PlayerExternalID <= 0 {
			continue
		}
		if teamID == "" && item.TeamExternalID <= 0 {
			continue
		}

		out[fixtureID] = append(out[fixtureID], playerstats.FixtureStat{
			FixtureID:         fixtureID,
			FixtureExternalID: item.FixtureExternalID,
			PlayerID:          playerID,
			PlayerExternalID:  item.PlayerExternalID,
			TeamID:            teamID,
			TeamExternalID:    item.TeamExternalID,
			MinutesPlayed:     maxInt(item.MinutesPlayed, 0),
			Goals:             maxInt(item.Goals, 0),
			Assists:           maxInt(item.Assists, 0),
			CleanSheet:        item.CleanSheet,
			YellowCards:       maxInt(item.YellowCards, 0),
			RedCards:          maxInt(item.RedCards, 0),
			Saves:             maxInt(item.Saves, 0),
			FantasyPoints:     maxInt(item.FantasyPoints, 0),
			AdvancedStats:     copyMap(item.AdvancedStats),
		})
	}
	return out
}

func mapExternalFixtureEventsByFixture(
	leagueID string,
	items []ExternalFixtureEvent,
	teamMappings teamMappings,
	playerMappings playerMappings,
) map[string][]playerstats.FixtureEvent {
	out := make(map[string][]playerstats.FixtureEvent)
	for _, item := range items {
		if item.FixtureExternalID <= 0 {
			continue
		}

		fixtureID := buildFixturePublicID(leagueID, item.FixtureExternalID)
		teamID := resolveTeamPublicID(teamMappings, item.TeamExternalID, "")
		playerID := strings.TrimSpace(playerMappings.byRefID[item.PlayerExternalID])
		assistID := strings.TrimSpace(playerMappings.byRefID[item.AssistPlayerExternalID])
		if fixtureID == "" || strings.TrimSpace(item.EventType) == "" {
			continue
		}

		out[fixtureID] = append(out[fixtureID], playerstats.FixtureEvent{
			EventID:                item.EventExternalID,
			FixtureID:              fixtureID,
			FixtureExternalID:      item.FixtureExternalID,
			TeamID:                 teamID,
			TeamExternalID:         item.TeamExternalID,
			PlayerID:               playerID,
			PlayerExternalID:       item.PlayerExternalID,
			AssistPlayerID:         assistID,
			AssistPlayerExternalID: item.AssistPlayerExternalID,
			EventType:              strings.TrimSpace(item.EventType),
			Detail:                 strings.TrimSpace(item.Detail),
			Minute:                 maxInt(item.Minute, 0),
			ExtraMinute:            maxInt(item.ExtraMinute, 0),
			Metadata:               copyMap(item.Metadata),
		})
	}
	return out
}

func mapExternalStatTypesToDomain(items []ExternalStatType) []statvalue.Type {
	if len(items) == 0 {
		return nil
	}

	out := make([]statvalue.Type, 0, len(items))
	seen := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item.ExternalTypeID <= 0 {
			continue
		}
		if _, ok := seen[item.ExternalTypeID]; ok {
			continue
		}
		seen[item.ExternalTypeID] = struct{}{}

		out = append(out, statvalue.Type{
			ExternalTypeID: item.ExternalTypeID,
			Name:           strings.TrimSpace(item.Name),
			DeveloperName:  strings.TrimSpace(item.DeveloperName),
			Code:           strings.TrimSpace(item.Code),
			ModelType:      strings.TrimSpace(item.ModelType),
			StatGroup:      strings.TrimSpace(item.StatGroup),
			Metadata:       copyMap(item.Metadata),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ExternalTypeID < out[j].ExternalTypeID
	})
	return out
}

func mapExternalTeamStatisticsToDomain(
	leagueID string,
	items []ExternalTeamStatValue,
	teamMappings teamMappings,
) []statvalue.TeamValue {
	if len(items) == 0 {
		return nil
	}

	out := make([]statvalue.TeamValue, 0, len(items))
	for _, item := range items {
		if item.SeasonRefID <= 0 || item.StatTypeExternalID <= 0 {
			continue
		}
		teamID := resolveTeamPublicID(teamMappings, item.TeamExternalID, "")
		fixtureID := ""
		if item.FixtureExternalID > 0 {
			fixtureID = buildFixturePublicID(leagueID, item.FixtureExternalID)
		}
		statKey := strings.TrimSpace(item.StatKey)
		if statKey == "" {
			statKey = strings.TrimSpace(item.StatTypeName)
		}
		if statKey == "" {
			statKey = fmt.Sprintf("type-%d", item.StatTypeExternalID)
		}

		out = append(out, statvalue.TeamValue{
			LeagueID:           leagueID,
			SeasonRefID:        item.SeasonRefID,
			TeamID:             teamID,
			ExternalTeamID:     item.TeamExternalID,
			FixtureID:          fixtureID,
			ExternalFixtureID:  item.FixtureExternalID,
			StatTypeExternalID: item.StatTypeExternalID,
			StatKey:            statKey,
			Scope:              normalizeStatScope(item.Scope),
			ValueNum:           cloneFloatPtr(item.ValueNum),
			ValueText:          strings.TrimSpace(item.ValueText),
			ValueJSON:          copyMap(item.ValueJSON),
			SourceUpdatedAt:    cloneTimePtr(item.SourceUpdatedAt),
			Metadata:           copyMap(item.Metadata),
		})
	}
	return out
}

func mapExternalPlayerStatisticsToDomain(
	leagueID string,
	items []ExternalPlayerStatValue,
	playerMappings playerMappings,
	teamMappings teamMappings,
) []statvalue.PlayerValue {
	if len(items) == 0 {
		return nil
	}

	out := make([]statvalue.PlayerValue, 0, len(items))
	for _, item := range items {
		if item.SeasonRefID <= 0 || item.StatTypeExternalID <= 0 {
			continue
		}
		playerID := strings.TrimSpace(playerMappings.byRefID[item.PlayerExternalID])
		teamID := resolveTeamPublicID(teamMappings, item.TeamExternalID, "")
		fixtureID := ""
		if item.FixtureExternalID > 0 {
			fixtureID = buildFixturePublicID(leagueID, item.FixtureExternalID)
		}
		statKey := strings.TrimSpace(item.StatKey)
		if statKey == "" {
			statKey = strings.TrimSpace(item.StatTypeName)
		}
		if statKey == "" {
			statKey = fmt.Sprintf("type-%d", item.StatTypeExternalID)
		}

		out = append(out, statvalue.PlayerValue{
			LeagueID:           leagueID,
			SeasonRefID:        item.SeasonRefID,
			PlayerID:           playerID,
			ExternalPlayerID:   item.PlayerExternalID,
			TeamID:             teamID,
			ExternalTeamID:     item.TeamExternalID,
			FixtureID:          fixtureID,
			ExternalFixtureID:  item.FixtureExternalID,
			StatTypeExternalID: item.StatTypeExternalID,
			StatKey:            statKey,
			Scope:              normalizeStatScope(item.Scope),
			ValueNum:           cloneFloatPtr(item.ValueNum),
			ValueText:          strings.TrimSpace(item.ValueText),
			ValueJSON:          copyMap(item.ValueJSON),
			SourceUpdatedAt:    cloneTimePtr(item.SourceUpdatedAt),
			Metadata:           copyMap(item.Metadata),
		})
	}
	return out
}

func normalizeStatScope(scope string) string {
	scope = strings.TrimSpace(strings.ToLower(scope))
	if scope == "" {
		return "total"
	}
	scope = strings.ReplaceAll(scope, "-", "_")
	scope = strings.ReplaceAll(scope, " ", "_")
	switch scope {
	case "overall", "all", "value":
		return "total"
	default:
		return scope
	}
}

func applyLeagueToPayloads(leagueID string, items []rawdata.Payload) []rawdata.Payload {
	if len(items) == 0 {
		return nil
	}

	out := make([]rawdata.Payload, 0, len(items))
	leagueID = strings.TrimSpace(leagueID)
	for _, item := range items {
		row := item
		if strings.TrimSpace(row.LeaguePublicID) == "" {
			row.LeaguePublicID = leagueID
		}
		out = append(out, row)
	}
	return out
}

func resolveTeamPublicID(mappings teamMappings, teamRefID int64, teamName string) string {
	if teamRefID > 0 {
		if teamID := strings.TrimSpace(mappings.byRefID[teamRefID]); teamID != "" {
			return teamID
		}
	}

	normalized := normalizeTeamName(teamName)
	if normalized == "" {
		return ""
	}

	if teamID := strings.TrimSpace(mappings.byName[normalized]); teamID != "" {
		return teamID
	}

	for key, teamID := range mappings.byName {
		if key == "" {
			continue
		}
		if strings.Contains(normalized, key) || strings.Contains(key, normalized) {
			return teamID
		}
	}

	return ""
}

func normalizeTeamName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}

	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}

	return strings.Trim(builder.String(), "-")
}

func normalizeTeamShort(short, name string) string {
	value := strings.TrimSpace(short)
	if value == "" {
		value = inferTeamShortFromName(name)
	}
	if len(value) < 2 {
		value = strings.ToUpper(value)
		if len(value) == 1 {
			value += "X"
		}
	}
	if len(value) > 10 {
		value = value[:10]
	}
	return value
}

func inferTeamShortFromName(name string) string {
	words := strings.Fields(strings.TrimSpace(name))
	if len(words) == 0 {
		return "TM"
	}
	if len(words) == 1 {
		word := strings.ToUpper(words[0])
		if len(word) >= 3 {
			return word[:3]
		}
		if len(word) == 2 {
			return word
		}
		return word + "X"
	}
	var out strings.Builder
	for _, word := range words {
		if len(out.String()) >= 4 {
			break
		}
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}
		out.WriteByte(byte(strings.ToUpper(word)[0]))
	}
	short := out.String()
	if len(short) < 2 {
		return "TM"
	}
	return short
}

func normalizeExternalPlayerPosition(raw string) player.Position {
	value := strings.ToUpper(strings.TrimSpace(raw))
	switch value {
	case string(player.PositionGoalkeeper):
		return player.PositionGoalkeeper
	case string(player.PositionDefender):
		return player.PositionDefender
	case string(player.PositionMidfielder):
		return player.PositionMidfielder
	case string(player.PositionForward):
		return player.PositionForward
	}

	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "goalkeeper", "keeper", "goalie", "gk":
		return player.PositionGoalkeeper
	case "defender", "def", "centre-back", "center-back", "full-back", "wing-back":
		return player.PositionDefender
	case "midfielder", "mid", "winger", "attacking midfielder", "defensive midfielder":
		return player.PositionMidfielder
	case "forward", "fwd", "attacker", "striker":
		return player.PositionForward
	default:
		return ""
	}
}

func buildTeamPublicID(leagueID string, teamRefID int64) string {
	return "sm-" + sanitizePublicIDSegment(leagueID) + "-team-" + fmt.Sprintf("%d", teamRefID)
}

func buildPlayerPublicID(leagueID string, playerRefID int64) string {
	return "sm-" + sanitizePublicIDSegment(leagueID) + "-player-" + fmt.Sprintf("%d", playerRefID)
}

func buildFixturePublicID(leagueID string, fixtureRefID int64) string {
	return "sm-" + sanitizePublicIDSegment(leagueID) + "-fixture-" + fmt.Sprintf("%d", fixtureRefID)
}

func sanitizePublicIDSegment(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "league"
	}

	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}

	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "league"
	}
	if idx := strings.IndexByte(out, '-'); idx > 0 {
		return out[:idx]
	}
	return out
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	v := value.UTC()
	return &v
}

func cloneFloatPtr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func resolveStandingsSnapshotGameweek(items []ExternalStanding) int {
	maxPlayed := 0
	for _, item := range items {
		if item.Played > maxPlayed {
			maxPlayed = item.Played
		}
	}
	if maxPlayed <= 0 {
		return 1
	}
	return maxPlayed
}

func copyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func fixtureIDsFromExternalFixtures(leagueID string, fixtures []ExternalFixture) []string {
	if len(fixtures) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(fixtures))
	out := make([]string, 0, len(fixtures))
	for _, item := range fixtures {
		if item.ExternalID <= 0 {
			continue
		}
		fixtureID := buildFixturePublicID(leagueID, item.ExternalID)
		if fixtureID == "" {
			continue
		}
		if _, exists := seen[fixtureID]; exists {
			continue
		}
		seen[fixtureID] = struct{}{}
		out = append(out, fixtureID)
	}
	sort.Strings(out)
	return out
}

func fixtureIDsForDerivedData(
	teamStatsByFixture map[string][]teamstats.FixtureStat,
	playerStatsByFixture map[string][]playerstats.FixtureStat,
	eventsByFixture map[string][]playerstats.FixtureEvent,
) []string {
	if len(teamStatsByFixture) == 0 && len(playerStatsByFixture) == 0 && len(eventsByFixture) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(teamStatsByFixture)+len(playerStatsByFixture)+len(eventsByFixture))
	out := make([]string, 0, len(seen))
	for fixtureID := range teamStatsByFixture {
		if strings.TrimSpace(fixtureID) == "" {
			continue
		}
		if _, ok := seen[fixtureID]; ok {
			continue
		}
		seen[fixtureID] = struct{}{}
		out = append(out, fixtureID)
	}
	for fixtureID := range playerStatsByFixture {
		if strings.TrimSpace(fixtureID) == "" {
			continue
		}
		if _, ok := seen[fixtureID]; ok {
			continue
		}
		seen[fixtureID] = struct{}{}
		out = append(out, fixtureID)
	}
	for fixtureID := range eventsByFixture {
		if strings.TrimSpace(fixtureID) == "" {
			continue
		}
		if _, ok := seen[fixtureID]; ok {
			continue
		}
		seen[fixtureID] = struct{}{}
		out = append(out, fixtureID)
	}
	sort.Strings(out)
	return out
}

func selectLiveWindowGameweeks(fixtures []ExternalFixture, now time.Time) map[int]struct{} {
	if len(fixtures) == 0 {
		return nil
	}

	type fixtureCandidate struct {
		gameweek int
		kickoff  time.Time
	}

	candidates := make([]fixtureCandidate, 0, len(fixtures))
	for _, item := range fixtures {
		if item.Gameweek <= 0 || item.KickoffAt.IsZero() {
			continue
		}
		candidates = append(candidates, fixtureCandidate{
			gameweek: item.Gameweek,
			kickoff:  item.KickoffAt.UTC(),
		})
	}
	if len(candidates) == 0 {
		return nil
	}

	closest := candidates[0]
	closestDiff := durationAbs(now.Sub(closest.kickoff))
	for _, item := range candidates[1:] {
		diff := durationAbs(now.Sub(item.kickoff))
		if diff < closestDiff || (diff == closestDiff && item.kickoff.Before(closest.kickoff)) {
			closest = item
			closestDiff = diff
		}
	}

	window := map[int]struct{}{
		closest.gameweek:     {},
		closest.gameweek + 1: {},
	}
	if closest.gameweek > 1 {
		window[closest.gameweek-1] = struct{}{}
	}
	return window
}

func durationAbs(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}

func countTeamStatsMap(items map[string][]teamstats.FixtureStat) int {
	total := 0
	for _, row := range items {
		total += len(row)
	}
	return total
}

func countPlayerStatsMap(items map[string][]playerstats.FixtureStat) int {
	total := 0
	for _, row := range items {
		total += len(row)
	}
	return total
}

func countFixtureEventsMap(items map[string][]playerstats.FixtureEvent) int {
	total := 0
	for _, row := range items {
		total += len(row)
	}
	return total
}
