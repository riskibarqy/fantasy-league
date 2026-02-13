package observability

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/config"
)

func StartPprofServer(cfg config.Config, logger *slog.Logger) (*http.Server, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if !cfg.PprofEnabled {
		logger.Info("pprof disabled", "reason", "PPROF_ENABLED=false")
		return nil, nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	srv := &http.Server{
		Addr:              cfg.PprofAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("pprof server starting", "addr", cfg.PprofAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("pprof server failed", "error", err)
		}
	}()

	return srv, nil
}

func StopPprofServer(srv *http.Server, logger *slog.Logger, timeout time.Duration) error {
	if srv == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}
	logger.Info("pprof server stopped")

	return nil
}
