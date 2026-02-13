package usecase

import "errors"

var (
	ErrInvalidInput          = errors.New("invalid input")
	ErrNotFound              = errors.New("resource not found")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrDependencyUnavailable = errors.New("dependency unavailable")
)
