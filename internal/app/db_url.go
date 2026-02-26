package app

import (
	"net/url"
	"strings"
)

func normalizeDBURL(raw string, disablePreparedBinaryResult bool) string {
	if !disablePreparedBinaryResult {
		return raw
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil {
		return raw
	}

	query := parsed.Query()
	if query.Get("disable_prepared_binary_result") == "" {
		query.Set("disable_prepared_binary_result", "yes")
		parsed.RawQuery = query.Encode()
	}

	return parsed.String()
}

func dbNameFromURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	parsed, err := url.Parse(trimmed)
	if err == nil && parsed != nil && parsed.Scheme != "" {
		name := strings.TrimSpace(strings.TrimPrefix(parsed.Path, "/"))
		if name != "" {
			return name
		}
	}

	for _, token := range strings.Fields(trimmed) {
		if !strings.HasPrefix(token, "dbname=") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(token, "dbname="))
		name = strings.Trim(name, `"'`)
		if name != "" {
			return name
		}
	}

	return ""
}
