package resilience

import "time"

type CircuitBreakerConfig struct {
	Enabled          bool
	FailureThreshold int
	OpenTimeout      time.Duration
	HalfOpenMaxReq   int
}

func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 5,
		OpenTimeout:      15 * time.Second,
		HalfOpenMaxReq:   2,
	}
}

func NormalizeCircuitBreakerConfig(cfg CircuitBreakerConfig) CircuitBreakerConfig {
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
