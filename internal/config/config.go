package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Config stores runtime configuration for the service.
type Config struct {
	HTTPAddr            string
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	AnubisBaseURL       string
	AnubisIntrospectURL string
	AnubisTimeout       time.Duration
	LogLevel            slog.Level
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:            getEnv("APP_HTTP_ADDR", ":8080"),
		AnubisBaseURL:       getEnv("ANUBIS_BASE_URL", "http://localhost:8081"),
		AnubisIntrospectURL: getEnv("ANUBIS_INTROSPECT_PATH", "/v1/auth/introspect"),
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

	logLevel := parseLogLevel(getEnv("APP_LOG_LEVEL", "info"))

	cfg.ReadTimeout = readTimeout
	cfg.WriteTimeout = writeTimeout
	cfg.AnubisTimeout = anubisTimeout
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
