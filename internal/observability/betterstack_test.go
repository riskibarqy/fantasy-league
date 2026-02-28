package observability

import (
	"context"
	"github.com/riskibarqy/fantasy-league/internal/platform/logging"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/config"
)

func TestInitBetterStackLogger_SendsErrorLog(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	requestCount := 0
	var lastAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		lastAuth = r.Header.Get("Authorization")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	baseLogger := logging.NewNop()
	cfg := config.Config{
		BetterStackEnabled:  true,
		BetterStackEndpoint: server.URL,
		BetterStackToken:    "secret-token",
		BetterStackTimeout:  2 * time.Second,
		BetterStackMinLevel: logging.LevelError,
		ServiceName:         "fantasy-league-api",
		AppEnv:              config.EnvDev,
	}

	logger, shutdown, err := InitBetterStackLogger(cfg, baseLogger)
	if err != nil {
		t.Fatalf("init betterstack logger: %v", err)
	}

	logger.ErrorContext(context.Background(), "backend error", "component", "httpapi")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown logger: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if requestCount == 0 {
		t.Fatalf("expected Better Stack endpoint to receive at least 1 request")
	}
	if lastAuth != "Bearer secret-token" {
		t.Fatalf("unexpected authorization header: %q", lastAuth)
	}
}

func TestInitBetterStackLogger_RespectsMinLevel(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	baseLogger := logging.NewNop()
	cfg := config.Config{
		BetterStackEnabled:  true,
		BetterStackEndpoint: server.URL,
		BetterStackTimeout:  2 * time.Second,
		BetterStackMinLevel: logging.LevelError,
		ServiceName:         "fantasy-league-api",
		AppEnv:              config.EnvDev,
	}

	logger, shutdown, err := InitBetterStackLogger(cfg, baseLogger)
	if err != nil {
		t.Fatalf("init betterstack logger: %v", err)
	}

	logger.InfoContext(context.Background(), "info log should not be shipped")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown logger: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if requestCount != 0 {
		t.Fatalf("expected no request for info log, got %d", requestCount)
	}
}
