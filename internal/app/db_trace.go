package app

import (
	"regexp"
	"strings"
)

const maxTracedQueryLength = 512

var queryWhitespaceRegex = regexp.MustCompile(`\s+`)

func formatDBQueryForTrace(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return query
	}

	normalized := queryWhitespaceRegex.ReplaceAllString(query, " ")
	if len(normalized) <= maxTracedQueryLength {
		return normalized
	}

	return normalized[:maxTracedQueryLength] + "..."
}
