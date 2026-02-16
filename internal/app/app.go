package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/riskibarqy/fantasy-league/internal/config"
	customleaguedomain "github.com/riskibarqy/fantasy-league/internal/domain/customleague"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	fixturedomain "github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	leaguedomain "github.com/riskibarqy/fantasy-league/internal/domain/league"
	lineupdomain "github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	playerdomain "github.com/riskibarqy/fantasy-league/internal/domain/player"
	playerstatsdomain "github.com/riskibarqy/fantasy-league/internal/domain/playerstats"
	teamdomain "github.com/riskibarqy/fantasy-league/internal/domain/team"
	"github.com/riskibarqy/fantasy-league/internal/infrastructure/account/anubis"
	cacherepo "github.com/riskibarqy/fantasy-league/internal/infrastructure/repository/cache"
	postgresrepo "github.com/riskibarqy/fantasy-league/internal/infrastructure/repository/postgres"
	"github.com/riskibarqy/fantasy-league/internal/interfaces/httpapi"
	basecache "github.com/riskibarqy/fantasy-league/internal/platform/cache"
	idgen "github.com/riskibarqy/fantasy-league/internal/platform/id"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func NewHTTPHandler(cfg config.Config, logger *slog.Logger) (http.Handler, func() error, error) {
	db, err := sqlx.Open("postgres", normalizeDBURL(cfg.DBURL, cfg.DBDisablePreparedBinary))
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
	var teamRepo teamdomain.Repository = postgresrepo.NewTeamRepository(db)
	var playerRepo playerdomain.Repository = postgresrepo.NewPlayerRepository(db)
	var fixtureRepo fixturedomain.Repository = postgresrepo.NewFixtureRepository(db)
	var lineupRepo lineupdomain.Repository = postgresrepo.NewLineupRepository(db)
	var squadRepo fantasy.Repository = postgresrepo.NewSquadRepository(db)
	var playerStatsRepo playerstatsdomain.Repository = postgresrepo.NewPlayerStatsRepository(db)
	var customLeagueRepo customleaguedomain.Repository = postgresrepo.NewCustomLeagueRepository(db)

	if cfg.CacheEnabled {
		cacheStore := basecache.NewStore(cfg.CacheTTL)
		leagueRepo = cacherepo.NewLeagueRepository(leagueRepo, cacheStore)
		teamRepo = cacherepo.NewTeamRepository(teamRepo, cacheStore)
		playerRepo = cacherepo.NewPlayerRepository(playerRepo, cacheStore)
		fixtureRepo = cacherepo.NewFixtureRepository(fixtureRepo, cacheStore)
		lineupRepo = cacherepo.NewLineupRepository(lineupRepo, cacheStore)
		squadRepo = cacherepo.NewSquadRepository(squadRepo, cacheStore)
		playerStatsRepo = cacherepo.NewPlayerStatsRepository(playerStatsRepo, cacheStore)
		customLeagueRepo = cacherepo.NewCustomLeagueRepository(customLeagueRepo, cacheStore)
	}

	leagueSvc := usecase.NewLeagueService(leagueRepo, teamRepo)
	playerSvc := usecase.NewPlayerService(leagueRepo, playerRepo)
	playerStatsSvc := usecase.NewPlayerStatsService(playerStatsRepo)
	fixtureSvc := usecase.NewFixtureService(leagueRepo, fixtureRepo)
	lineupSvc := usecase.NewLineupService(leagueRepo, playerRepo, lineupRepo, squadRepo)
	dashboardSvc := usecase.NewDashboardService(leagueRepo, playerRepo, fixtureRepo, lineupRepo)
	customLeagueSvc := usecase.NewCustomLeagueService(leagueRepo, squadRepo, customLeagueRepo, idgen.NewRandomGenerator())
	squadSvc := usecase.NewSquadService(
		leagueRepo,
		playerRepo,
		squadRepo,
		fantasy.DefaultRules(),
		idgen.NewRandomGenerator(),
		logger,
	)
	squadSvc.SetDefaultLeagueJoiner(customLeagueSvc)

	anubisClient := anubis.NewClient(
		&http.Client{Timeout: cfg.AnubisTimeout},
		cfg.AnubisBaseURL,
		cfg.AnubisIntrospectURL,
		cfg.AnubisAdminKey,
		anubis.CircuitBreakerConfig{
			Enabled:          cfg.AnubisCircuitEnabled,
			FailureThreshold: cfg.AnubisCircuitFailureCount,
			OpenTimeout:      cfg.AnubisCircuitOpenTimeout,
			HalfOpenMaxReq:   cfg.AnubisCircuitHalfOpenMaxReq,
		},
		logger,
	)

	handler := httpapi.NewHandler(leagueSvc, playerSvc, playerStatsSvc, fixtureSvc, lineupSvc, dashboardSvc, squadSvc, customLeagueSvc, logger)
	router := httpapi.NewRouter(handler, anubisClient, logger, cfg.SwaggerEnabled, cfg.CORSAllowedOrigins)

	return router, db.Close, nil
}
