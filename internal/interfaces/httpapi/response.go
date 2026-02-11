package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	status, code := mapError(err)
	writeJSON(w, status, errorEnvelope{
		Error: errorBody{Code: code, Message: err.Error()},
	})
}

func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput):
		return http.StatusBadRequest, "INVALID_INPUT"
	case errors.Is(err, usecase.ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, usecase.ErrUnauthorized):
		return http.StatusUnauthorized, "UNAUTHORIZED"
	case errors.Is(err, fantasy.ErrInvalidSquadSize),
		errors.Is(err, fantasy.ErrExceededBudget),
		errors.Is(err, fantasy.ErrExceededTeamLimit),
		errors.Is(err, fantasy.ErrInsufficientFormation),
		errors.Is(err, fantasy.ErrUnknownPlayerPosition),
		errors.Is(err, fantasy.ErrDuplicatePlayerInSquad):
		return http.StatusBadRequest, "INVALID_SQUAD"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}
