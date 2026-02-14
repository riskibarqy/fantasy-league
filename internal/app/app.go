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
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/infrastructure/account/anubis"
	postgresrepo "github.com/riskibarqy/fantasy-league/internal/infrastructure/repository/postgres"
	"github.com/riskibarqy/fantasy-league/internal/interfaces/httpapi"
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

	seedCtx, seedCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer seedCancel()
	if err := postgresrepo.BootstrapSeed(seedCtx, db); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("bootstrap seed data: %w", err)
	}

	leagueRepo := postgresrepo.NewLeagueRepository(db)
	teamRepo := postgresrepo.NewTeamRepository(db)
	playerRepo := postgresrepo.NewPlayerRepository(db)
	fixtureRepo := postgresrepo.NewFixtureRepository(db)
	lineupRepo := postgresrepo.NewLineupRepository(db)
	squadRepo := postgresrepo.NewSquadRepository(db)

	leagueSvc := usecase.NewLeagueService(leagueRepo, teamRepo)
	playerSvc := usecase.NewPlayerService(leagueRepo, playerRepo)
	fixtureSvc := usecase.NewFixtureService(leagueRepo, fixtureRepo)
	lineupSvc := usecase.NewLineupService(leagueRepo, playerRepo, lineupRepo)
	dashboardSvc := usecase.NewDashboardService(leagueRepo, playerRepo, fixtureRepo, lineupRepo)
	squadSvc := usecase.NewSquadService(
		leagueRepo,
		playerRepo,
		squadRepo,
		fantasy.DefaultRules(),
		idgen.NewRandomGenerator(),
		logger,
	)

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

	handler := httpapi.NewHandler(leagueSvc, playerSvc, fixtureSvc, lineupSvc, dashboardSvc, squadSvc, logger)
	router := httpapi.NewRouter(handler, anubisClient, logger, cfg.SwaggerEnabled, cfg.CORSAllowedOrigins)

	return router, db.Close, nil
}
