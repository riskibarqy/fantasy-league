package config

import (
	"testing"
	"time"
)

func TestLoad_AppEnvValidation(t *testing.T) {
	t.Setenv("APP_ENV", "invalid")
	if _, err := Load(); err == nil {
		t.Fatalf("expected error for invalid APP_ENV")
	}
}

func TestLoad_UptraceRequiresDSNWhenEnabled(t *testing.T) {
	t.Setenv("APP_ENV", EnvDev)
	t.Setenv("UPTRACE_ENABLED", "true")
	t.Setenv("UPTRACE_DSN", "")

	if _, err := Load(); err == nil {
		t.Fatalf("expected error when UPTRACE_ENABLED=true without UPTRACE_DSN")
	}
}

func TestLoad_BetterStackRequiresEndpointWhenEnabled(t *testing.T) {
	t.Setenv("APP_ENV", EnvDev)
	t.Setenv("UPTRACE_ENABLED", "false")
	t.Setenv("BETTERSTACK_ENABLED", "true")
	t.Setenv("BETTERSTACK_ENDPOINT", "")

	if _, err := Load(); err == nil {
		t.Fatalf("expected error when BETTERSTACK_ENABLED=true without BETTERSTACK_ENDPOINT")
	}
}

func TestLoad_BetterStackConfigParsing(t *testing.T) {
	t.Setenv("APP_ENV", EnvDev)
	t.Setenv("UPTRACE_ENABLED", "false")
	t.Setenv("BETTERSTACK_ENABLED", "true")
	t.Setenv("BETTERSTACK_ENDPOINT", "s1765114.eu-fsn-3.betterstackdata.com")
	t.Setenv("BETTERSTACK_TOKEN", "token-123")
	t.Setenv("BETTERSTACK_TIMEOUT", "4s")
	t.Setenv("BETTERSTACK_MIN_LEVEL", "warn")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.BetterStackEnabled {
		t.Fatalf("expected BetterStackEnabled=true")
	}
	if cfg.BetterStackEndpoint != "s1765114.eu-fsn-3.betterstackdata.com" {
		t.Fatalf("unexpected BetterStackEndpoint: %q", cfg.BetterStackEndpoint)
	}
	if cfg.BetterStackToken != "token-123" {
		t.Fatalf("unexpected BetterStackToken")
	}
	if cfg.BetterStackTimeout != 4*time.Second {
		t.Fatalf("unexpected BetterStackTimeout: %s", cfg.BetterStackTimeout)
	}
	if cfg.BetterStackMinLevel.String() != "warn" {
		t.Fatalf("unexpected BetterStackMinLevel: %s", cfg.BetterStackMinLevel.String())
	}
}

func TestLoad_DefaultsByEnv(t *testing.T) {
	t.Run("prod disables swagger by default", func(t *testing.T) {
		t.Setenv("APP_ENV", EnvProd)
		t.Setenv("UPTRACE_ENABLED", "false")
		t.Setenv("SWAGGER_ENABLED", "")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.SwaggerEnabled {
			t.Fatalf("expected SwaggerEnabled=false in prod by default")
		}
	})

	t.Run("dev enables swagger by default", func(t *testing.T) {
		t.Setenv("APP_ENV", EnvDev)
		t.Setenv("UPTRACE_ENABLED", "false")
		t.Setenv("SWAGGER_ENABLED", "")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if !cfg.SwaggerEnabled {
			t.Fatalf("expected SwaggerEnabled=true in dev by default")
		}
	})
}

func TestLoad_PprofDefaultsAddrWhenEnabled(t *testing.T) {
	t.Setenv("APP_ENV", EnvDev)
	t.Setenv("UPTRACE_ENABLED", "false")
	t.Setenv("PPROF_ENABLED", "true")
	t.Setenv("PPROF_ADDR", "  ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.PprofAddr != ":6060" {
		t.Fatalf("expected default pprof addr :6060, got %q", cfg.PprofAddr)
	}
}

func TestLoad_PyroscopeRequiresServerAddressWhenEnabled(t *testing.T) {
	t.Setenv("APP_ENV", EnvDev)
	t.Setenv("UPTRACE_ENABLED", "false")
	t.Setenv("PYROSCOPE_ENABLED", "true")
	t.Setenv("PYROSCOPE_SERVER_ADDRESS", "")

	if _, err := Load(); err == nil {
		t.Fatalf("expected error when PYROSCOPE_ENABLED=true without PYROSCOPE_SERVER_ADDRESS")
	}
}

func TestLoad_PyroscopeAppNameDefaultsToServiceName(t *testing.T) {
	t.Setenv("APP_ENV", EnvDev)
	t.Setenv("UPTRACE_ENABLED", "false")
	t.Setenv("APP_SERVICE_NAME", "fantasy-league-api-test")
	t.Setenv("PYROSCOPE_ENABLED", "true")
	t.Setenv("PYROSCOPE_SERVER_ADDRESS", "http://localhost:4040")
	t.Setenv("PYROSCOPE_APP_NAME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.PyroscopeAppName != "fantasy-league-api-test" {
		t.Fatalf("unexpected pyroscope app name: %q", cfg.PyroscopeAppName)
	}
}

func TestLoad_CORSOriginsDefaultAndParsing(t *testing.T) {
	t.Setenv("APP_ENV", EnvDev)
	t.Setenv("UPTRACE_ENABLED", "false")

	t.Run("default wildcard", func(t *testing.T) {
		t.Setenv("CORS_ALLOWED_ORIGINS", "")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "*" {
			t.Fatalf("unexpected default CORS origins: %+v", cfg.CORSAllowedOrigins)
		}
	})

	t.Run("comma separated parsing", func(t *testing.T) {
		t.Setenv("CORS_ALLOWED_ORIGINS", " https://a.example.com, http://localhost:5173 ")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if len(cfg.CORSAllowedOrigins) != 2 {
			t.Fatalf("unexpected CORS origins length: %d", len(cfg.CORSAllowedOrigins))
		}
		if cfg.CORSAllowedOrigins[0] != "https://a.example.com" {
			t.Fatalf("unexpected first CORS origin: %s", cfg.CORSAllowedOrigins[0])
		}
		if cfg.CORSAllowedOrigins[1] != "http://localhost:5173" {
			t.Fatalf("unexpected second CORS origin: %s", cfg.CORSAllowedOrigins[1])
		}
	})
}

func TestLoad_DBDisablePreparedBinaryResultParsing(t *testing.T) {
	t.Setenv("APP_ENV", EnvDev)
	t.Setenv("UPTRACE_ENABLED", "false")

	t.Run("default true", func(t *testing.T) {
		t.Setenv("DB_DISABLE_PREPARED_BINARY_RESULT", "")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if !cfg.DBDisablePreparedBinary {
			t.Fatalf("expected DBDisablePreparedBinary=true by default")
		}
	})

	t.Run("invalid value", func(t *testing.T) {
		t.Setenv("DB_DISABLE_PREPARED_BINARY_RESULT", "not-bool")
		if _, err := Load(); err == nil {
			t.Fatalf("expected error for invalid DB_DISABLE_PREPARED_BINARY_RESULT")
		}
	})
}

func TestLoad_CacheConfigParsing(t *testing.T) {
	t.Setenv("APP_ENV", EnvDev)
	t.Setenv("UPTRACE_ENABLED", "false")

	t.Run("defaults", func(t *testing.T) {
		t.Setenv("CACHE_ENABLED", "")
		t.Setenv("CACHE_TTL", "")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if !cfg.CacheEnabled {
			t.Fatalf("expected cache enabled by default")
		}
		if cfg.CacheTTL != 60*time.Second {
			t.Fatalf("unexpected default cache ttl: %s", cfg.CacheTTL)
		}
	})

	t.Run("invalid ttl", func(t *testing.T) {
		t.Setenv("CACHE_TTL", "bad")
		if _, err := Load(); err == nil {
			t.Fatalf("expected error for invalid CACHE_TTL")
		}
	})
}

func TestLoad_QStashConfigParsing(t *testing.T) {
	t.Setenv("APP_ENV", EnvDev)
	t.Setenv("UPTRACE_ENABLED", "false")

	t.Run("disabled by default", func(t *testing.T) {
		t.Setenv("QSTASH_ENABLED", "")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.QStashEnabled {
			t.Fatalf("expected QStashEnabled=false by default")
		}
		if cfg.JobScheduleInterval != 15*time.Minute {
			t.Fatalf("unexpected default job schedule interval: %s", cfg.JobScheduleInterval)
		}
		if cfg.JobLiveInterval != 5*time.Minute {
			t.Fatalf("unexpected default job live interval: %s", cfg.JobLiveInterval)
		}
		if cfg.JobPreKickoffLead != 15*time.Minute {
			t.Fatalf("unexpected default job pre kickoff lead: %s", cfg.JobPreKickoffLead)
		}
	})

	t.Run("enabled requires token and target and internal token", func(t *testing.T) {
		t.Setenv("QSTASH_ENABLED", "true")
		t.Setenv("QSTASH_TOKEN", "")
		t.Setenv("QSTASH_TARGET_BASE_URL", "")
		t.Setenv("INTERNAL_JOB_TOKEN", "")

		if _, err := Load(); err == nil {
			t.Fatalf("expected error when QSTASH_ENABLED=true without required env")
		}
	})

	t.Run("enabled with required values", func(t *testing.T) {
		t.Setenv("QSTASH_ENABLED", "true")
		t.Setenv("QSTASH_TOKEN", "qstash-token")
		t.Setenv("QSTASH_TARGET_BASE_URL", "https://fantasy-league.fly.dev")
		t.Setenv("INTERNAL_JOB_TOKEN", "internal-job-token")
		t.Setenv("QSTASH_RETRIES", "2")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if !cfg.QStashEnabled {
			t.Fatalf("expected QStashEnabled=true")
		}
		if cfg.QStashRetries != 2 {
			t.Fatalf("unexpected qstash retries: %d", cfg.QStashRetries)
		}
		if cfg.InternalJobToken != "internal-job-token" {
			t.Fatalf("unexpected internal job token: %q", cfg.InternalJobToken)
		}
	})
}

func TestLoad_SportMonksConfigParsing(t *testing.T) {
	t.Setenv("APP_ENV", EnvDev)
	t.Setenv("UPTRACE_ENABLED", "false")

	t.Run("disabled by default", func(t *testing.T) {
		t.Setenv("SPORTMONKS_ENABLED", "")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.SportMonksEnabled {
			t.Fatalf("expected SportMonksEnabled=false by default")
		}
	})

	t.Run("enabled requires token and season map", func(t *testing.T) {
		t.Setenv("SPORTMONKS_ENABLED", "true")
		t.Setenv("SPORTMONKS_TOKEN", "")
		t.Setenv("SPORTMONKS_SEASON_ID_MAP", "")
		if _, err := Load(); err == nil {
			t.Fatalf("expected error when SPORTMONKS_ENABLED=true without token/season map")
		}
	})

	t.Run("enabled with valid values", func(t *testing.T) {
		t.Setenv("SPORTMONKS_ENABLED", "true")
		t.Setenv("SPORTMONKS_TOKEN", "token")
		t.Setenv("SPORTMONKS_SEASON_ID_MAP", "idn-liga-1-2025:25965,global-liga-1-2025:23614")
		t.Setenv("SPORTMONKS_LEAGUE_ID_MAP", "idn-liga-1-2025:123")
		t.Setenv("SPORTMONKS_TIMEOUT", "15s")
		t.Setenv("SPORTMONKS_MAX_RETRIES", "2")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if !cfg.SportMonksEnabled {
			t.Fatalf("expected SportMonksEnabled=true")
		}
		if cfg.SportMonksSeasonIDByLeague["idn-liga-1-2025"] != 25965 {
			t.Fatalf("unexpected season map value")
		}
		if cfg.SportMonksLeagueIDByLeague["idn-liga-1-2025"] != 123 {
			t.Fatalf("unexpected league map value")
		}
	})
}
