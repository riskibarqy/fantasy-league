package observability

import (
	"context"
	"log/slog"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/config"
	"github.com/uptrace/uptrace-go/uptrace"
)

// InitUptrace configures global OpenTelemetry providers for Uptrace.
func InitUptrace(cfg config.Config, logger *slog.Logger) (func(context.Context) error, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if !cfg.UptraceEnabled {
		logger.Info("uptrace disabled", "reason", "UPTRACE_ENABLED=false")
		return func(context.Context) error { return nil }, nil
	}

	if strings.TrimSpace(cfg.UptraceDSN) == "" {
		logger.Info("uptrace disabled", "reason", "UPTRACE_DSN empty")
		return func(context.Context) error { return nil }, nil
	}

	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN(cfg.UptraceDSN),
		uptrace.WithServiceName(cfg.ServiceName),
		uptrace.WithServiceVersion(cfg.ServiceVersion),
		uptrace.WithDeploymentEnvironment(cfg.AppEnv),
	)

	logger.Info("uptrace enabled",
		"service_name", cfg.ServiceName,
		"service_version", cfg.ServiceVersion,
		"environment", cfg.AppEnv,
	)

	return uptrace.Shutdown, nil
}
