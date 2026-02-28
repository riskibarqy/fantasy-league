package httpapi

import (
	"context"
	"errors"
	"github.com/riskibarqy/fantasy-league/internal/platform/logging"
	"net/http"

	sonic "github.com/bytedance/sonic"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	googleAPIVersion = "2.0"
	errorDomain      = "fantasy-league"
)

type googleResponseEnvelope struct {
	APIVersion string           `json:"apiVersion"`
	Data       any              `json:"data,omitempty"`
	Error      *googleErrorBody `json:"error,omitempty"`
}

type googleErrorBody struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Status  string            `json:"status"`
	Errors  []googleErrorItem `json:"errors,omitempty"`
}

type googleErrorItem struct {
	Domain  string `json:"domain"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type mappedError struct {
	HTTPStatus    int
	Reason        string
	Status        string
	PublicMessage string
}

func writeJSON(ctx context.Context, w http.ResponseWriter, status int, payload any) {
	ctx, span := startSpan(ctx, "httpapi.writeJSON")
	defer span.End()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = sonic.ConfigDefault.NewEncoder(w).Encode(payload)
}

func writeSuccess(ctx context.Context, w http.ResponseWriter, status int, data any) {
	ctx, span := startSpan(ctx, "httpapi.writeSuccess")
	defer span.End()

	writeJSON(ctx, w, status, googleResponseEnvelope{
		APIVersion: googleAPIVersion,
		Data:       data,
	})
}

func writeError(ctx context.Context, w http.ResponseWriter, err error) {
	ctx, span := startSpan(ctx, "httpapi.writeError")
	defer span.End()

	mapped := mapError(ctx, err)
	internalMessage := err.Error()
	if internalMessage == "" {
		internalMessage = http.StatusText(mapped.HTTPStatus)
	}

	logging.Default().ErrorContext(ctx, "api error response",
		"event", "api_error",
		"error_code", mapped.Reason,
		"http_status", mapped.HTTPStatus,
		"error_status", mapped.Status,
		"user_message", mapped.PublicMessage,
		"internal_message", internalMessage,
	)

	span.RecordError(err)
	span.SetStatus(codes.Error, mapped.Reason)
	span.SetAttributes(
		attribute.Int("error.http_status", mapped.HTTPStatus),
		attribute.String("error.reason", mapped.Reason),
		attribute.String("error.status", mapped.Status),
		attribute.String("error.public_message", mapped.PublicMessage),
		attribute.String("error.internal_message", internalMessage),
	)

	writeJSON(ctx, w, mapped.HTTPStatus, googleResponseEnvelope{
		APIVersion: googleAPIVersion,
		Error: &googleErrorBody{
			Code:    mapped.HTTPStatus,
			Message: mapped.PublicMessage,
			Status:  mapped.Status,
			Errors: []googleErrorItem{
				{
					Domain:  errorDomain,
					Reason:  mapped.Reason,
					Message: mapped.PublicMessage,
				},
			},
		},
	})
}

func writeInternalError(ctx context.Context, w http.ResponseWriter) {
	ctx, span := startSpan(ctx, "httpapi.writeInternalError")
	defer span.End()

	const msg = "internal server error"

	writeJSON(ctx, w, http.StatusInternalServerError, googleResponseEnvelope{
		APIVersion: googleAPIVersion,
		Error: &googleErrorBody{
			Code:    http.StatusInternalServerError,
			Message: msg,
			Status:  "INTERNAL",
			Errors: []googleErrorItem{
				{
					Domain:  errorDomain,
					Reason:  "internalError",
					Message: msg,
				},
			},
		},
	})
}

func mapError(ctx context.Context, err error) mappedError {
	ctx, span := startSpan(ctx, "httpapi.mapError")
	defer span.End()

	switch {
	case errors.Is(err, usecase.ErrInvalidInput):
		return mappedError{
			HTTPStatus:    http.StatusBadRequest,
			Reason:        "invalidInput",
			Status:        "INVALID_ARGUMENT",
			PublicMessage: "invalid request",
		}
	case errors.Is(err, usecase.ErrNotFound):
		return mappedError{
			HTTPStatus:    http.StatusNotFound,
			Reason:        "notFound",
			Status:        "NOT_FOUND",
			PublicMessage: "resource not found",
		}
	case errors.Is(err, usecase.ErrUnauthorized):
		return mappedError{
			HTTPStatus:    http.StatusUnauthorized,
			Reason:        "unauthorized",
			Status:        "UNAUTHENTICATED",
			PublicMessage: "unauthorized",
		}
	case errors.Is(err, usecase.ErrDependencyUnavailable):
		return mappedError{
			HTTPStatus:    http.StatusServiceUnavailable,
			Reason:        "dependencyUnavailable",
			Status:        "UNAVAILABLE",
			PublicMessage: "dependency unavailable",
		}
	case errors.Is(err, fantasy.ErrInvalidSquadSize),
		errors.Is(err, fantasy.ErrExceededBudget),
		errors.Is(err, fantasy.ErrExceededTeamLimit),
		errors.Is(err, fantasy.ErrInsufficientFormation),
		errors.Is(err, fantasy.ErrUnknownPlayerPosition),
		errors.Is(err, fantasy.ErrDuplicatePlayerInSquad):
		return mappedError{
			HTTPStatus:    http.StatusBadRequest,
			Reason:        "invalidSquad",
			Status:        "INVALID_ARGUMENT",
			PublicMessage: "invalid squad",
		}
	default:
		return mappedError{
			HTTPStatus:    http.StatusInternalServerError,
			Reason:        "internalError",
			Status:        "INTERNAL",
			PublicMessage: "internal server error",
		}
	}
}
