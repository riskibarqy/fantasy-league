package postgres

import (
	"database/sql"
	"strings"
)

func isNotFound(err error) bool {
	return err == sql.ErrNoRows
}

func isBindParameterMismatch(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "bind message supplies") &&
		strings.Contains(text, "prepared statement") &&
		strings.Contains(text, "requires")
}
