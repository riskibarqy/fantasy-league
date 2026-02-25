package usecase

import crerr "github.com/cockroachdb/errors"

var (
	ErrInvalidInput          = crerr.New("invalid input")
	ErrNotFound              = crerr.New("resource not found")
	ErrUnauthorized          = crerr.New("unauthorized")
	ErrDependencyUnavailable = crerr.New("dependency unavailable")
)
