package anubis

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func TestClientVerifyAccessToken_SendsAdminKeyAndParsesResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/auth/introspect" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("x-admin-key"); got != "admin-secret" {
			t.Fatalf("unexpected x-admin-key: %s", got)
		}

		var req map[string]string
		if err := jsoniter.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req["token"] != "token-abc" {
			t.Fatalf("unexpected token value: %s", req["token"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = jsoniter.NewEncoder(w).Encode(map[string]any{
			"active":      true,
			"user_id":     "user-123",
			"app_id":      "app-001",
			"roles":       []string{"viewer"},
			"permissions": []string{"users.read"},
			"exp":         1730000000,
			"iat":         1729990000,
			"jti":         "jti-001",
		})
	}))
	defer srv.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewClient(
		srv.Client(),
		srv.URL,
		"/v1/auth/introspect",
		"admin-secret",
		CircuitBreakerConfig{Enabled: false},
		logger,
	)

	principal, err := client.VerifyAccessToken(context.Background(), "token-abc")
	if err != nil {
		t.Fatalf("verify token failed: %v", err)
	}

	if principal.UserID != "user-123" {
		t.Fatalf("unexpected user id: %s", principal.UserID)
	}
	if principal.Email != "" {
		t.Fatalf("expected empty email, got %s", principal.Email)
	}
}

func TestClientVerifyAccessToken_InactiveToken(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = jsoniter.NewEncoder(w).Encode(map[string]any{"active": false})
	}))
	defer srv.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewClient(
		srv.Client(),
		srv.URL,
		"/v1/auth/introspect",
		"admin-secret",
		CircuitBreakerConfig{Enabled: false},
		logger,
	)

	_, err := client.VerifyAccessToken(context.Background(), "invalid-token")
	if !errors.Is(err, usecase.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestClientVerifyAccessToken_ForbiddenMappedToDependencyUnavailable(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer srv.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewClient(
		srv.Client(),
		srv.URL,
		"/v1/auth/introspect",
		"wrong-key",
		CircuitBreakerConfig{Enabled: false},
		logger,
	)

	_, err := client.VerifyAccessToken(context.Background(), "token-abc")
	if !errors.Is(err, usecase.ErrDependencyUnavailable) {
		t.Fatalf("expected ErrDependencyUnavailable, got %v", err)
	}
}

func TestClientVerifyAccessToken_UsesInMemoryCache(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = jsoniter.NewEncoder(w).Encode(map[string]any{
			"active":  true,
			"user_id": "user-cache",
		})
	}))
	defer srv.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewClient(
		srv.Client(),
		srv.URL,
		"/v1/auth/introspect",
		"admin-secret",
		CircuitBreakerConfig{Enabled: false},
		logger,
	)

	for i := 0; i < 2; i++ {
		principal, err := client.VerifyAccessToken(context.Background(), "cached-token")
		if err != nil {
			t.Fatalf("verify token failed: %v", err)
		}
		if principal.UserID != "user-cache" {
			t.Fatalf("unexpected user id: %s", principal.UserID)
		}
	}

	if calls.Load() != 1 {
		t.Fatalf("expected one introspection call with cache, got %d", calls.Load())
	}
}
