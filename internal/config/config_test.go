package config

import (
	"testing"
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

func TestLoad_DefaultsByEnv(t *testing.T) {
	t.Run("prod disables swagger by default", func(t *testing.T) {
		t.Setenv("APP_ENV", EnvProd)
		t.Setenv("UPTRACE_ENABLED", "false")

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
