package anubis

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

func isCircuitFailure(err error) bool {
	return errors.Is(err, errAnubisTransient)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func buildURL(baseURL, path string) string {
	baseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	path = strings.TrimSpace(path)
	if path == "" {
		return baseURL
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return baseURL + path
}
