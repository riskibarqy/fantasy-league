package anubis

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/riskibarqy/fantasy-league/internal/domain/user"
	"github.com/riskibarqy/fantasy-league/internal/platform/resilience"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

var errAnubisTransient = errors.New("anubis transient failure")

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

type Client struct {
	httpClient     *http.Client
	introspectURL  string
	adminKey       string
	logger         *slog.Logger
	cache          *inMemoryPrincipalCache
	breaker        *resilience.CircuitBreaker
	circuitEnabled bool
	flightGroup    resilience.SingleFlight
}

func NewClient(
	httpClient *http.Client,
	baseURL, introspectPath string,
	adminKey string,
	breakerCfg CircuitBreakerConfig,
	logger *slog.Logger,
) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	if httpClient == nil {
		httpClient = &http.Client{}
	}

	breakerCfg = normalizeCircuitBreakerConfig(breakerCfg)

	return &Client{
		httpClient:     httpClient,
		introspectURL:  buildURL(baseURL, introspectPath),
		adminKey:       strings.TrimSpace(adminKey),
		logger:         logger,
		cache:          newInMemoryPrincipalCache(30*time.Second, 10_000),
		breaker:        resilience.NewCircuitBreaker(breakerCfg.FailureThreshold, breakerCfg.OpenTimeout, breakerCfg.HalfOpenMaxReq),
		circuitEnabled: breakerCfg.Enabled,
	}
}

func (c *Client) VerifyAccessToken(ctx context.Context, token string) (user.Principal, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return user.Principal{}, fmt.Errorf("%w: token is required", usecase.ErrUnauthorized)
	}
	tokenKey := hashToken(token)
	if principal, ok := c.cache.Get(tokenKey); ok {
		return principal, nil
	}

	if c.circuitEnabled {
		if err := c.breaker.Allow(); err != nil {
			c.logger.WarnContext(ctx, "anubis circuit breaker rejected request",
				"state", c.breaker.State(),
			)
			return user.Principal{}, fmt.Errorf("%w: auth service is temporarily unavailable", usecase.ErrDependencyUnavailable)
		}
	}

	resultCh := make(chan verifyResult, 1)
	requestCtx := context.WithoutCancel(ctx)

	go func() {
		value, err, _ := c.flightGroup.Do(tokenKey, func() (any, error) {
			if principal, ok := c.cache.Get(tokenKey); ok {
				return principal, nil
			}

			principal, callErr := c.verifyByHTTP(requestCtx, token)
			if c.circuitEnabled {
				if callErr != nil {
					if isCircuitFailure(callErr) {
						c.breaker.RecordFailure()
					} else {
						c.breaker.RecordSuccess()
					}
				} else {
					c.breaker.RecordSuccess()
				}
			}
			if callErr == nil {
				c.cache.Set(tokenKey, principal)
			}

			return principal, callErr
		})
		resultCh <- verifyResult{value: value, err: err}
	}()

	select {
	case <-ctx.Done():
		return user.Principal{}, fmt.Errorf("verify access token canceled: %w", ctx.Err())
	case result := <-resultCh:
		if result.err != nil {
			return user.Principal{}, result.err
		}

		principal, ok := result.value.(user.Principal)
		if !ok {
			return user.Principal{}, fmt.Errorf("invalid singleflight result type for principal")
		}

		return principal, nil
	}
}

func (c *Client) verifyByHTTP(ctx context.Context, token string) (user.Principal, error) {
	payload := introspectRequest{Token: token}
	encoded, err := jsoniter.Marshal(payload)
	if err != nil {
		return user.Principal{}, fmt.Errorf("marshal introspect request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.introspectURL, bytes.NewReader(encoded))
	if err != nil {
		return user.Principal{}, fmt.Errorf("create introspect request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.adminKey != "" {
		req.Header.Set("x-admin-key", c.adminKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return user.Principal{}, fmt.Errorf("%w: request introspection to anubis: %v", errAnubisTransient, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return user.Principal{}, fmt.Errorf("%w: introspection forbidden, verify ANUBIS_ADMIN_KEY", usecase.ErrDependencyUnavailable)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return user.Principal{}, fmt.Errorf("%w: introspection denied", usecase.ErrUnauthorized)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return user.Principal{}, fmt.Errorf("%w: read introspect response: %v", errAnubisTransient, err)
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
		c.logger.WarnContext(ctx, "anubis introspection transient failure",
			"status_code", resp.StatusCode,
		)
		return user.Principal{}, fmt.Errorf("%w: anubis introspection failed with status %d", errAnubisTransient, resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.WarnContext(ctx, "anubis introspection non-200",
			"status_code", resp.StatusCode,
		)
		return user.Principal{}, fmt.Errorf("anubis introspection failed with status %d", resp.StatusCode)
	}

	var decoded introspectResponse
	if err := jsoniter.Unmarshal(body, &decoded); err != nil {
		return user.Principal{}, fmt.Errorf("%w: unmarshal introspect response: %v", errAnubisTransient, err)
	}

	if !decoded.Active {
		return user.Principal{}, fmt.Errorf("%w: inactive token", usecase.ErrUnauthorized)
	}
	if strings.TrimSpace(decoded.UserID) == "" {
		return user.Principal{}, fmt.Errorf("%w: invalid introspect response: user_id is empty", errAnubisTransient)
	}

	return user.Principal{
		UserID: decoded.UserID,
		Email:  "",
	}, nil
}

type introspectRequest struct {
	Token string `json:"token"`
}

type introspectResponse struct {
	Active      bool     `json:"active"`
	UserID      string   `json:"user_id"`
	AppID       string   `json:"app_id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	Exp         uint64   `json:"exp"`
	Iat         uint64   `json:"iat"`
	JTI         string   `json:"jti"`
}

type verifyResult struct {
	value any
	err   error
}
