package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/app"
	"github.com/riskibarqy/fantasy-league/internal/config"
	"github.com/riskibarqy/fantasy-league/internal/observability"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	shutdownTelemetry, err := observability.InitUptrace(cfg, logger)
	if err != nil {
		logger.Error("init observability", "error", err)
		os.Exit(1)
	}
	stopPyroscope, err := observability.InitPyroscope(cfg, logger)
	if err != nil {
		logger.Error("init pyroscope", "error", err)
		os.Exit(1)
	}
	pprofServer, err := observability.StartPprofServer(cfg, logger)
	if err != nil {
		logger.Error("start pprof server", "error", err)
		os.Exit(1)
	}

	httpHandler, closeDB, err := app.NewHTTPHandler(cfg, logger)
	if err != nil {
		logger.Error("build app", "error", err)
		os.Exit(1)
	}

	server := &fasthttp.Server{
		Name:         cfg.ServiceName,
		Handler:      fasthttpadaptor.NewFastHTTPHandler(httpHandler),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	var shuttingDown atomic.Bool
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("http server starting", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(cfg.HTTPAddr); err != nil && !shuttingDown.Load() {
			serverErr <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		logger.Error("http server failed", "error", err)
		os.Exit(1)
	}

	shuttingDown.Store(true)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	if err := observability.StopPprofServer(pprofServer, logger, 5*time.Second); err != nil {
		logger.Error("shutdown pprof server failed", "error", err)
		os.Exit(1)
	}
	if err := stopPyroscope(); err != nil {
		logger.Error("shutdown pyroscope failed", "error", err)
		os.Exit(1)
	}
	if err := shutdownTelemetry(shutdownCtx); err != nil {
		logger.Error("shutdown observability failed", "error", err)
		os.Exit(1)
	}
	if err := closeDB(); err != nil {
		logger.Error("close database failed", "error", err)
		os.Exit(1)
	}

	logger.Info("http server stopped")
}
