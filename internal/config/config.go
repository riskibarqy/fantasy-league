package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/platform/logging"
)

// Config stores runtime configuration for the service.
type Config struct {
	AppEnv                          string
	ServiceName                     string
	ServiceVersion                  string
	HTTPAddr                        string
	DBURL                           string
	DBDisablePreparedBinary         bool
	CacheEnabled                    bool
	CacheTTL                        time.Duration
	CORSAllowedOrigins              []string
	ReadTimeout                     time.Duration
	WriteTimeout                    time.Duration
	PprofEnabled                    bool
	PprofAddr                       string
	SwaggerEnabled                  bool
	AnubisBaseURL                   string
	AnubisIntrospectURL             string
	AnubisAdminKey                  string
	AnubisTimeout                   time.Duration
	AnubisCircuitEnabled            bool
	AnubisCircuitFailureCount       int
	AnubisCircuitOpenTimeout        time.Duration
	AnubisCircuitHalfOpenMaxReq     int
	UptraceEnabled                  bool
	UptraceDSN                      string
	UptraceLogsEnabled              bool
	UptraceCaptureRequestBody       bool
	UptraceRequestBodyMaxBytes      int
	BetterStackEnabled              bool
	BetterStackEndpoint             string
	BetterStackToken                string
	BetterStackTimeout              time.Duration
	BetterStackMinLevel             logging.Level
	PyroscopeEnabled                bool
	PyroscopeServerAddress          string
	PyroscopeAppName                string
	PyroscopeAuthToken              string
	PyroscopeBasicAuthUser          string
	PyroscopeBasicAuthPassword      string
	PyroscopeUploadRate             time.Duration
	SportMonksEnabled               bool
	SportMonksBaseURL               string
	SportMonksToken                 string
	SportMonksTimeout               time.Duration
	SportMonksMaxRetries            int
	SportMonksCircuitEnabled        bool
	SportMonksCircuitFailureCount   int
	SportMonksCircuitOpenTimeout    time.Duration
	SportMonksCircuitHalfOpenMaxReq int
	SportMonksSeasonIDByLeague      map[string]int64
	SportMonksLeagueIDByLeague      map[string]int64
	InternalJobToken                string
	QStashEnabled                   bool
	QStashBaseURL                   string
	QStashToken                     string
	QStashTargetBaseURL             string
	QStashRetries                   int
	QStashCircuitEnabled            bool
	QStashCircuitFailureCount       int
	QStashCircuitOpenTimeout        time.Duration
	QStashCircuitHalfOpenMaxReq     int
	JobScheduleInterval             time.Duration
	JobLiveInterval                 time.Duration
	JobPreKickoffLead               time.Duration
	LogLevel                        logging.Level
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
	if uptraceDSN == "" {
		uptraceDSN = parseUptraceDSNFromOTLPHeaders(getEnv("OTEL_EXPORTER_OTLP_HEADERS", ""))
	}
	if uptraceEnabled && uptraceDSN == "" {
		return Config{}, fmt.Errorf("UPTRACE_DSN is required when UPTRACE_ENABLED=true")
	}
	uptraceLogsEnabled, err := strconv.ParseBool(getEnv("UPTRACE_LOGS_ENABLED", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("parse UPTRACE_LOGS_ENABLED: %w", err)
	}
	uptraceCaptureRequestBody, err := strconv.ParseBool(getEnv("UPTRACE_CAPTURE_REQUEST_BODY", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("parse UPTRACE_CAPTURE_REQUEST_BODY: %w", err)
	}
	uptraceRequestBodyMaxBytes, err := getEnvAsInt("UPTRACE_REQUEST_BODY_MAX_BYTES", 8192)
	if err != nil {
		return Config{}, fmt.Errorf("parse UPTRACE_REQUEST_BODY_MAX_BYTES: %w", err)
	}
	if uptraceRequestBodyMaxBytes <= 0 {
		return Config{}, fmt.Errorf("UPTRACE_REQUEST_BODY_MAX_BYTES must be > 0")
	}

	betterStackEnabled, err := strconv.ParseBool(getEnv("BETTERSTACK_ENABLED", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("parse BETTERSTACK_ENABLED: %w", err)
	}
	betterStackEndpoint := strings.TrimSpace(getEnv("BETTERSTACK_ENDPOINT", ""))
	if betterStackEnabled && betterStackEndpoint == "" {
		return Config{}, fmt.Errorf("BETTERSTACK_ENDPOINT is required when BETTERSTACK_ENABLED=true")
	}
	betterStackTimeout, err := time.ParseDuration(getEnv("BETTERSTACK_TIMEOUT", "3s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse BETTERSTACK_TIMEOUT: %w", err)
	}
	if betterStackTimeout <= 0 {
		return Config{}, fmt.Errorf("BETTERSTACK_TIMEOUT must be > 0")
	}
	betterStackMinLevel := parseLogLevel(getEnv("BETTERSTACK_MIN_LEVEL", "error"))

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

	jobScheduleInterval, err := time.ParseDuration(getEnv("JOB_SCHEDULE_INTERVAL", "15m"))
	if err != nil {
		return Config{}, fmt.Errorf("parse JOB_SCHEDULE_INTERVAL: %w", err)
	}
	if jobScheduleInterval <= 0 {
		return Config{}, fmt.Errorf("JOB_SCHEDULE_INTERVAL must be > 0")
	}

	jobLiveInterval, err := time.ParseDuration(getEnv("JOB_LIVE_INTERVAL", "5m"))
	if err != nil {
		return Config{}, fmt.Errorf("parse JOB_LIVE_INTERVAL: %w", err)
	}
	if jobLiveInterval <= 0 {
		return Config{}, fmt.Errorf("JOB_LIVE_INTERVAL must be > 0")
	}

	jobPreKickoffLead, err := time.ParseDuration(getEnv("JOB_PRE_KICKOFF_LEAD", "15m"))
	if err != nil {
		return Config{}, fmt.Errorf("parse JOB_PRE_KICKOFF_LEAD: %w", err)
	}
	if jobPreKickoffLead <= 0 {
		return Config{}, fmt.Errorf("JOB_PRE_KICKOFF_LEAD must be > 0")
	}

	sportMonksEnabled, err := strconv.ParseBool(getEnv("SPORTMONKS_ENABLED", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("parse SPORTMONKS_ENABLED: %w", err)
	}
	sportMonksTimeout, err := time.ParseDuration(getEnv("SPORTMONKS_TIMEOUT", "20s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse SPORTMONKS_TIMEOUT: %w", err)
	}
	if sportMonksTimeout <= 0 {
		return Config{}, fmt.Errorf("SPORTMONKS_TIMEOUT must be > 0")
	}
	sportMonksMaxRetries, err := getEnvAsInt("SPORTMONKS_MAX_RETRIES", 1)
	if err != nil {
		return Config{}, fmt.Errorf("parse SPORTMONKS_MAX_RETRIES: %w", err)
	}
	if sportMonksMaxRetries < 0 {
		return Config{}, fmt.Errorf("SPORTMONKS_MAX_RETRIES must be >= 0")
	}
	sportMonksCircuitEnabled, err := strconv.ParseBool(getEnv("SPORTMONKS_CIRCUIT_ENABLED", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("parse SPORTMONKS_CIRCUIT_ENABLED: %w", err)
	}
	sportMonksCircuitFailureCount, err := getEnvAsInt("SPORTMONKS_CIRCUIT_FAILURE_COUNT", 5)
	if err != nil {
		return Config{}, fmt.Errorf("parse SPORTMONKS_CIRCUIT_FAILURE_COUNT: %w", err)
	}
	if sportMonksCircuitFailureCount < 1 {
		return Config{}, fmt.Errorf("SPORTMONKS_CIRCUIT_FAILURE_COUNT must be >= 1")
	}
	sportMonksCircuitOpenTimeout, err := time.ParseDuration(getEnv("SPORTMONKS_CIRCUIT_OPEN_TIMEOUT", "15s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse SPORTMONKS_CIRCUIT_OPEN_TIMEOUT: %w", err)
	}
	if sportMonksCircuitOpenTimeout <= 0 {
		return Config{}, fmt.Errorf("SPORTMONKS_CIRCUIT_OPEN_TIMEOUT must be > 0")
	}
	sportMonksCircuitHalfOpenMaxReq, err := getEnvAsInt("SPORTMONKS_CIRCUIT_HALF_OPEN_MAX_REQ", 2)
	if err != nil {
		return Config{}, fmt.Errorf("parse SPORTMONKS_CIRCUIT_HALF_OPEN_MAX_REQ: %w", err)
	}
	if sportMonksCircuitHalfOpenMaxReq < 1 {
		return Config{}, fmt.Errorf("SPORTMONKS_CIRCUIT_HALF_OPEN_MAX_REQ must be >= 1")
	}
	sportMonksBaseURL := strings.TrimSpace(getEnv("SPORTMONKS_BASE_URL", "https://api.sportmonks.com/v3/football"))
	sportMonksToken := strings.TrimSpace(getEnv("SPORTMONKS_TOKEN", ""))
	sportMonksSeasonIDByLeague, err := parseIDMap(getEnv("SPORTMONKS_SEASON_ID_MAP", ""))
	if err != nil {
		return Config{}, fmt.Errorf("parse SPORTMONKS_SEASON_ID_MAP: %w", err)
	}
	sportMonksLeagueIDByLeague, err := parseIDMap(getEnv("SPORTMONKS_LEAGUE_ID_MAP", ""))
	if err != nil {
		return Config{}, fmt.Errorf("parse SPORTMONKS_LEAGUE_ID_MAP: %w", err)
	}
	if sportMonksEnabled {
		if sportMonksToken == "" {
			return Config{}, fmt.Errorf("SPORTMONKS_TOKEN is required when SPORTMONKS_ENABLED=true")
		}
		if len(sportMonksSeasonIDByLeague) == 0 {
			return Config{}, fmt.Errorf("SPORTMONKS_SEASON_ID_MAP is required when SPORTMONKS_ENABLED=true")
		}
	}

	qstashEnabled, err := strconv.ParseBool(getEnv("QSTASH_ENABLED", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("parse QSTASH_ENABLED: %w", err)
	}
	qstashRetries, err := getEnvAsInt("QSTASH_RETRIES", 3)
	if err != nil {
		return Config{}, fmt.Errorf("parse QSTASH_RETRIES: %w", err)
	}
	if qstashRetries < 0 {
		return Config{}, fmt.Errorf("QSTASH_RETRIES must be >= 0")
	}
	qstashCircuitEnabled, err := strconv.ParseBool(getEnv("QSTASH_CIRCUIT_ENABLED", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("parse QSTASH_CIRCUIT_ENABLED: %w", err)
	}
	qstashCircuitFailureCount, err := getEnvAsInt("QSTASH_CIRCUIT_FAILURE_COUNT", 5)
	if err != nil {
		return Config{}, fmt.Errorf("parse QSTASH_CIRCUIT_FAILURE_COUNT: %w", err)
	}
	if qstashCircuitFailureCount < 1 {
		return Config{}, fmt.Errorf("QSTASH_CIRCUIT_FAILURE_COUNT must be >= 1")
	}
	qstashCircuitOpenTimeout, err := time.ParseDuration(getEnv("QSTASH_CIRCUIT_OPEN_TIMEOUT", "15s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse QSTASH_CIRCUIT_OPEN_TIMEOUT: %w", err)
	}
	if qstashCircuitOpenTimeout <= 0 {
		return Config{}, fmt.Errorf("QSTASH_CIRCUIT_OPEN_TIMEOUT must be > 0")
	}
	qstashCircuitHalfOpenMaxReq, err := getEnvAsInt("QSTASH_CIRCUIT_HALF_OPEN_MAX_REQ", 2)
	if err != nil {
		return Config{}, fmt.Errorf("parse QSTASH_CIRCUIT_HALF_OPEN_MAX_REQ: %w", err)
	}
	if qstashCircuitHalfOpenMaxReq < 1 {
		return Config{}, fmt.Errorf("QSTASH_CIRCUIT_HALF_OPEN_MAX_REQ must be >= 1")
	}
	qstashBaseURL := strings.TrimSpace(getEnv("QSTASH_BASE_URL", "https://qstash.upstash.io"))
	qstashToken := strings.TrimSpace(getEnv("QSTASH_TOKEN", ""))
	qstashTargetBaseURL := strings.TrimSpace(getEnv("QSTASH_TARGET_BASE_URL", ""))
	internalJobToken := strings.TrimSpace(getEnv("INTERNAL_JOB_TOKEN", ""))
	if qstashEnabled {
		if qstashToken == "" {
			return Config{}, fmt.Errorf("QSTASH_TOKEN is required when QSTASH_ENABLED=true")
		}
		if qstashTargetBaseURL == "" {
			return Config{}, fmt.Errorf("QSTASH_TARGET_BASE_URL is required when QSTASH_ENABLED=true")
		}
		if internalJobToken == "" {
			return Config{}, fmt.Errorf("INTERNAL_JOB_TOKEN is required when QSTASH_ENABLED=true")
		}
	}

	cfg := Config{
		AppEnv:                          appEnv,
		ServiceName:                     getEnv("APP_SERVICE_NAME", "fantasy-league-api"),
		ServiceVersion:                  getEnv("APP_SERVICE_VERSION", "dev"),
		HTTPAddr:                        getEnv("APP_HTTP_ADDR", ":8080"),
		DBURL:                           getEnv("DB_URL", "postgres://postgres:postgres@localhost:5432/fantasy_league?sslmode=disable"),
		DBDisablePreparedBinary:         true,
		CORSAllowedOrigins:              splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "*")),
		PprofEnabled:                    pprofEnabled,
		PprofAddr:                       pprofAddr,
		SwaggerEnabled:                  swaggerEnabled,
		AnubisBaseURL:                   getEnv("ANUBIS_BASE_URL", "http://localhost:8081"),
		AnubisIntrospectURL:             getEnv("ANUBIS_INTROSPECT_PATH", "/v1/auth/introspect"),
		AnubisAdminKey:                  getEnv("ANUBIS_ADMIN_KEY", ""),
		UptraceEnabled:                  uptraceEnabled,
		UptraceDSN:                      uptraceDSN,
		UptraceLogsEnabled:              uptraceLogsEnabled,
		UptraceCaptureRequestBody:       uptraceCaptureRequestBody,
		UptraceRequestBodyMaxBytes:      uptraceRequestBodyMaxBytes,
		BetterStackEnabled:              betterStackEnabled,
		BetterStackEndpoint:             betterStackEndpoint,
		BetterStackToken:                strings.TrimSpace(getEnv("BETTERSTACK_TOKEN", "")),
		BetterStackTimeout:              betterStackTimeout,
		BetterStackMinLevel:             betterStackMinLevel,
		PyroscopeEnabled:                pyroscopeEnabled,
		PyroscopeServerAddress:          pyroscopeServerAddress,
		PyroscopeAuthToken:              strings.TrimSpace(getEnv("PYROSCOPE_AUTH_TOKEN", "")),
		PyroscopeBasicAuthUser:          strings.TrimSpace(getEnv("PYROSCOPE_BASIC_AUTH_USER", "")),
		PyroscopeBasicAuthPassword:      strings.TrimSpace(getEnv("PYROSCOPE_BASIC_AUTH_PASSWORD", "")),
		PyroscopeUploadRate:             pyroscopeUploadRate,
		SportMonksEnabled:               sportMonksEnabled,
		SportMonksBaseURL:               sportMonksBaseURL,
		SportMonksToken:                 sportMonksToken,
		SportMonksTimeout:               sportMonksTimeout,
		SportMonksMaxRetries:            sportMonksMaxRetries,
		SportMonksCircuitEnabled:        sportMonksCircuitEnabled,
		SportMonksCircuitFailureCount:   sportMonksCircuitFailureCount,
		SportMonksCircuitOpenTimeout:    sportMonksCircuitOpenTimeout,
		SportMonksCircuitHalfOpenMaxReq: sportMonksCircuitHalfOpenMaxReq,
		SportMonksSeasonIDByLeague:      sportMonksSeasonIDByLeague,
		SportMonksLeagueIDByLeague:      sportMonksLeagueIDByLeague,
		InternalJobToken:                internalJobToken,
		QStashEnabled:                   qstashEnabled,
		QStashBaseURL:                   qstashBaseURL,
		QStashToken:                     qstashToken,
		QStashTargetBaseURL:             qstashTargetBaseURL,
		QStashRetries:                   qstashRetries,
		QStashCircuitEnabled:            qstashCircuitEnabled,
		QStashCircuitFailureCount:       qstashCircuitFailureCount,
		QStashCircuitOpenTimeout:        qstashCircuitOpenTimeout,
		QStashCircuitHalfOpenMaxReq:     qstashCircuitHalfOpenMaxReq,
		JobScheduleInterval:             jobScheduleInterval,
		JobLiveInterval:                 jobLiveInterval,
		JobPreKickoffLead:               jobPreKickoffLead,
	}
	cfg.PyroscopeAppName = strings.TrimSpace(getEnv("PYROSCOPE_APP_NAME", cfg.ServiceName))
	if cfg.PyroscopeEnabled && cfg.PyroscopeAppName == "" {
		return Config{}, fmt.Errorf("PYROSCOPE_APP_NAME cannot be empty when PYROSCOPE_ENABLED=true")
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		return Config{}, fmt.Errorf("CORS_ALLOWED_ORIGINS cannot be empty")
	}

	dbDisablePreparedBinary, err := strconv.ParseBool(getEnv("DB_DISABLE_PREPARED_BINARY_RESULT", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("parse DB_DISABLE_PREPARED_BINARY_RESULT: %w", err)
	}
	cfg.DBDisablePreparedBinary = dbDisablePreparedBinary

	cacheEnabled, err := strconv.ParseBool(getEnv("CACHE_ENABLED", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("parse CACHE_ENABLED: %w", err)
	}
	cacheTTL, err := time.ParseDuration(getEnv("CACHE_TTL", "60s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse CACHE_TTL: %w", err)
	}
	if cacheTTL <= 0 {
		return Config{}, fmt.Errorf("CACHE_TTL must be > 0")
	}
	cfg.CacheEnabled = cacheEnabled
	cfg.CacheTTL = cacheTTL

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

func parseLogLevel(v string) logging.Level {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "debug":
		return logging.LevelDebug
	case "warn", "warning":
		return logging.LevelWarn
	case "error":
		return logging.LevelError
	default:
		return logging.LevelInfo
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

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		out = append(out, item)
	}

	return out
}

func parseIDMap(raw string) (map[string]int64, error) {
	out := make(map[string]int64)
	parts := strings.Split(raw, ",")
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}

		segments := strings.SplitN(item, ":", 2)
		if len(segments) != 2 {
			return nil, fmt.Errorf("invalid map item %q, expected league_id:number", item)
		}

		key := strings.TrimSpace(segments[0])
		if key == "" {
			return nil, fmt.Errorf("empty league id in item %q", item)
		}
		value, err := strconv.ParseInt(strings.TrimSpace(segments[1]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number in item %q: %w", item, err)
		}
		if value <= 0 {
			return nil, fmt.Errorf("id must be > 0 in item %q", item)
		}

		out[key] = value
	}
	return out, nil
}

func parseUptraceDSNFromOTLPHeaders(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	items := strings.Split(raw, ",")
	for _, item := range items {
		parts := strings.SplitN(strings.TrimSpace(item), "=", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(parts[0]), "uptrace-dsn") {
			value := strings.TrimSpace(parts[1])
			return strings.Trim(value, "\"'")
		}
	}

	return ""
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
