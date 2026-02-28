package observability

import (
	"context"
	"github.com/riskibarqy/fantasy-league/internal/platform/logging"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/config"
	"github.com/uptrace/uptrace-go/uptrace"
)

// InitUptrace configures global OpenTelemetry providers for Uptrace.
func InitUptrace(cfg config.Config, logger *logging.Logger) (func(context.Context) error, error) {
	if logger == nil {
		logger = logging.Default()
	}

	if !cfg.UptraceEnabled {
		logging.SetMirror(nil)
		logger.Info("uptrace disabled", "reason", "UPTRACE_ENABLED=false")
		return func(context.Context) error { return nil }, nil
	}

	if strings.TrimSpace(cfg.UptraceDSN) == "" {
		logging.SetMirror(nil)
		logger.Info("uptrace disabled", "reason", "UPTRACE_DSN empty")
		return func(context.Context) error { return nil }, nil
	}

	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN(cfg.UptraceDSN),
		uptrace.WithServiceName(cfg.ServiceName),
		uptrace.WithServiceVersion(cfg.ServiceVersion),
		uptrace.WithDeploymentEnvironment(cfg.AppEnv),
		uptrace.WithLoggingEnabled(cfg.UptraceLogsEnabled),
	)
	if cfg.UptraceLogsEnabled {
		logging.SetMirror(newUptraceLogMirror(cfg.ServiceVersion))
	} else {
		logging.SetMirror(nil)
	}

	logger.Info("uptrace enabled",
		"service_name", cfg.ServiceName,
		"service_version", cfg.ServiceVersion,
		"environment", cfg.AppEnv,
		"logs_enabled", cfg.UptraceLogsEnabled,
	)

	return func(ctx context.Context) error {
		logging.SetMirror(nil)
		return uptrace.Shutdown(ctx)
	}, nil
}
