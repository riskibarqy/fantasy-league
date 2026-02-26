package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	sonic "github.com/bytedance/sonic"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

func TestWriteSuccess_GoogleEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	writeSuccess(context.Background(), rec, http.StatusOK, map[string]string{"status": "ok"})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := sonic.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	if got, _ := body["apiVersion"].(string); got != "2.0" {
		t.Fatalf("expected apiVersion=2.0, got %v", body["apiVersion"])
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("expected data key in success response")
	}
	if _, ok := body["error"]; ok {
		t.Fatalf("did not expect error key in success response")
	}
}

func TestWriteError_GoogleEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(context.Background(), rec, fmt.Errorf("%w: bad payload", usecase.ErrInvalidInput))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var body map[string]any
	if err := sonic.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	if got, _ := body["apiVersion"].(string); got != "2.0" {
		t.Fatalf("expected apiVersion=2.0, got %v", body["apiVersion"])
	}
	errorObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object in response")
	}
	if got, _ := errorObj["status"].(string); got != "INVALID_ARGUMENT" {
		t.Fatalf("expected error status INVALID_ARGUMENT, got %v", errorObj["status"])
	}
	if got, _ := errorObj["message"].(string); got != "invalid request" {
		t.Fatalf("expected public message 'invalid request', got %v", errorObj["message"])
	}

	items, ok := errorObj["errors"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected error items in response")
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first error item to be object")
	}
	if got, _ := first["message"].(string); got != "invalid request" {
		t.Fatalf("expected error item message 'invalid request', got %v", first["message"])
	}
	if got, _ := first["reason"].(string); got != "invalidInput" {
		t.Fatalf("expected error reason invalidInput, got %v", first["reason"])
	}
}

func TestWriteError_DoesNotLeakInternalMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(context.Background(), rec, fmt.Errorf("%w: db select failed: timeout", usecase.ErrDependencyUnavailable))

	var body map[string]any
	if err := sonic.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	errorObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object in response")
	}
	if got, _ := errorObj["message"].(string); got != "dependency unavailable" {
		t.Fatalf("expected public message 'dependency unavailable', got %v", errorObj["message"])
	}
}
