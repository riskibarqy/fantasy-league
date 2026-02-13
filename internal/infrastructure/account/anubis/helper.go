package anubis

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

func normalizeCircuitBreakerConfig(cfg CircuitBreakerConfig) CircuitBreakerConfig {
	defaults := DefaultCircuitBreakerConfig()

	if cfg.FailureThreshold < 1 {
		cfg.FailureThreshold = defaults.FailureThreshold
	}
	if cfg.OpenTimeout <= 0 {
		cfg.OpenTimeout = defaults.OpenTimeout
	}
	if cfg.HalfOpenMaxReq < 1 {
		cfg.HalfOpenMaxReq = defaults.HalfOpenMaxReq
	}

	return cfg
}

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
