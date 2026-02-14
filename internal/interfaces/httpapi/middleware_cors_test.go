package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_AllowsConfiguredOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CORS([]string{"https://fantasy-league-fe.vercel.app"}, next)

	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard", nil)
	req.Header.Set("Origin", "https://fantasy-league-fe.vercel.app")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://fantasy-league-fe.vercel.app" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
}

func TestCORS_OptionsPreflight(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CORS([]string{"*"}, next)

	req := httptest.NewRequest(http.MethodOptions, "/v1/dashboard", nil)
	req.Header.Set("Origin", "https://fantasy-league-fe.vercel.app")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
}

func TestCORS_DisallowsUnconfiguredOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CORS([]string{"https://allowed.example.com"}, next)

	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard", nil)
	req.Header.Set("Origin", "https://not-allowed.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected empty Access-Control-Allow-Origin, got %q", got)
	}
}
