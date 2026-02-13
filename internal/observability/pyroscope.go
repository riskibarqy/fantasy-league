package observability

import (
	"log/slog"

	"github.com/grafana/pyroscope-go"
	"github.com/riskibarqy/fantasy-league/internal/config"
)

// InitPyroscope starts continuous profiling when enabled.
func InitPyroscope(cfg config.Config, logger *slog.Logger) (func() error, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if !cfg.PyroscopeEnabled {
		logger.Info("pyroscope disabled", "reason", "PYROSCOPE_ENABLED=false")
		return func() error { return nil }, nil
	}

	profiler, err := pyroscope.Start(pyroscope.Config{
		ApplicationName:   cfg.PyroscopeAppName,
		ServerAddress:     cfg.PyroscopeServerAddress,
		AuthToken:         cfg.PyroscopeAuthToken,
		BasicAuthUser:     cfg.PyroscopeBasicAuthUser,
		BasicAuthPassword: cfg.PyroscopeBasicAuthPassword,
		UploadRate:        cfg.PyroscopeUploadRate,
		Tags: map[string]string{
			"env":     cfg.AppEnv,
			"service": cfg.ServiceName,
		},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		return nil, err
	}

	logger.Info("pyroscope enabled",
		"server_address", cfg.PyroscopeServerAddress,
		"application", cfg.PyroscopeAppName,
	)

	return profiler.Stop, nil
}
