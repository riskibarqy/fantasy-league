package app

import (
	"context"
	"fmt"
	"github.com/riskibarqy/fantasy-league/internal/domain/topscorers"
	"github.com/riskibarqy/fantasy-league/internal/platform/logging"
	"net/http"
	"time"

	_ "github.com/lib/pq"
	"github.com/riskibarqy/fantasy-league/external/anubis"
	"github.com/riskibarqy/fantasy-league/external/jobqueue"
	"github.com/riskibarqy/fantasy-league/external/sportmonks"
	"github.com/riskibarqy/fantasy-league/internal/config"
	customleaguedomain "github.com/riskibarqy/fantasy-league/internal/domain/customleague"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	fixturedomain "github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	jobschedulerdomain "github.com/riskibarqy/fantasy-league/internal/domain/jobscheduler"
	leaguedomain "github.com/riskibarqy/fantasy-league/internal/domain/league"
	leaguestandingdomain "github.com/riskibarqy/fantasy-league/internal/domain/leaguestanding"
	lineupdomain "github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	onboardingdomain "github.com/riskibarqy/fantasy-league/internal/domain/onboarding"
	playerdomain "github.com/riskibarqy/fantasy-league/internal/domain/player"
	playerstatsdomain "github.com/riskibarqy/fantasy-league/internal/domain/playerstats"
	scoringdomain "github.com/riskibarqy/fantasy-league/internal/domain/scoring"
	teamdomain "github.com/riskibarqy/fantasy-league/internal/domain/team"
	teamstatsdomain "github.com/riskibarqy/fantasy-league/internal/domain/teamstats"
	cacherepo "github.com/riskibarqy/fantasy-league/internal/infrastructure/repository/cache"
	postgresrepo "github.com/riskibarqy/fantasy-league/internal/infrastructure/repository/postgres"
	"github.com/riskibarqy/fantasy-league/internal/interfaces/httpapi"
	basecache "github.com/riskibarqy/fantasy-league/internal/platform/cache"
	idgen "github.com/riskibarqy/fantasy-league/internal/platform/id"
	"github.com/riskibarqy/fantasy-league/internal/platform/resilience"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"github.com/uptrace/opentelemetry-go-extra/otelsqlx"
)

type fixtureIngestionWriter interface {
	UpsertFixtures(ctx context.Context, items []fixturedomain.Fixture) error
}

func NewHTTPHandler(cfg config.Config, logger *logging.Logger) (http.Handler, func() error, error) {
	db, err := otelsqlx.Open("postgres", normalizeDBURL(cfg.DBURL, cfg.DBDisablePreparedBinary),
		otelsql.WithDBSystem("postgresql"),
		otelsql.WithDBName(dbNameFromURL(cfg.DBURL)),
		otelsql.WithQueryFormatter(formatDBQueryForTrace),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("open postgres connection: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("ping postgres: %w", err)
	}

	var leagueRepo leaguedomain.Repository = postgresrepo.NewLeagueRepository(db)
	var topScoreRepo topscorers.Repository = postgresrepo.NewTopScorersRepository(db)
	var teamRepo teamdomain.Repository = postgresrepo.NewTeamRepository(db)
	var playerRepo playerdomain.Repository = postgresrepo.NewPlayerRepository(db)
	fixtureWriterBase := postgresrepo.NewFixtureRepository(db)
	var fixtureWriter fixtureIngestionWriter = fixtureWriterBase
	var fixtureRepo fixturedomain.Repository = fixtureWriterBase
	var leagueStandingRepo leaguestandingdomain.Repository = postgresrepo.NewLeagueStandingRepository(db)
	var lineupRepo lineupdomain.Repository = postgresrepo.NewLineupRepository(db)
	var squadRepo fantasy.Repository = postgresrepo.NewSquadRepository(db)
	var playerStatsRepo playerstatsdomain.Repository = postgresrepo.NewPlayerStatsRepository(db)
	var teamStatsRepo teamstatsdomain.Repository = postgresrepo.NewTeamStatsRepository(db)
	statValueRepo := postgresrepo.NewStatValueRepository(db)
	rawDataRepo := postgresrepo.NewRawDataRepository(db)
	var customLeagueRepo customleaguedomain.Repository = postgresrepo.NewCustomLeagueRepository(db)
	var onboardingRepo onboardingdomain.Repository = postgresrepo.NewOnboardingRepository(db)
	var scoringRepo scoringdomain.Repository = postgresrepo.NewScoringRepository(db)
	var jobDispatchRepo jobschedulerdomain.Repository = postgresrepo.NewJobDispatchRepository(db)

	if cfg.CacheEnabled {
		cacheStore := basecache.NewStore(cfg.CacheTTL)
		leagueRepo = cacherepo.NewLeagueRepository(leagueRepo, cacheStore)
		teamRepo = cacherepo.NewTeamRepository(teamRepo, cacheStore)
		playerRepo = cacherepo.NewPlayerRepository(playerRepo, cacheStore)
		fixtureCachedRepo := cacherepo.NewFixtureRepository(fixtureRepo, cacheStore)
		fixtureRepo = fixtureCachedRepo
		fixtureWriter = fixtureCachedRepo
		lineupRepo = cacherepo.NewLineupRepository(lineupRepo, cacheStore)
		squadRepo = cacherepo.NewSquadRepository(squadRepo, cacheStore)
		playerStatsRepo = cacherepo.NewPlayerStatsRepository(playerStatsRepo, cacheStore)
		teamStatsRepo = cacherepo.NewTeamStatsRepository(teamStatsRepo, cacheStore)
		customLeagueRepo = cacherepo.NewCustomLeagueRepository(customLeagueRepo, cacheStore)
	}

	leagueSvc := usecase.NewLeagueService(leagueRepo, teamRepo)
	teamSvc := usecase.NewTeamService(leagueRepo, teamRepo, teamStatsRepo)
	playerSvc := usecase.NewPlayerService(leagueRepo, playerRepo)
	topScoreSvc := usecase.NewTopScoreService(topScoreRepo)
	playerStatsSvc := usecase.NewPlayerStatsService(playerStatsRepo)
	fixtureSvc := usecase.NewFixtureService(leagueRepo, fixtureRepo)
	leagueStandingSvc := usecase.NewLeagueStandingService(leagueRepo, leagueStandingRepo, fixtureRepo)
	lineupSvc := usecase.NewLineupService(leagueRepo, playerRepo, lineupRepo, squadRepo)
	scoringSvc := usecase.NewScoringService(fixtureRepo, squadRepo, lineupRepo, playerStatsRepo, customLeagueRepo, scoringRepo)
	dashboardSvc := usecase.NewDashboardService(leagueRepo, fixtureRepo, squadRepo, customLeagueRepo, scoringSvc)
	customLeagueSvc := usecase.NewCustomLeagueService(leagueRepo, squadRepo, customLeagueRepo, scoringSvc, idgen.NewRandomGenerator())
	ingestionSvc := usecase.NewIngestionService(fixtureWriter, leagueStandingRepo, playerStatsRepo, teamStatsRepo, rawDataRepo)
	var sportDataProvider usecase.SportDataSyncProvider
	if cfg.SportMonksEnabled {
		sportDataProvider = sportmonks.NewClient(sportmonks.ClientConfig{
			BaseURL:    cfg.SportMonksBaseURL,
			Token:      cfg.SportMonksToken,
			Timeout:    cfg.SportMonksTimeout,
			MaxRetries: cfg.SportMonksMaxRetries,
			Logger:     logger,
			CircuitBreaker: resilience.CircuitBreakerConfig{
				Enabled:          cfg.SportMonksCircuitEnabled,
				FailureThreshold: cfg.SportMonksCircuitFailureCount,
				OpenTimeout:      cfg.SportMonksCircuitOpenTimeout,
				HalfOpenMaxReq:   cfg.SportMonksCircuitHalfOpenMaxReq,
			},
		})
	}
	sportDataSyncSvc := usecase.NewSportDataSyncService(
		sportDataProvider,
		teamRepo,
		playerRepo,
		topScoreRepo,
		ingestionSvc,
		usecase.SportDataSyncConfig{
			Enabled:          cfg.SportMonksEnabled,
			SeasonIDByLeague: cfg.SportMonksSeasonIDByLeague,
			LeagueIDByLeague: cfg.SportMonksLeagueIDByLeague,
		},
		logger,
	)
	sportDataSyncSvc.SetStatValueRepository(statValueRepo)
	jobQueue := usecase.NewNoopJobQueue()
	if cfg.QStashEnabled {
		jobQueue = jobqueue.NewQStashPublisher(jobqueue.QStashPublisherConfig{
			BaseURL:          cfg.QStashBaseURL,
			Token:            cfg.QStashToken,
			TargetBaseURL:    cfg.QStashTargetBaseURL,
			Retries:          cfg.QStashRetries,
			InternalJobToken: cfg.InternalJobToken,
			CircuitBreaker: resilience.CircuitBreakerConfig{
				Enabled:          cfg.QStashCircuitEnabled,
				FailureThreshold: cfg.QStashCircuitFailureCount,
				OpenTimeout:      cfg.QStashCircuitOpenTimeout,
				HalfOpenMaxReq:   cfg.QStashCircuitHalfOpenMaxReq,
			},
		}, logger)
	}
	jobOrchestrator := usecase.NewJobOrchestratorService(
		leagueRepo,
		fixtureRepo,
		scoringSvc,
		sportDataSyncSvc,
		jobQueue,
		jobDispatchRepo,
		usecase.JobOrchestratorConfig{
			ScheduleInterval: cfg.JobScheduleInterval,
			LiveInterval:     cfg.JobLiveInterval,
			PreKickoffLead:   cfg.JobPreKickoffLead,
		},
		logger,
	)
	squadSvc := usecase.NewSquadService(
		leagueRepo,
		playerRepo,
		squadRepo,
		fantasy.DefaultRules(),
		idgen.NewRandomGenerator(),
		logger,
	)
	squadSvc.SetScoringUpdater(scoringSvc)
	squadSvc.SetDefaultLeagueJoiner(customLeagueSvc)
	lineupSvc.SetScoringUpdater(scoringSvc)
	onboardingSvc := usecase.NewOnboardingService(teamRepo, onboardingRepo, squadSvc, lineupSvc, customLeagueSvc)

	anubisClient := anubis.NewClient(
		&http.Client{Timeout: cfg.AnubisTimeout},
		cfg.AnubisBaseURL,
		cfg.AnubisIntrospectURL,
		cfg.AnubisAdminKey,
		resilience.CircuitBreakerConfig{
			Enabled:          cfg.AnubisCircuitEnabled,
			FailureThreshold: cfg.AnubisCircuitFailureCount,
			OpenTimeout:      cfg.AnubisCircuitOpenTimeout,
			HalfOpenMaxReq:   cfg.AnubisCircuitHalfOpenMaxReq,
		},
		logger,
	)

	handler := httpapi.NewHandler(
		leagueSvc,
		teamSvc,
		playerSvc,
		playerStatsSvc,
		fixtureSvc,
		leagueStandingSvc,
		jobOrchestrator,
		lineupSvc,
		dashboardSvc,
		squadSvc,
		ingestionSvc,
		sportDataSyncSvc,
		customLeagueSvc,
		scoringSvc,
		onboardingSvc,
		jobDispatchRepo,
		topScoreSvc,
		logger,
	)
	router := httpapi.NewRouter(
		handler,
		anubisClient,
		logger,
		cfg.SwaggerEnabled,
		cfg.CORSAllowedOrigins,
		cfg.InternalJobToken,
		cfg.UptraceCaptureRequestBody,
		cfg.UptraceRequestBodyMaxBytes,
	)

	return router, db.Close, nil
}
