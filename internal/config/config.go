package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config stores runtime configuration for the service.
type Config struct {
	AppEnv                      string
	ServiceName                 string
	ServiceVersion              string
	HTTPAddr                    string
	DBURL                       string
	ReadTimeout                 time.Duration
	WriteTimeout                time.Duration
	PprofEnabled                bool
	PprofAddr                   string
	SwaggerEnabled              bool
	AnubisBaseURL               string
	AnubisIntrospectURL         string
	AnubisAdminKey              string
	AnubisTimeout               time.Duration
	AnubisCircuitEnabled        bool
	AnubisCircuitFailureCount   int
	AnubisCircuitOpenTimeout    time.Duration
	AnubisCircuitHalfOpenMaxReq int
	UptraceEnabled              bool
	UptraceDSN                  string
	PyroscopeEnabled            bool
	PyroscopeServerAddress      string
	PyroscopeAppName            string
	PyroscopeAuthToken          string
	PyroscopeBasicAuthUser      string
	PyroscopeBasicAuthPassword  string
	PyroscopeUploadRate         time.Duration
	LogLevel                    slog.Level
}

func Load() (Config, error) {
	appEnv, err := parseAppEnv(getEnv("APP_ENV", EnvDev))
	if err != nil {
		return Config{}, err
	}

	swaggerDefault := "true"
	if appEnv == EnvProd {
		swaggerDefault = "false"
	}

	swaggerEnabled, err := strconv.ParseBool(getEnv("SWAGGER_ENABLED", swaggerDefault))
	if err != nil {
		return Config{}, fmt.Errorf("parse SWAGGER_ENABLED: %w", err)
	}

	uptraceEnabled, err := strconv.ParseBool(getEnv("UPTRACE_ENABLED", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("parse UPTRACE_ENABLED: %w", err)
	}

	uptraceDSN := strings.TrimSpace(getEnv("UPTRACE_DSN", ""))
	if uptraceEnabled && uptraceDSN == "" {
		return Config{}, fmt.Errorf("UPTRACE_DSN is required when UPTRACE_ENABLED=true")
	}

	pprofEnabled, err := strconv.ParseBool(getEnv("PPROF_ENABLED", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("parse PPROF_ENABLED: %w", err)
	}
	pprofAddr := strings.TrimSpace(getEnv("PPROF_ADDR", ":6060"))
	if pprofEnabled && pprofAddr == "" {
		return Config{}, fmt.Errorf("PPROF_ADDR is required when PPROF_ENABLED=true")
	}

	pyroscopeEnabled, err := strconv.ParseBool(getEnv("PYROSCOPE_ENABLED", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("parse PYROSCOPE_ENABLED: %w", err)
	}
	pyroscopeServerAddress := strings.TrimSpace(getEnv("PYROSCOPE_SERVER_ADDRESS", ""))
	if pyroscopeEnabled && pyroscopeServerAddress == "" {
		return Config{}, fmt.Errorf("PYROSCOPE_SERVER_ADDRESS is required when PYROSCOPE_ENABLED=true")
	}
	pyroscopeUploadRate, err := time.ParseDuration(getEnv("PYROSCOPE_UPLOAD_RATE", "15s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse PYROSCOPE_UPLOAD_RATE: %w", err)
	}
	if pyroscopeUploadRate <= 0 {
		return Config{}, fmt.Errorf("PYROSCOPE_UPLOAD_RATE must be > 0")
	}

	cfg := Config{
		AppEnv:                     appEnv,
		ServiceName:                getEnv("APP_SERVICE_NAME", "fantasy-league-api"),
		ServiceVersion:             getEnv("APP_SERVICE_VERSION", "dev"),
		HTTPAddr:                   getEnv("APP_HTTP_ADDR", ":8080"),
		DBURL:                      getEnv("DB_URL", "postgres://postgres:postgres@localhost:5432/fantasy_league?sslmode=disable"),
		PprofEnabled:               pprofEnabled,
		PprofAddr:                  pprofAddr,
		SwaggerEnabled:             swaggerEnabled,
		AnubisBaseURL:              getEnv("ANUBIS_BASE_URL", "http://localhost:8081"),
		AnubisIntrospectURL:        getEnv("ANUBIS_INTROSPECT_PATH", "/v1/auth/introspect"),
		AnubisAdminKey:             getEnv("ANUBIS_ADMIN_KEY", ""),
		UptraceEnabled:             uptraceEnabled,
		UptraceDSN:                 uptraceDSN,
		PyroscopeEnabled:           pyroscopeEnabled,
		PyroscopeServerAddress:     pyroscopeServerAddress,
		PyroscopeAuthToken:         strings.TrimSpace(getEnv("PYROSCOPE_AUTH_TOKEN", "")),
		PyroscopeBasicAuthUser:     strings.TrimSpace(getEnv("PYROSCOPE_BASIC_AUTH_USER", "")),
		PyroscopeBasicAuthPassword: strings.TrimSpace(getEnv("PYROSCOPE_BASIC_AUTH_PASSWORD", "")),
		PyroscopeUploadRate:        pyroscopeUploadRate,
	}
	cfg.PyroscopeAppName = strings.TrimSpace(getEnv("PYROSCOPE_APP_NAME", cfg.ServiceName))
	if cfg.PyroscopeEnabled && cfg.PyroscopeAppName == "" {
		return Config{}, fmt.Errorf("PYROSCOPE_APP_NAME cannot be empty when PYROSCOPE_ENABLED=true")
	}

	readTimeout, err := time.ParseDuration(getEnv("APP_READ_TIMEOUT", "10s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse APP_READ_TIMEOUT: %w", err)
	}

	writeTimeout, err := time.ParseDuration(getEnv("APP_WRITE_TIMEOUT", "15s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse APP_WRITE_TIMEOUT: %w", err)
	}

	anubisTimeout, err := time.ParseDuration(getEnv("ANUBIS_TIMEOUT", "3s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse ANUBIS_TIMEOUT: %w", err)
	}

	anubisCircuitEnabled, err := strconv.ParseBool(getEnv("ANUBIS_CIRCUIT_ENABLED", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("parse ANUBIS_CIRCUIT_ENABLED: %w", err)
	}

	anubisCircuitFailureCount, err := getEnvAsInt("ANUBIS_CIRCUIT_FAILURE_COUNT", 5)
	if err != nil {
		return Config{}, fmt.Errorf("parse ANUBIS_CIRCUIT_FAILURE_COUNT: %w", err)
	}
	if anubisCircuitFailureCount < 1 {
		return Config{}, fmt.Errorf("ANUBIS_CIRCUIT_FAILURE_COUNT must be >= 1")
	}

	anubisCircuitOpenTimeout, err := time.ParseDuration(getEnv("ANUBIS_CIRCUIT_OPEN_TIMEOUT", "15s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse ANUBIS_CIRCUIT_OPEN_TIMEOUT: %w", err)
	}
	if anubisCircuitOpenTimeout <= 0 {
		return Config{}, fmt.Errorf("ANUBIS_CIRCUIT_OPEN_TIMEOUT must be > 0")
	}

	anubisCircuitHalfOpenMaxReq, err := getEnvAsInt("ANUBIS_CIRCUIT_HALF_OPEN_MAX_REQ", 2)
	if err != nil {
		return Config{}, fmt.Errorf("parse ANUBIS_CIRCUIT_HALF_OPEN_MAX_REQ: %w", err)
	}
	if anubisCircuitHalfOpenMaxReq < 1 {
		return Config{}, fmt.Errorf("ANUBIS_CIRCUIT_HALF_OPEN_MAX_REQ must be >= 1")
	}

	logLevel := parseLogLevel(getEnv("APP_LOG_LEVEL", "info"))

	cfg.ReadTimeout = readTimeout
	cfg.WriteTimeout = writeTimeout
	cfg.AnubisTimeout = anubisTimeout
	cfg.AnubisCircuitEnabled = anubisCircuitEnabled
	cfg.AnubisCircuitFailureCount = anubisCircuitFailureCount
	cfg.AnubisCircuitOpenTimeout = anubisCircuitOpenTimeout
	cfg.AnubisCircuitHalfOpenMaxReq = anubisCircuitHalfOpenMaxReq
	cfg.LogLevel = logLevel

	return cfg, nil
}

func parseLogLevel(v string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

func getEnvAsInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	out, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return out, nil
}

const (
	EnvDev   = "dev"
	EnvStage = "stage"
	EnvProd  = "prod"
)

func parseAppEnv(v string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(v))
	switch value {
	case EnvDev, EnvStage, EnvProd:
		return value, nil
	default:
		return "", fmt.Errorf("invalid APP_ENV %q: valid values are %s, %s, %s", v, EnvDev, EnvStage, EnvProd)
	}
}
