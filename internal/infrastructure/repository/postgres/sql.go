package postgres

import (
	"database/sql"
	"strconv"
	"strings"
	"time"
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

func nullTimeToTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}

	value := v.Time
	return &value
}

func timeToUnix(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UTC().Unix()
}

func unixToTime(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.Unix(value, 0).UTC()
}

func nullableUnix(value *time.Time) *int64 {
	if value == nil || value.IsZero() {
		return nil
	}
	v := value.UTC().Unix()
	return &v
}

func nullUnixToTimePtr(value sql.NullInt64) *time.Time {
	if !value.Valid || value.Int64 <= 0 {
		return nil
	}
	v := time.Unix(value.Int64, 0).UTC()
	return &v
}

func nullStringToInt64(v sql.NullString) int64 {
	if !v.Valid {
		return 0
	}

	text := strings.TrimSpace(v.String)
	if text == "" {
		return 0
	}

	parsed, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0
	}

	return parsed
}
