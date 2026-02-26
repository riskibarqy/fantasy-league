package httpapi

import (
	"context"
	"fmt"
	"github.com/riskibarqy/fantasy-league/internal/platform/logging"
	"hash/fnv"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/riskibarqy/fantasy-league/internal/domain/customleague"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/jobscheduler"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/leaguestanding"
	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	"github.com/riskibarqy/fantasy-league/internal/domain/onboarding"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	"github.com/riskibarqy/fantasy-league/internal/domain/playerstats"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
	"github.com/riskibarqy/fantasy-league/internal/domain/teamstats"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

type Handler struct {
	leagueService         *usecase.LeagueService
	teamService           *usecase.TeamService
	playerService         *usecase.PlayerService
	playerStatsService    *usecase.PlayerStatsService
	fixtureService        *usecase.FixtureService
	leagueStandingService *usecase.LeagueStandingService
	jobOrchestrator       *usecase.JobOrchestratorService
	lineupService         *usecase.LineupService
	dashboardService      *usecase.DashboardService
	squadService          *usecase.SquadService
	ingestionService      *usecase.IngestionService
	sportDataSyncService  *usecase.SportDataSyncService
	customLeagueService   *usecase.CustomLeagueService
	onboardingService     *usecase.OnboardingService
	jobDispatchRepo       jobscheduler.Repository
	logger                *logging.Logger
	validator             *validator.Validate
}

func NewHandler(
	leagueService *usecase.LeagueService,
	teamService *usecase.TeamService,
	playerService *usecase.PlayerService,
	playerStatsService *usecase.PlayerStatsService,
	fixtureService *usecase.FixtureService,
	leagueStandingService *usecase.LeagueStandingService,
	jobOrchestrator *usecase.JobOrchestratorService,
	lineupService *usecase.LineupService,
	dashboardService *usecase.DashboardService,
	squadService *usecase.SquadService,
	ingestionService *usecase.IngestionService,
	sportDataSyncService *usecase.SportDataSyncService,
	customLeagueService *usecase.CustomLeagueService,
	onboardingService *usecase.OnboardingService,
	jobDispatchRepo jobscheduler.Repository,
	logger *logging.Logger,
) *Handler {
	if logger == nil {
		logger = logging.Default()
	}

	return &Handler{
		leagueService:         leagueService,
		teamService:           teamService,
		playerService:         playerService,
		playerStatsService:    playerStatsService,
		fixtureService:        fixtureService,
		leagueStandingService: leagueStandingService,
		jobOrchestrator:       jobOrchestrator,
		lineupService:         lineupService,
		dashboardService:      dashboardService,
		squadService:          squadService,
		ingestionService:      ingestionService,
		sportDataSyncService:  sportDataSyncService,
		customLeagueService:   customLeagueService,
		onboardingService:     onboardingService,
		jobDispatchRepo:       jobDispatchRepo,
		logger:                logger,
		validator:             validator.New(),
	}
}
func (h *Handler) validateRequest(ctx context.Context, payload any) error {
	ctx, span := startSpan(ctx, "httpapi.Handler.validateRequest")
	defer span.End()

	if err := h.validator.StructCtx(ctx, payload); err != nil {
		return fmt.Errorf("%w: validation failed: %v", usecase.ErrInvalidInput, err)
	}

	return nil
}

type upsertSquadRequest struct {
	LeagueID  string   `json:"league_id" validate:"required"`
	SquadName string   `json:"squad_name" validate:"required,max=100"`
	PlayerIDs []string `json:"player_ids" validate:"required,len=15,dive,required"`
}

type pickSquadRequest struct {
	LeagueID  string   `json:"league_id" validate:"required"`
	SquadName string   `json:"squad_name" validate:"omitempty,max=100"`
	PlayerIDs []string `json:"player_ids" validate:"required,len=15,dive,required"`
}

type onboardingFavoriteClubRequest struct {
	LeagueID string `json:"league_id" validate:"required"`
	TeamID   string `json:"team_id" validate:"required"`
}

type onboardingPickSquadRequest struct {
	LeagueID      string   `json:"league_id" validate:"required"`
	SquadName     string   `json:"squad_name" validate:"omitempty,max=100"`
	PlayerIDs     []string `json:"player_ids" validate:"required,len=15,dive,required"`
	GoalkeeperID  string   `json:"goalkeeper_id" validate:"required"`
	DefenderIDs   []string `json:"defender_ids" validate:"required,min=3,max=5,dive,required"`
	MidfielderIDs []string `json:"midfielder_ids" validate:"required,min=3,max=5,dive,required"`
	ForwardIDs    []string `json:"forward_ids" validate:"required,min=1,max=3,dive,required"`
	SubstituteIDs []string `json:"substitute_ids" validate:"required,len=4,dive,required"`
	CaptainID     string   `json:"captain_id" validate:"required"`
	ViceCaptainID string   `json:"vice_captain_id" validate:"required"`
}

type addPlayerToSquadRequest struct {
	LeagueID  string `json:"league_id" validate:"required"`
	SquadName string `json:"squad_name" validate:"omitempty,max=100"`
	PlayerID  string `json:"player_id" validate:"required"`
}

type ingestPlayerFixtureStatsRequest struct {
	FixtureID string                          `json:"fixture_id" validate:"required"`
	Stats     []ingestPlayerFixtureStatRecord `json:"stats" validate:"required,dive"`
}

type ingestFixturesRequest struct {
	Fixtures []ingestFixtureRecord `json:"fixtures" validate:"required,dive"`
}

type ingestFixtureRecord struct {
	ID           string         `json:"id" validate:"required"`
	LeagueID     string         `json:"league_id" validate:"required"`
	Gameweek     int            `json:"gameweek" validate:"required,gt=0"`
	HomeTeam     string         `json:"home_team" validate:"required"`
	AwayTeam     string         `json:"away_team" validate:"required"`
	HomeTeamID   string         `json:"home_team_id"`
	AwayTeamID   string         `json:"away_team_id"`
	FixtureRefID int64          `json:"fixture_ref_id"`
	KickoffAt    string         `json:"kickoff_at" validate:"required"`
	Venue        string         `json:"venue"`
	HomeScore    *int           `json:"home_score,omitempty"`
	AwayScore    *int           `json:"away_score,omitempty"`
	Status       string         `json:"status"`
	WinnerTeamID string         `json:"winner_team_id"`
	FinishedAt   string         `json:"finished_at,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
}

type ingestPlayerFixtureStatRecord struct {
	PlayerID          string         `json:"player_id"`
	ExternalPlayerID  int64          `json:"external_player_id"`
	TeamID            string         `json:"team_id"`
	ExternalTeamID    int64          `json:"external_team_id"`
	ExternalFixtureID int64          `json:"external_fixture_id"`
	MinutesPlayed     int            `json:"minutes_played"`
	Goals             int            `json:"goals"`
	Assists           int            `json:"assists"`
	CleanSheet        bool           `json:"clean_sheet"`
	YellowCards       int            `json:"yellow_cards"`
	RedCards          int            `json:"red_cards"`
	Saves             int            `json:"saves"`
	FantasyPoints     int            `json:"fantasy_points"`
	AdvancedStats     map[string]any `json:"advanced_stats,omitempty"`
	Payload           map[string]any `json:"payload,omitempty"`
}

type ingestTeamFixtureStatsRequest struct {
	FixtureID string                        `json:"fixture_id" validate:"required"`
	Stats     []ingestTeamFixtureStatRecord `json:"stats" validate:"required,dive"`
}

type ingestTeamFixtureStatRecord struct {
	TeamID            string         `json:"team_id"`
	ExternalTeamID    int64          `json:"external_team_id"`
	ExternalFixtureID int64          `json:"external_fixture_id"`
	PossessionPct     float64        `json:"possession_pct"`
	Shots             int            `json:"shots"`
	ShotsOnTarget     int            `json:"shots_on_target"`
	Corners           int            `json:"corners"`
	Fouls             int            `json:"fouls"`
	Offsides          int            `json:"offsides"`
	AdvancedStats     map[string]any `json:"advanced_stats,omitempty"`
	Payload           map[string]any `json:"payload,omitempty"`
}

type ingestFixtureEventsRequest struct {
	FixtureID string                     `json:"fixture_id" validate:"required"`
	Events    []ingestFixtureEventRecord `json:"events" validate:"required,dive"`
}

type ingestFixtureEventRecord struct {
	EventID                int64          `json:"event_id"`
	TeamID                 string         `json:"team_id"`
	ExternalTeamID         int64          `json:"external_team_id"`
	PlayerID               string         `json:"player_id"`
	ExternalPlayerID       int64          `json:"external_player_id"`
	AssistPlayerID         string         `json:"assist_player_id"`
	ExternalAssistPlayerID int64          `json:"external_assist_player_id"`
	ExternalFixtureID      int64          `json:"external_fixture_id"`
	EventType              string         `json:"event_type" validate:"required"`
	Detail                 string         `json:"detail"`
	Minute                 int            `json:"minute"`
	ExtraMinute            int            `json:"extra_minute"`
	Metadata               map[string]any `json:"metadata,omitempty"`
	Payload                map[string]any `json:"payload,omitempty"`
}

type ingestRawPayloadsRequest struct {
	Source  string                   `json:"source" validate:"omitempty,max=40"`
	Records []ingestRawPayloadRecord `json:"records" validate:"required,dive"`
}

type ingestLeagueStandingsRequest struct {
	LeagueID string                       `json:"league_id" validate:"required"`
	IsLive   bool                         `json:"is_live"`
	Items    []ingestLeagueStandingRecord `json:"items" validate:"required,min=1,dive"`
}

type ingestLeagueStandingRecord struct {
	TeamID          string `json:"team_id" validate:"required"`
	Position        int    `json:"position" validate:"required,gt=0"`
	Played          int    `json:"played" validate:"gte=0"`
	Won             int    `json:"won" validate:"gte=0"`
	Draw            int    `json:"draw" validate:"gte=0"`
	Lost            int    `json:"lost" validate:"gte=0"`
	GoalsFor        int    `json:"goals_for" validate:"gte=0"`
	GoalsAgainst    int    `json:"goals_against" validate:"gte=0"`
	GoalDifference  int    `json:"goal_difference"`
	Points          int    `json:"points" validate:"gte=0"`
	Form            string `json:"form" validate:"max=64"`
	SourceUpdatedAt string `json:"source_updated_at"`
}

type ingestRawPayloadRecord struct {
	EntityType      string         `json:"entity_type" validate:"required,max=80"`
	EntityKey       string         `json:"entity_key" validate:"required,max=200"`
	LeagueID        string         `json:"league_id,omitempty"`
	FixtureID       string         `json:"fixture_id,omitempty"`
	TeamID          string         `json:"team_id,omitempty"`
	PlayerID        string         `json:"player_id,omitempty"`
	SourceUpdatedAt string         `json:"source_updated_at,omitempty"`
	Payload         map[string]any `json:"payload" validate:"required"`
}

type createCustomLeagueRequest struct {
	LeagueID string `json:"league_id" validate:"required"`
	Name     string `json:"name" validate:"required,max=120"`
}

type updateCustomLeagueRequest struct {
	Name string `json:"name" validate:"required,max=120"`
}

type joinCustomLeagueByInviteRequest struct {
	InviteCode string `json:"invite_code" validate:"required,min=6,max=32"`
}

type lineupUpsertRequest struct {
	LeagueID      string   `json:"leagueId" validate:"required"`
	GoalkeeperID  string   `json:"goalkeeperId" validate:"required"`
	DefenderIDs   []string `json:"defenderIds" validate:"required,min=3,max=5,dive,required"`
	MidfielderIDs []string `json:"midfielderIds" validate:"required,min=3,max=5,dive,required"`
	ForwardIDs    []string `json:"forwardIds" validate:"required,min=1,max=3,dive,required"`
	SubstituteIDs []string `json:"substituteIds" validate:"required,len=4,dive,required"`
	CaptainID     string   `json:"captainId" validate:"required"`
	ViceCaptainID string   `json:"viceCaptainId" validate:"required"`
	UpdatedAt     string   `json:"updatedAt"`
}

type internalJobSyncRequest struct {
	LeagueID   string `json:"league_id"`
	Force      bool   `json:"force"`
	DispatchID string `json:"dispatch_id"`
}

type getSquadRequest struct {
	LeagueID string `validate:"required"`
}

type dashboardDTO struct {
	Gameweek         int     `json:"gameweek"`
	Budget           float64 `json:"budget"`
	TeamValue        float64 `json:"teamValue"`
	TotalPoints      int     `json:"totalPoints"`
	Rank             int     `json:"rank"`
	SelectedLeagueID string  `json:"selectedLeagueId"`
}

type leaguePublicDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CountryCode string `json:"countryCode"`
	LogoURL     string `json:"logoUrl"`
}

type teamDTO struct {
	ID        string   `json:"id"`
	LeagueID  string   `json:"leagueId"`
	Name      string   `json:"name"`
	Short     string   `json:"short"`
	LogoURL   string   `json:"logoUrl"`
	TeamColor []string `json:"teamColor,omitempty"`
}

type teamDetailDTO struct {
	Team       teamDTO            `json:"team"`
	Statistics teamSeasonStatsDTO `json:"statistics"`
}

type teamSeasonStatsDTO struct {
	Appearances          int     `json:"appearances"`
	AveragePossessionPct float64 `json:"averagePossessionPct"`
	TotalShots           int     `json:"totalShots"`
	TotalShotsOnTarget   int     `json:"totalShotsOnTarget"`
	TotalCorners         int     `json:"totalCorners"`
	TotalFouls           int     `json:"totalFouls"`
	TotalOffsides        int     `json:"totalOffsides"`
}

type teamMatchHistoryDTO struct {
	FixtureID      string  `json:"fixtureId"`
	Gameweek       int     `json:"gameweek"`
	KickoffAt      string  `json:"kickoffAt"`
	HomeTeam       string  `json:"homeTeam"`
	AwayTeam       string  `json:"awayTeam"`
	OpponentTeam   string  `json:"opponentTeam"`
	OpponentTeamID string  `json:"opponentTeamId,omitempty"`
	IsHome         bool    `json:"isHome"`
	PossessionPct  float64 `json:"possessionPct"`
	Shots          int     `json:"shots"`
	ShotsOnTarget  int     `json:"shotsOnTarget"`
	Corners        int     `json:"corners"`
	Fouls          int     `json:"fouls"`
	Offsides       int     `json:"offsides"`
}

type playerPublicDTO struct {
	ID              string   `json:"id"`
	LeagueID        string   `json:"leagueId"`
	Name            string   `json:"name"`
	Club            string   `json:"club"`
	Position        string   `json:"position"`
	Price           float64  `json:"price"`
	Form            float64  `json:"form"`
	ProjectedPoints float64  `json:"projectedPoints"`
	IsInjured       bool     `json:"isInjured"`
	ImageURL        string   `json:"imageUrl"`
	TeamLogoURL     string   `json:"teamLogoUrl"`
	TeamColor       []string `json:"teamColor,omitempty"`
}

type squadPlayerDTO struct {
	ID              string   `json:"id"`
	LeagueID        string   `json:"leagueId"`
	Name            string   `json:"name"`
	Club            string   `json:"club"`
	Position        string   `json:"position"`
	Price           float64  `json:"price"`
	Form            float64  `json:"form"`
	ProjectedPoints float64  `json:"projectedPoints"`
	IsInjured       bool     `json:"isInjured"`
	InSquad         bool     `json:"inSquad"`
	ImageURL        string   `json:"imageUrl"`
	TeamLogoURL     string   `json:"teamLogoUrl"`
	TeamColor       []string `json:"teamColor,omitempty"`
}

type fixtureDTO struct {
	ID              string `json:"id"`
	LeagueID        string `json:"leagueId"`
	Gameweek        int    `json:"gameweek"`
	HomeTeam        string `json:"homeTeam"`
	AwayTeam        string `json:"awayTeam"`
	HomeTeamLogoURL string `json:"homeTeamLogoUrl"`
	AwayTeamLogoURL string `json:"awayTeamLogoUrl"`
	Kickoff         string `json:"kickoffAt"`
	Venue           string `json:"venue"`
	HomeScore       *int   `json:"homeScore,omitempty"`
	AwayScore       *int   `json:"awayScore,omitempty"`
	Status          string `json:"status"`
	WinnerTeamID    string `json:"winnerTeamId,omitempty"`
	FinishedAt      string `json:"finishedAt,omitempty"`
}

type fixtureEventDTO struct {
	EventID                int64          `json:"eventId"`
	FixtureID              string         `json:"fixtureId"`
	FixtureExternalID      int64          `json:"fixtureExternalId,omitempty"`
	TeamID                 string         `json:"teamId,omitempty"`
	TeamExternalID         int64          `json:"teamExternalId,omitempty"`
	PlayerID               string         `json:"playerId,omitempty"`
	PlayerExternalID       int64          `json:"playerExternalId,omitempty"`
	AssistPlayerID         string         `json:"assistPlayerId,omitempty"`
	AssistPlayerExternalID int64          `json:"assistPlayerExternalId,omitempty"`
	EventType              string         `json:"eventType"`
	Detail                 string         `json:"detail,omitempty"`
	Minute                 int            `json:"minute"`
	ExtraMinute            int            `json:"extraMinute"`
	Metadata               map[string]any `json:"metadata,omitempty"`
}

type fixtureTeamStatsDTO struct {
	TeamID         string         `json:"teamId"`
	TeamExternalID int64          `json:"teamExternalId,omitempty"`
	TeamName       string         `json:"teamName,omitempty"`
	PossessionPct  float64        `json:"possessionPct"`
	Shots          int            `json:"shots"`
	ShotsOnTarget  int            `json:"shotsOnTarget"`
	Corners        int            `json:"corners"`
	Fouls          int            `json:"fouls"`
	Offsides       int            `json:"offsides"`
	AdvancedStats  map[string]any `json:"advancedStats,omitempty"`
}

type fixturePlayerStatsDTO struct {
	PlayerID         string         `json:"playerId"`
	PlayerExternalID int64          `json:"playerExternalId,omitempty"`
	PlayerName       string         `json:"playerName,omitempty"`
	TeamID           string         `json:"teamId"`
	TeamExternalID   int64          `json:"teamExternalId,omitempty"`
	TeamName         string         `json:"teamName,omitempty"`
	MinutesPlayed    int            `json:"minutesPlayed"`
	Goals            int            `json:"goals"`
	Assists          int            `json:"assists"`
	CleanSheet       bool           `json:"cleanSheet"`
	YellowCards      int            `json:"yellowCards"`
	RedCards         int            `json:"redCards"`
	Saves            int            `json:"saves"`
	FantasyPoints    int            `json:"fantasyPoints"`
	AdvancedStats    map[string]any `json:"advancedStats,omitempty"`
}

type fixtureDetailsDTO struct {
	Fixture     fixtureDTO              `json:"fixture"`
	TeamStats   []fixtureTeamStatsDTO   `json:"teamStats"`
	PlayerStats []fixturePlayerStatsDTO `json:"playerStats"`
	Events      []fixtureEventDTO       `json:"events"`
}

type leagueStandingDTO struct {
	LeagueID       string `json:"league_id"`
	TeamID         string `json:"team_id"`
	TeamName       string `json:"team_name,omitempty"`
	TeamLogoURL    string `json:"team_logo_url,omitempty"`
	Position       int    `json:"position"`
	Played         int    `json:"played"`
	Won            int    `json:"won"`
	Draw           int    `json:"draw"`
	Lost           int    `json:"lost"`
	GoalsFor       int    `json:"goals_for"`
	GoalsAgainst   int    `json:"goals_against"`
	GoalDifference int    `json:"goal_difference"`
	Points         int    `json:"points"`
	Form           string `json:"form"`
	IsLive         bool   `json:"is_live"`
}

type playerDetailDTO struct {
	Player     playerPublicDTO         `json:"player"`
	Statistics playerStatisticsDTO     `json:"statistics"`
	History    []playerMatchHistoryDTO `json:"history"`
}

type playerStatisticsDTO struct {
	MinutesPlayed int `json:"minutesPlayed"`
	Goals         int `json:"goals"`
	Assists       int `json:"assists"`
	CleanSheets   int `json:"cleanSheets"`
	YellowCards   int `json:"yellowCards"`
	RedCards      int `json:"redCards"`
	Appearances   int `json:"appearances"`
	TotalPoints   int `json:"totalPoints"`
}

type playerMatchHistoryDTO struct {
	FixtureID   string `json:"fixtureId"`
	Gameweek    int    `json:"gameweek"`
	Opponent    string `json:"opponent"`
	HomeAway    string `json:"homeAway"`
	KickoffAt   string `json:"kickoffAt"`
	Minutes     int    `json:"minutes"`
	Goals       int    `json:"goals"`
	Assists     int    `json:"assists"`
	CleanSheet  bool   `json:"cleanSheet"`
	YellowCards int    `json:"yellowCards"`
	RedCards    int    `json:"redCards"`
	Points      int    `json:"points"`
}

type lineupDTO struct {
	LeagueID      string   `json:"leagueId"`
	GoalkeeperID  string   `json:"goalkeeperId"`
	DefenderIDs   []string `json:"defenderIds"`
	MidfielderIDs []string `json:"midfielderIds"`
	ForwardIDs    []string `json:"forwardIds"`
	SubstituteIDs []string `json:"substituteIds"`
	CaptainID     string   `json:"captainId"`
	ViceCaptainID string   `json:"viceCaptainId"`
	UpdatedAt     string   `json:"updatedAt"`
}

type onboardingProfileDTO struct {
	UserID              string `json:"user_id"`
	FavoriteLeagueID    string `json:"favorite_league_id,omitempty"`
	FavoriteTeamID      string `json:"favorite_team_id,omitempty"`
	CountryCode         string `json:"country_code,omitempty"`
	IPAddress           string `json:"ip_address,omitempty"`
	OnboardingCompleted bool   `json:"onboarding_completed"`
	UpdatedAtUTC        string `json:"updated_at_utc,omitempty"`
}

type onboardingCompleteResponseDTO struct {
	Profile onboardingProfileDTO `json:"profile"`
	Squad   squadDTO             `json:"squad"`
	Lineup  lineupDTO            `json:"lineup"`
}

type squadDTO struct {
	ID           string         `json:"id"`
	UserID       string         `json:"user_id"`
	LeagueID     string         `json:"league_id"`
	Name         string         `json:"name"`
	BudgetCap    int64          `json:"budget_cap"`
	TotalCost    int64          `json:"total_cost"`
	Picks        []squadPickDTO `json:"picks"`
	CreatedAtUTC string         `json:"created_at_utc"`
	UpdatedAtUTC string         `json:"updated_at_utc"`
}

type squadPickDTO struct {
	PlayerID string `json:"player_id"`
	TeamID   string `json:"team_id"`
	Position string `json:"position"`
	Price    int64  `json:"price"`
}

type customLeagueDTO struct {
	ID           string `json:"id"`
	LeagueID     string `json:"league_id"`
	CountryCode  string `json:"country_code,omitempty"`
	OwnerUserID  string `json:"owner_user_id"`
	Name         string `json:"name"`
	InviteCode   string `json:"invite_code"`
	IsDefault    bool   `json:"is_default"`
	CreatedAtUTC string `json:"created_at_utc"`
	UpdatedAtUTC string `json:"updated_at_utc"`
}

type customLeagueListDTO struct {
	ID           string `json:"id"`
	LeagueID     string `json:"league_id"`
	CountryCode  string `json:"country_code,omitempty"`
	OwnerUserID  string `json:"owner_user_id"`
	Name         string `json:"name"`
	InviteCode   string `json:"invite_code"`
	IsDefault    bool   `json:"is_default"`
	MyRank       int    `json:"my_rank"`
	RankMovement string `json:"rank_movement"`
	CreatedAtUTC string `json:"created_at_utc"`
	UpdatedAtUTC string `json:"updated_at_utc"`
}

type customLeagueStandingDTO struct {
	UserID           string `json:"user_id"`
	SquadID          string `json:"squad_id"`
	Points           int    `json:"points"`
	Rank             int    `json:"rank"`
	LastCalculatedAt string `json:"last_calculated_at,omitempty"`
	UpdatedAtUTC     string `json:"updated_at_utc"`
}

func leagueToPublicDTO(ctx context.Context, v league.League) leaguePublicDTO {
	ctx, span := startSpan(ctx, "httpapi.leagueToPublicDTO")
	defer span.End()

	return leaguePublicDTO{
		ID:          v.ID,
		Name:        v.Name,
		CountryCode: v.CountryCode,
		LogoURL:     leagueLogoURL(ctx, v.Name, v.CountryCode),
	}
}

func teamToDTO(ctx context.Context, v team.Team) teamDTO {
	ctx, span := startSpan(ctx, "httpapi.teamToDTO")
	defer span.End()

	return teamDTO{
		ID:        v.ID,
		LeagueID:  v.LeagueID,
		Name:      v.Name,
		Short:     v.Short,
		LogoURL:   teamLogoWithFallback(ctx, v.Name, v.ImageURL),
		TeamColor: teamColorArray(v.PrimaryColor, v.SecondaryColor),
	}
}

func teamDetailToDTO(ctx context.Context, v usecase.TeamDetails) teamDetailDTO {
	ctx, span := startSpan(ctx, "httpapi.teamDetailToDTO")
	defer span.End()

	return teamDetailDTO{
		Team:       teamToDTO(ctx, v.Team),
		Statistics: teamSeasonStatsToDTO(ctx, v.Statistics),
	}
}

func teamSeasonStatsToDTO(ctx context.Context, stats teamstats.SeasonStats) teamSeasonStatsDTO {
	ctx, span := startSpan(ctx, "httpapi.teamSeasonStatsToDTO")
	defer span.End()

	return teamSeasonStatsDTO{
		Appearances:          stats.Appearances,
		AveragePossessionPct: round1(ctx, stats.AveragePossessionPct),
		TotalShots:           stats.TotalShots,
		TotalShotsOnTarget:   stats.TotalShotsOnTarget,
		TotalCorners:         stats.TotalCorners,
		TotalFouls:           stats.TotalFouls,
		TotalOffsides:        stats.TotalOffsides,
	}
}

func leagueStandingToDTO(
	ctx context.Context,
	item leaguestanding.Standing,
	teamName string,
	teamLogoURL string,
) leagueStandingDTO {
	ctx, span := startSpan(ctx, "httpapi.leagueStandingToDTO")
	defer span.End()

	return leagueStandingDTO{
		LeagueID:       item.LeagueID,
		TeamID:         item.TeamID,
		TeamName:       strings.TrimSpace(teamName),
		TeamLogoURL:    strings.TrimSpace(teamLogoURL),
		Position:       item.Position,
		Played:         item.Played,
		Won:            item.Won,
		Draw:           item.Draw,
		Lost:           item.Lost,
		GoalsFor:       item.GoalsFor,
		GoalsAgainst:   item.GoalsAgainst,
		GoalDifference: item.GoalDifference,
		Points:         item.Points,
		Form:           strings.TrimSpace(item.Form),
		IsLive:         item.IsLive,
	}
}

func playerToPublicDTO(
	ctx context.Context,
	v player.Player,
	teamName,
	playerImage,
	teamLogo string,
	teamColor []string,
) playerPublicDTO {
	ctx, span := startSpan(ctx, "httpapi.playerToPublicDTO")
	defer span.End()

	if strings.TrimSpace(teamName) == "" {
		teamName = v.TeamID
	}

	form, projected := derivedPlayerMetrics(ctx, v.ID)
	injured := isInjured(ctx, v.ID)

	return playerPublicDTO{
		ID:              v.ID,
		LeagueID:        v.LeagueID,
		Name:            v.Name,
		Club:            teamName,
		Position:        string(v.Position),
		Price:           float64(v.Price) / 10.0,
		Form:            form,
		ProjectedPoints: projected,
		IsInjured:       injured,
		ImageURL:        playerImageWithFallback(ctx, v.ID, v.Name, playerImage),
		TeamLogoURL:     teamLogoWithFallback(ctx, teamName, teamLogo),
		TeamColor:       copyTeamColor(teamColor),
	}
}

func teamColorArray(primary, secondary string) []string {
	primary = strings.TrimSpace(primary)
	secondary = strings.TrimSpace(secondary)
	if primary == "" || secondary == "" {
		return nil
	}

	return []string{primary, secondary}
}

func copyTeamColor(teamColor []string) []string {
	if len(teamColor) < 2 {
		return nil
	}

	return []string{
		strings.TrimSpace(teamColor[0]),
		strings.TrimSpace(teamColor[1]),
	}
}

func fixtureToDTO(ctx context.Context, v fixture.Fixture, teamLogoByID map[string]string) fixtureDTO {
	ctx, span := startSpan(ctx, "httpapi.fixtureToDTO")
	defer span.End()

	homeLogo := teamLogoByID[v.HomeTeamID]
	if strings.TrimSpace(homeLogo) == "" {
		homeLogo = teamLogoURL(ctx, v.HomeTeam)
	}
	awayLogo := teamLogoByID[v.AwayTeamID]
	if strings.TrimSpace(awayLogo) == "" {
		awayLogo = teamLogoURL(ctx, v.AwayTeam)
	}

	return fixtureDTO{
		ID:              v.ID,
		LeagueID:        v.LeagueID,
		Gameweek:        v.Gameweek,
		HomeTeam:        v.HomeTeam,
		AwayTeam:        v.AwayTeam,
		HomeTeamLogoURL: homeLogo,
		AwayTeamLogoURL: awayLogo,
		Kickoff:         v.KickoffAt.UTC().Format(time.RFC3339),
		Venue:           v.Venue,
		HomeScore:       v.HomeScore,
		AwayScore:       v.AwayScore,
		Status:          fixture.NormalizeStatus(v.Status),
		WinnerTeamID:    strings.TrimSpace(v.WinnerTeamID),
		FinishedAt:      formatOptionalTime(v.FinishedAt),
	}
}

func lineupToDTO(ctx context.Context, item lineup.Lineup) lineupDTO {
	ctx, span := startSpan(ctx, "httpapi.lineupToDTO")
	defer span.End()

	return lineupDTO{
		LeagueID:      item.LeagueID,
		GoalkeeperID:  item.GoalkeeperID,
		DefenderIDs:   append([]string(nil), item.DefenderIDs...),
		MidfielderIDs: append([]string(nil), item.MidfielderIDs...),
		ForwardIDs:    append([]string(nil), item.ForwardIDs...),
		SubstituteIDs: append([]string(nil), item.SubstituteIDs...),
		CaptainID:     item.CaptainID,
		ViceCaptainID: item.ViceCaptainID,
		UpdatedAt:     item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func onboardingProfileToDTO(ctx context.Context, item onboarding.Profile) onboardingProfileDTO {
	ctx, span := startSpan(ctx, "httpapi.onboardingProfileToDTO")
	defer span.End()

	dto := onboardingProfileDTO{
		UserID:              item.UserID,
		FavoriteLeagueID:    item.FavoriteLeagueID,
		FavoriteTeamID:      item.FavoriteTeamID,
		CountryCode:         item.CountryCode,
		IPAddress:           item.IPAddress,
		OnboardingCompleted: item.OnboardingCompleted,
	}
	if !item.UpdatedAt.IsZero() {
		dto.UpdatedAtUTC = item.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return dto
}

func derivedPlayerMetrics(ctx context.Context, playerID string) (float64, float64) {
	ctx, span := startSpan(ctx, "httpapi.derivedPlayerMetrics")
	defer span.End()

	base := hash(ctx, playerID)
	form := 5.0 + float64(base%45)/10.0
	projected := 3.0 + float64(hash(ctx, playerID+"-proj")%80)/10.0
	return round1(ctx, form), round1(ctx, projected)
}

func derivedForm(ctx context.Context, playerID string) float64 {
	ctx, span := startSpan(ctx, "httpapi.derivedForm")
	defer span.End()

	base := hash(ctx, playerID)
	form := 5.0 + float64(base%45)/10.0
	return round1(ctx, form)
}

func derivedProjectedPoints(ctx context.Context, playerID string) float64 {
	ctx, span := startSpan(ctx, "httpapi.derivedProjectedPoints")
	defer span.End()

	projected := 3.0 + float64(hash(ctx, playerID+"-proj")%80)/10.0
	return round1(ctx, projected)
}

func isInjured(ctx context.Context, playerID string) bool {
	ctx, span := startSpan(ctx, "httpapi.isInjured")
	defer span.End()

	return hash(ctx, playerID+"-inj")%20 == 0
}

func hash(ctx context.Context, v string) uint32 {
	ctx, span := startSpan(ctx, "httpapi.hash")
	defer span.End()

	h := fnv.New32a()
	_, _ = h.Write([]byte(v))
	return h.Sum32()
}

func round1(ctx context.Context, v float64) float64 {
	ctx, span := startSpan(ctx, "httpapi.round1")
	defer span.End()

	return float64(int(v*10+0.5)) / 10.0
}

func leagueLogoURL(ctx context.Context, name, countryCode string) string {
	ctx, span := startSpan(ctx, "httpapi.leagueLogoURL")
	defer span.End()

	initials := leagueInitials(ctx, name)
	svg := fmt.Sprintf(
		`<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 320 180'><rect width='320' height='180' fill='#0b5730'/><rect x='8' y='8' width='304' height='164' rx='12' fill='#0f6e3d'/><text x='160' y='92' text-anchor='middle' fill='white' font-family='Arial, sans-serif' font-size='44' font-weight='700'>%s</text><text x='160' y='126' text-anchor='middle' fill='#e7f7eb' font-family='Arial, sans-serif' font-size='16'>%s</text></svg>`,
		initials,
		countryCode,
	)
	return "data:image/svg+xml," + url.QueryEscape(svg)
}

func teamLogoURL(ctx context.Context, teamName string) string {
	ctx, span := startSpan(ctx, "httpapi.teamLogoURL")
	defer span.End()

	name := strings.TrimSpace(teamName)
	if name == "" {
		name = "Team"
	}
	initials := leagueInitials(ctx, name)
	svg := fmt.Sprintf(
		`<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 256 256'><defs><linearGradient id='g' x1='0' y1='0' x2='1' y2='1'><stop offset='0%%' stop-color='#0f2d5c'/><stop offset='100%%' stop-color='#154a8a'/></linearGradient></defs><circle cx='128' cy='128' r='120' fill='url(#g)'/><circle cx='128' cy='128' r='96' fill='none' stroke='#dce9ff' stroke-width='8'/><text x='128' y='144' text-anchor='middle' fill='white' font-family='Arial, sans-serif' font-size='58' font-weight='700'>%s</text></svg>`,
		initials,
	)
	return "data:image/svg+xml," + url.QueryEscape(svg)
}

func teamLogoWithFallback(ctx context.Context, teamName, imageURL string) string {
	ctx, span := startSpan(ctx, "httpapi.teamLogoWithFallback")
	defer span.End()

	imageURL = strings.TrimSpace(imageURL)
	if imageURL != "" {
		return imageURL
	}

	return teamLogoURL(ctx, teamName)
}

func playerImageURL(ctx context.Context, playerID, playerName string) string {
	ctx, span := startSpan(ctx, "httpapi.playerImageURL")
	defer span.End()

	seed := strings.TrimSpace(playerID)
	if seed == "" {
		seed = strings.TrimSpace(playerName)
	}
	if seed == "" {
		seed = "player"
	}
	// Public placeholder avatar by deterministic seed.
	return "https://api.dicebear.com/9.x/adventurer-neutral/svg?seed=" + url.QueryEscape(seed)
}

func playerImageWithFallback(ctx context.Context, playerID, playerName, imageURL string) string {
	ctx, span := startSpan(ctx, "httpapi.playerImageWithFallback")
	defer span.End()

	imageURL = strings.TrimSpace(imageURL)
	if imageURL != "" {
		return imageURL
	}

	return playerImageURL(ctx, playerID, playerName)
}

func historyToDTO(ctx context.Context, playerTeamID string, history []playerstats.MatchHistory, teamNameByID map[string]string) []playerMatchHistoryDTO {
	ctx, span := startSpan(ctx, "httpapi.historyToDTO")
	defer span.End()

	out := make([]playerMatchHistoryDTO, 0, len(history))
	for _, item := range history {
		clubName := strings.TrimSpace(teamNameByID[item.TeamID])
		if clubName == "" {
			clubName = strings.TrimSpace(teamNameByID[playerTeamID])
		}

		homeAway := "away"
		opponent := item.HomeTeam
		if clubName != "" && strings.EqualFold(clubName, item.HomeTeam) {
			homeAway = "home"
			opponent = item.AwayTeam
		}
		if clubName != "" && strings.EqualFold(clubName, item.AwayTeam) {
			homeAway = "away"
			opponent = item.HomeTeam
		}
		if strings.TrimSpace(opponent) == "" {
			opponent = "TBD"
		}

		out = append(out, playerMatchHistoryDTO{
			FixtureID:   item.FixtureID,
			Gameweek:    item.Gameweek,
			Opponent:    opponent,
			HomeAway:    homeAway,
			KickoffAt:   item.KickoffAt.UTC().Format(time.RFC3339),
			Minutes:     item.MinutesPlayed,
			Goals:       item.Goals,
			Assists:     item.Assists,
			CleanSheet:  item.CleanSheet,
			YellowCards: item.YellowCards,
			RedCards:    item.RedCards,
			Points:      item.FantasyPoints,
		})
	}

	return out
}

func teamHistoryToDTO(ctx context.Context, history []teamstats.MatchHistory, teamNameByID map[string]string) []teamMatchHistoryDTO {
	ctx, span := startSpan(ctx, "httpapi.teamHistoryToDTO")
	defer span.End()

	out := make([]teamMatchHistoryDTO, 0, len(history))
	for _, item := range history {
		opponent := teamNameByID[item.OpponentTeamID]
		if strings.TrimSpace(opponent) == "" {
			opponent = item.OpponentTeamID
		}

		out = append(out, teamMatchHistoryDTO{
			FixtureID:      item.FixtureID,
			Gameweek:       item.Gameweek,
			KickoffAt:      item.KickoffAt.UTC().Format(time.RFC3339),
			HomeTeam:       item.HomeTeam,
			AwayTeam:       item.AwayTeam,
			OpponentTeam:   opponent,
			OpponentTeamID: item.OpponentTeamID,
			IsHome:         item.IsHome,
			PossessionPct:  round1(ctx, item.PossessionPct),
			Shots:          item.Shots,
			ShotsOnTarget:  item.ShotsOnTarget,
			Corners:        item.Corners,
			Fouls:          item.Fouls,
			Offsides:       item.Offsides,
		})
	}

	return out
}

func seasonStatsToDTO(ctx context.Context, stats playerstats.SeasonStats) playerStatisticsDTO {
	ctx, span := startSpan(ctx, "httpapi.seasonStatsToDTO")
	defer span.End()

	return playerStatisticsDTO{
		MinutesPlayed: stats.MinutesPlayed,
		Goals:         stats.Goals,
		Assists:       stats.Assists,
		CleanSheets:   stats.CleanSheets,
		YellowCards:   stats.YellowCards,
		RedCards:      stats.RedCards,
		Appearances:   stats.Appearances,
		TotalPoints:   stats.TotalPoints,
	}
}

func formatOptionalTime(v *time.Time) string {
	if v == nil || v.IsZero() {
		return ""
	}
	return v.UTC().Format(time.RFC3339)
}

func leagueInitials(ctx context.Context, name string) string {
	ctx, span := startSpan(ctx, "httpapi.leagueInitials")
	defer span.End()

	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return "FL"
	}
	if len(parts) == 1 {
		up := strings.ToUpper(parts[0])
		if len(up) <= 2 {
			return up
		}
		return up[:2]
	}

	return strings.ToUpper(parts[0][:1] + parts[1][:1])
}

func squadToDTO(ctx context.Context, v fantasy.Squad) squadDTO {
	ctx, span := startSpan(ctx, "httpapi.squadToDTO")
	defer span.End()

	picks := make([]squadPickDTO, 0, len(v.Picks))
	var total int64
	for _, pick := range v.Picks {
		total += pick.Price
		picks = append(picks, squadPickDTO{
			PlayerID: pick.PlayerID,
			TeamID:   pick.TeamID,
			Position: string(pick.Position),
			Price:    pick.Price,
		})
	}

	return squadDTO{
		ID:           v.ID,
		UserID:       v.UserID,
		LeagueID:     v.LeagueID,
		Name:         v.Name,
		BudgetCap:    v.BudgetCap,
		TotalCost:    total,
		Picks:        picks,
		CreatedAtUTC: v.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAtUTC: v.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func customLeagueToDTO(ctx context.Context, v customleague.Group) customLeagueDTO {
	ctx, span := startSpan(ctx, "httpapi.customLeagueToDTO")
	defer span.End()

	return customLeagueDTO{
		ID:           v.ID,
		LeagueID:     v.LeagueID,
		CountryCode:  v.CountryCode,
		OwnerUserID:  v.OwnerUserID,
		Name:         v.Name,
		InviteCode:   v.InviteCode,
		IsDefault:    v.IsDefault,
		CreatedAtUTC: v.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAtUTC: v.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func customLeagueListToDTO(ctx context.Context, v customleague.GroupWithMyStanding) customLeagueListDTO {
	ctx, span := startSpan(ctx, "httpapi.customLeagueListToDTO")
	defer span.End()

	return customLeagueListDTO{
		ID:           v.Group.ID,
		LeagueID:     v.Group.LeagueID,
		CountryCode:  v.Group.CountryCode,
		OwnerUserID:  v.Group.OwnerUserID,
		Name:         v.Group.Name,
		InviteCode:   v.Group.InviteCode,
		IsDefault:    v.Group.IsDefault,
		MyRank:       v.MyRank,
		RankMovement: string(v.RankMovement),
		CreatedAtUTC: v.Group.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAtUTC: v.Group.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func customLeagueStandingToDTO(ctx context.Context, v customleague.Standing) customLeagueStandingDTO {
	ctx, span := startSpan(ctx, "httpapi.customLeagueStandingToDTO")
	defer span.End()

	lastCalculatedAt := ""
	if v.LastCalculatedAt != nil && !v.LastCalculatedAt.IsZero() {
		lastCalculatedAt = v.LastCalculatedAt.UTC().Format(time.RFC3339)
	}

	return customLeagueStandingDTO{
		UserID:           v.UserID,
		SquadID:          v.SquadID,
		Points:           v.Points,
		Rank:             v.Rank,
		LastCalculatedAt: lastCalculatedAt,
		UpdatedAtUTC:     v.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
