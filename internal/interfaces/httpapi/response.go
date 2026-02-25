package httpapi

import (
	"context"
	"errors"
	"net/http"

	sonic "github.com/bytedance/sonic"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
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
	HTTPStatus int
	Reason     string
	Status     string
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
	writeJSON(ctx, w, mapped.HTTPStatus, googleResponseEnvelope{
		APIVersion: googleAPIVersion,
		Error: &googleErrorBody{
			Code:    mapped.HTTPStatus,
			Message: err.Error(),
			Status:  mapped.Status,
			Errors: []googleErrorItem{
				{
					Domain:  errorDomain,
					Reason:  mapped.Reason,
					Message: err.Error(),
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
			HTTPStatus: http.StatusBadRequest,
			Reason:     "invalidInput",
			Status:     "INVALID_ARGUMENT",
		}
	case errors.Is(err, usecase.ErrNotFound):
		return mappedError{
			HTTPStatus: http.StatusNotFound,
			Reason:     "notFound",
			Status:     "NOT_FOUND",
		}
	case errors.Is(err, usecase.ErrUnauthorized):
		return mappedError{
			HTTPStatus: http.StatusUnauthorized,
			Reason:     "unauthorized",
			Status:     "UNAUTHENTICATED",
		}
	case errors.Is(err, usecase.ErrDependencyUnavailable):
		return mappedError{
			HTTPStatus: http.StatusServiceUnavailable,
			Reason:     "dependencyUnavailable",
			Status:     "UNAVAILABLE",
		}
	case errors.Is(err, fantasy.ErrInvalidSquadSize),
		errors.Is(err, fantasy.ErrExceededBudget),
		errors.Is(err, fantasy.ErrExceededTeamLimit),
		errors.Is(err, fantasy.ErrInsufficientFormation),
		errors.Is(err, fantasy.ErrUnknownPlayerPosition),
		errors.Is(err, fantasy.ErrDuplicatePlayerInSquad):
		return mappedError{
			HTTPStatus: http.StatusBadRequest,
			Reason:     "invalidSquad",
			Status:     "INVALID_ARGUMENT",
		}
	default:
		return mappedError{
			HTTPStatus: http.StatusInternalServerError,
			Reason:     "internalError",
			Status:     "INTERNAL",
		}
	}
}
