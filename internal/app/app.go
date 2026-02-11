package app

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/riskibarqy/fantasy-league/internal/config"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/infrastructure/account/anubis"
	"github.com/riskibarqy/fantasy-league/internal/infrastructure/repository/memory"
	"github.com/riskibarqy/fantasy-league/internal/interfaces/httpapi"
	idgen "github.com/riskibarqy/fantasy-league/internal/platform/id"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func NewHTTPServer(cfg config.Config, logger *slog.Logger) (*http.Server, error) {
	leagueRepo := memory.NewLeagueRepository(memory.SeedLeagues())
	teamRepo := memory.NewTeamRepository(memory.SeedTeams())
	playerRepo := memory.NewPlayerRepository(memory.SeedPlayers())
	squadRepo := memory.NewSquadRepository()

	leagueSvc := usecase.NewLeagueService(leagueRepo, teamRepo)
	playerSvc := usecase.NewPlayerService(leagueRepo, playerRepo)
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
		logger,
	)

	handler := httpapi.NewHandler(leagueSvc, playerSvc, squadSvc, logger)
	router := httpapi.NewRouter(handler, anubisClient, logger)

	server := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	if server.Addr == "" {
		return nil, fmt.Errorf("http server addr cannot be empty")
	}

	return server, nil
}
