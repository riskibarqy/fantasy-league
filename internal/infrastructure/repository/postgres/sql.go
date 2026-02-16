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

func isUnnamedPreparedStatementMissing(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "unnamed prepared statement does not exist") ||
		(strings.Contains(text, "(26000)") &&
			strings.Contains(text, "prepared statement"))
}

func quoteLiteral(value string) string {
	escaped := strings.ReplaceAll(value, "'", "''")
	return "'" + escaped + "'"
}

func nullInt64ToInt64(v sql.NullInt64) int64 {
	if !v.Valid {
		return 0
	}

	return v.Int64
}

func nullInt64ToIntPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}

	value := int(v.Int64)
	return &value
}
