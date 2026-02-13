package observability

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/riskibarqy/fantasy-league/internal/config"
)

func TestInitUptrace_Disabled(t *testing.T) {
	cfg := config.Config{
		UptraceEnabled: false,
		ServiceName:    "fantasy-league-api",
		ServiceVersion: "dev",
		AppEnv:         config.EnvDev,
	}

	shutdown, err := InitUptrace(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("init uptrace: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown uptrace: %v", err)
	}
}
