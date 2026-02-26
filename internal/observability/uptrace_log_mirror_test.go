package observability

import (
	"testing"

	otellog "go.opentelemetry.io/otel/log"
)

func TestShouldSkipUptraceLog(t *testing.T) {
	if !shouldSkipUptraceLog("http_request", []any{"http_path", "/healthz"}) {
		t.Fatalf("expected health check log to be skipped")
	}
	if shouldSkipUptraceLog("http_request", []any{"http_path", "/v1/leagues"}) {
		t.Fatalf("did not expect non-health log to be skipped")
	}
	if shouldSkipUptraceLog("qstash publish request", []any{"http_path", "/healthz"}) {
		t.Fatalf("did not expect non-http_request event to be skipped")
	}
}

func TestBuildOTelLogAttributes(t *testing.T) {
	attrs := buildOTelLogAttributes([]any{"league_id", "idn-liga-1-2025", "attempt", 2, "payload"})
	if len(attrs) != 3 {
		t.Fatalf("expected 3 attributes, got %d", len(attrs))
	}
	if attrs[0].Key != "league_id" || attrs[0].Value.AsString() != "idn-liga-1-2025" {
		t.Fatalf("unexpected league_id attribute")
	}
	if attrs[1].Key != "attempt" || attrs[1].Value.AsInt64() != 2 {
		t.Fatalf("unexpected attempt attribute")
	}
	if attrs[2].Key != "payload" || attrs[2].Value.Kind() != otellog.KindEmpty {
		t.Fatalf("unexpected payload attribute")
	}
}

func TestToOTelLogValue_Map(t *testing.T) {
	v := toOTelLogValue(map[string]any{
		"shots": 11,
		"win":   true,
	}, 0)
	if v.Kind() != otellog.KindMap {
		t.Fatalf("expected map value, got %s", v.Kind())
	}
	items := v.AsMap()
	if len(items) != 2 {
		t.Fatalf("expected 2 map items, got %d", len(items))
	}
}
