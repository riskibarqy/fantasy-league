package app

import "net/url"

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
