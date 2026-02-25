package httpapi

import "testing"

func TestShouldTraceRequest_HealthPaths(t *testing.T) {
	paths := []string{"/healthz", "/health", "/livez", "/readyz", " /healthz "}
	for _, path := range paths {
		if shouldTraceRequest(path) {
			t.Fatalf("expected no tracing for path %q", path)
		}
	}
}

func TestShouldTraceRequest_NonHealthPaths(t *testing.T) {
	paths := []string{"/v1/dashboard", "/v1/leagues", "/", "/docs"}
	for _, path := range paths {
		if !shouldTraceRequest(path) {
			t.Fatalf("expected tracing for path %q", path)
		}
	}
}
